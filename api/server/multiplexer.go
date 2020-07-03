package server

import (
	"log"
	"fmt"

	zmq "github.com/pebbe/zmq4"
)

type job struct {
	jobID   string
	request []byte
	reply   chan []byte
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
	jobID string
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
	jobID string
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
	return socket.SendMessage(p.address, p.jobID, p.request)
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
	p.jobID = string(msg[1])
	p.request = msg[2]
	return nil
}

func (p *partialResult) loadZMQ(msg [][]byte) error {
	if len(msg) != 2 {
		return fmt.Errorf("len(msg) = %d; want 2", len(msg))
	}
	p.jobID = string(msg[0])
	p.payload = msg[1]
	return nil
}

func (p *partialResult) sendZMQ(socket *zmq.Socket) (total int, err error) {
	return socket.SendMessage(p.jobID, p.payload)
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
	return socket.SendMessage(p.address, p.partial.jobID, p.partial.payload)
}

func multiplexer(jobs chan job, address string, reqNdpt string, repNdpt string) {
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

		r.SetIdentity(address)
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
				 *   2. log all received bytes, try to recover at least jobID
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

	// TODO: Clean up in case a reply never arrives?
	replyChnls := make(map[string]chan []byte)

	for {
		select {
		case r := <-rep:
			rc := replyChnls[r.jobID]
			rc <- r.payload
			delete(replyChnls, r.jobID)

		case j := <-jobs:
            proc := process{
                address: address,
                jobID: j.jobID,
                request: j.request,
            }
			replyChnls[j.jobID] = j.reply
			proc.sendZMQ(req)
		}
	}
}
