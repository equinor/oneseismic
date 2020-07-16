package server

import (
	"log"
	"fmt"

	zmq "github.com/pebbe/zmq4"
	"github.com/google/uuid"
)

type job struct {
	pid     string
	request []byte
	io      procIO
}

type wire interface {
	loadZMQ(msg [][]byte) error
	sendZMQ(socket *zmq.Socket) (total int, err error)
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

/*
 * The routedPartialRequest is a more faithful representation of what the
 * worker nodes *actually* send - they need to include a return address to the
 * node that holds the session (this program, really). However, ZMQ at some
 * point strips this address as a part of its routing protocol, and the rest of
 * the application sees the message as partialResult.
 */
type routedPartialResult struct {
	address string
	partial partialResult
}

func (p *process) sendZMQ(socket *zmq.Socket) (total int, err error) {
	return socket.SendMessage(p.address, p.pid, p.request)
}

/*
 * Parse a partition request as it is delivered in a ZMQ multipart message.
 * While this currently has no error checking, or really does anything
 * sophisticated, it's the canonical way to obtain a process from a
 * multipart message, and *the* go reference for what the messages from the
 * fragment/worker looks like.
 */
func (p *process) loadZMQ(msg [][]byte) error {
	if len(msg) != 3 {
		return fmt.Errorf("len(msg) = %d; want 3", len(msg))
	}

	p.address = string(msg[0])
	p.pid = string(msg[1])
	p.request = msg[2]
	return nil
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

func (p *routedPartialResult) loadZMQ(msg [][]byte) error {
	if len(msg) != 3 {
		return fmt.Errorf("len(msg) = %d; want 3", len(msg))
	}
	p.address = string(msg[0])
	err := p.partial.loadZMQ(msg[1:])
	if err != nil {
		return fmt.Errorf("routedPartialResult.partial: %s", err.Error())
	}
	return nil
}

func (p *routedPartialResult) sendZMQ(socket *zmq.Socket) (total int, err error) {
	return socket.SendMessage(
		p.address,
		p.partial.pid,
		fmt.Sprintf("%d/%d", p.partial.n, p.partial.m),
		p.partial.payload,
	)
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

func (s *sessions) Schedule(proc *process) procIO {
	io := procIO {
		out: make(chan partialResult),
		err: make(chan string),
	}
	s.queue <- job{
		pid: proc.pid,
		request: proc.request,
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


func (s *sessions) Run(reqNdpt string, repNdpt string) {
	req, err := zmq.NewSocket(zmq.PUSH)

	if err != nil {
		log.Fatal(err)
	}

	req.Bind(reqNdpt)

	rep := make(chan partialResult)
	go func() {
		r, err := zmq.NewSocket(zmq.DEALER)

		if err != nil {
			log.Fatal(err)
		}

		r.SetIdentity(s.identity)
		r.Bind(repNdpt)

		var partial partialResult
		for {
			m, err := r.RecvMessageBytes(0)

			if err != nil {
				log.Fatal(err)
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
				log.Fatalf("Broken partialResult (loadZMQ): %s", err.Error())
			}

			rep <- partial
		}
	}()

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
			proc := processes[r.pid]

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
