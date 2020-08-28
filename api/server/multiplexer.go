package server

import (
	"fmt"

	"github.com/google/uuid"
	zmq "github.com/pebbe/zmq4"
	"github.com/rs/zerolog/log"
)

type job struct {
	pid     string
	request []byte
	io      procIO
}

/*
 * The partialResult is this model internal representation of the message
 * sent by the worker nodes when a task is done.
 *
 * The payload being an opaque blob of bytes is very useful for testing, since
 * the ZMQ and channel messaging infrastructure now does not depend on the
 * payload, and instead of structured protobuf messages we can send strings to
 * compare.
 */
type partialResult struct {
	pid     string
	m       int
	n       int
	payload []byte
}

func (p *partialResult) loadZMQ(msg [][]byte) error {
	if len(msg) != 3 {
		return fmt.Errorf("len(msg) = %d; want 3", len(msg))
	}
	p.pid = string(msg[0])
	// TODO have separate []byte for m and n
	_, err := fmt.Sscanf(string(msg[1]), "%d/%d", &p.m, &p.n)
	if err != nil {
		return fmt.Errorf("%s: %s", err.Error(), string(msg[1]))
	}
	p.payload = msg[2]
	return nil
}

type sessions struct {
	identity string
	queue    chan job
}

/*
 * Per-process I/O channels, similar to stdout/stderr
 */
type procIO struct {
	/*
	 * Callers read the results from this channel
	 */
	out chan partialResult
	/*
	 * A message on this channel means the process failed
	 */
	err chan string
}

func (s *sessions) schedule(pid string, request []byte) procIO {
	io := procIO{
		out: make(chan partialResult),
		err: make(chan string),
	}
	s.queue <- job{
		pid:     pid,
		request: request,
		io:      io,
	}
	return io
}

func newSessions() *sessions {
	return &sessions{
		identity: uuid.New().String(),
		queue:    make(chan job),
	}
}

func all(a []bool) bool {
	for _, x := range a {
		if !x {
			return false
		}
	}
	return true
}

/*
 * Pipe failures from ZMQ into the failures channel, so it can be given to the
 * right session
 */
func pipeFailures(addr string, failures chan []string) {
	r, err := zmq.NewSocket(zmq.PULL)
	if err != nil {
		log.Fatal().Err(err)
	}

	err = r.Bind(addr)
	if err != nil {
		log.Fatal().Err(err)
	}

	for {
		msg, err := r.RecvMessage(0)
		if err != nil {
			/*
			 * An error reading from the failure channel is *probably* not
			 * fatal, and can be logged & ignored. More sophisticated
			 * handling can be necessary, but don't deal with it until it
			 * becomes an issue
			 */
			log.Fatal().Err(err)
			continue
		}
		log.Trace().Msg(msg[1])
		failures <- msg
	}
}

func listenCore(identity string, repNdpt string, rep chan partialResult) {
	r, err := zmq.NewSocket(zmq.DEALER)

	if err != nil {
		log.Fatal().Err(err)
	}

	r.SetIdentity(identity)
	r.Bind(repNdpt)
	log.Debug().Msgf("Listening on core: %v", repNdpt)

	var partial partialResult
	for {
		m, err := r.RecvMessageBytes(0)
		if err != nil {
			log.Fatal().Err(err)
		}

		err = partial.loadZMQ(m)
		if err != nil {
			/*
			 * This is likely to mean a bug somewhere, so eventually:
			 *   1. drop the message and fail the request
			 *   2. log all received bytes, try to recover at least pid
			 *   3. then carry on
			 *
			 * For now, neither the experience nor infrastructure is in
			 * place for that, so just log and exit
			 */
			log.Fatal().Err(err).Msg("Broken partialResult (loadZMQ)")
		}
		log.Trace().Msgf("%v: got partial on zmq, forwarding", partial.pid)
		rep <- partial
	}
}

func (s *sessions) Run(reqNdpt string, repNdpt string, failureAddr string) {
	req, err := zmq.NewSocket(zmq.PUSH)

	if err != nil {
		log.Fatal().Err(err)
	}

	req.Bind(reqNdpt)

	rep := make(chan partialResult)
	go listenCore(s.identity, repNdpt, rep)

	failures := make(chan []string)
	go pipeFailures(failureAddr, failures)

	type procstatus struct {
		io procIO
		/*
		 * Bit-array of already-received parts for this process
		 * TODO why not just keep count? In case of duplicates?
		 * TODO If so, we have bigger issues.
		 */
		completed []bool
		errClosed bool // TODO hacky
	}

	// TODO: Clean up in case a reply never arrives?
	processes := make(map[string]*procstatus)
	for {
		select {
		case r := <-rep:
			log.Trace().Msgf("%v: received %v/%v", r.pid, r.m, r.n)
			proc, ok := processes[r.pid]
			if !ok {
				errmsg := "%s %d/%d dropped; was not in process table"
				log.Error().Msgf(errmsg, r.pid, r.m, r.n)
				break
			}
			if proc.completed == nil {
				proc.completed = make([]bool, r.n)
			}

			// Close error channel before sending to out
			// Any errors coming later will just be logged
			// That is OK as the client cannot easily read them anyway
			if !proc.errClosed {
				log.Trace().Msgf("%v: closing err", r.pid)
				close(proc.io.err)
				proc.errClosed = true
			}
			log.Trace().Msgf("%v: sending to controller", r.pid)
			proc.io.out <- r
			proc.completed[r.m] = true

			if all(proc.completed) {
				log.Trace().Msgf("%v: closing out", r.pid)
				close(proc.io.out)
				delete(processes, r.pid)
			}

		case f := <-failures:
			log.Error()
			pid := f[0]
			msg := f[1]

			proc, ok := processes[pid]
			if !ok {
				errmsg := "%v: failed (%v); not in process table. Continue and ignore."
				log.Error().Msgf(errmsg, pid, msg)
			} else {
				errmsg := "%v: failed (%v); removing '%v' from process table"
				log.Error().Msgf(errmsg, pid, msg, pid)
				close(proc.io.out)

				// Errors cannot cleanly be sent to client after streaming has started
				// Logging them here should suffice
				if !proc.errClosed {
					proc.io.err <- msg
					close(proc.io.err)
				}
				delete(processes, pid)
			}

		case j := <-s.queue:
			processes[j.pid] = &procstatus{
				io:        j.io,
				completed: nil,
				errClosed: false,
			}
			log.Trace().Msgf("%v: expecting core to listen on %v", j.pid, reqNdpt)
			req.SendMessage(s.identity, j.pid, j.request)
		}
	}
}
