package server

import (
	"fmt"

	zmq "github.com/pebbe/zmq4"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type job struct {
	pid     string
	request []byte
	io      procIO
}

/*
 * The process is the message sent from this (the session manager) over ZMQ.
 * The request payload is the protobuf-encoded description description of what
 * to retrieve. This is conceptually similar to the message/event that spawns
 * processes on unix systems.
 */
type process struct {
	// Return-address that flows through to make sure that data is returned to
	// the correct node that manages the session
	address string
	pid string
	request []byte
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
	pid string
	n int
	m int
	payload []byte
}

func (p *process) sendZMQ(socket *zmq.Socket) (total int, err error) {
	return socket.SendMessage(p.address, p.pid, p.request)
}

func (p *partialResult) loadZMQ(msg [][]byte) error {
	if len(msg) != 3 {
		return fmt.Errorf("len(msg) = %d; want 3", len(msg))
	}
	p.pid = string(msg[0])
	_, err := fmt.Sscanf(string(msg[1]), "%d/%d", &p.n, &p.m)
	if err != nil {
		return fmt.Errorf("%s: %s", err.Error(), string(msg[1]))
	}
	p.payload = msg[2]
	return nil
}

func (p *partialResult) sendZMQ(socket *zmq.Socket) (total int, err error) {
	return socket.SendMessage(p.pid, fmt.Sprintf("%d/%d", p.n, p.m), p.payload)
}

type sessions struct {
	identity string
	queue chan job
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
	io := procIO {
		out: make(chan partialResult),
		err: make(chan string),
	}
	s.queue <- job{
		pid: pid,
		request: request,
		io: io,
	}
	return io
}

func newSessions() *sessions {
	return &sessions{
		identity: uuid.New().String(),
		queue: make(chan job),
	}
}

func all(a []bool) bool {
	for _, x := range a {
		if (!x) {
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
		failures <- msg
	}
}

func (s *sessions) Run(reqNdpt string, repNdpt string, failureAddr string) {
	req, err := zmq.NewSocket(zmq.PUSH)

	if err != nil {
		log.Fatal().Err(err)
	}

	req.Bind(reqNdpt)

	rep := make(chan partialResult)
	go func() {
		r, err := zmq.NewSocket(zmq.DEALER)

		if err != nil {
			log.Fatal().Err(err)
		}

		r.SetIdentity(s.identity)
		r.Bind(repNdpt)

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

			rep <- partial
		}
	}()

	failures := make(chan []string)
	go pipeFailures(failureAddr, failures)

	type procstatus struct {
		io procIO
		/*
		 * Bit-array of already-received parts for this process
		 */
		completed []bool
	}

	// TODO: Clean up in case a reply never arrives?
	processes := make(map[string]*procstatus)

	for {
		select {
		case r := <-rep:
			proc, ok := processes[r.pid]
			if !ok {
				errmsg := "%s %d/%d dropped; was not in process table"
				log.Error().Msgf(errmsg, r.pid, r.n, r.m)
				break
			}

			if proc.completed == nil {
				proc.completed = make([]bool, r.m)
			}

			proc.io.out <- r
			proc.completed[r.n] = true

			if all(proc.completed) {
				close(proc.io.out)
				close(proc.io.err)
				delete(processes, r.pid)
			}

		case f := <-failures:
			pid := f[0]
			msg := f[1]

			proc, ok := processes[pid]
			/*
			 * If not in the process table, just ignore the fail command and
			 * continue
			 */
			if !ok {
				errmsg := "%s failed (%s); was not in process table"
				log.Error().Msgf(errmsg, pid, msg)
			} else {
				errmsg := "%s failed (%s); removing from process table"
				log.Error().Msgf(errmsg, pid, msg)
				/*
				 * The order of statements creates an interesting race
				 * condition:
				 *
				 * If the io.err <- msg is sent *before* the io.out is closed,
				 * callers cannot use a range-for to aggregate partial results,
				 * because the sending on io.err will block until it is read.
				 *
				 * It is not strictly necessary to close these channels, but it
				 * does enable to use of ranged-for, which makes for much more
				 * elegant assembly.
				 */
				close(proc.io.out)
				proc.io.err <- msg
				close(proc.io.err)
				delete(processes, pid)
			}

		case j := <-s.queue:
			proc := process{
				address: s.identity,
				pid: j.pid,
				request: j.request,
			}
			processes[j.pid] = &procstatus {
				io: j.io,
				completed: nil,
			}
			proc.sendZMQ(req)
		}
	}
}
