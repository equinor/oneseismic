package server

import (
	"log"

	zmq "github.com/pebbe/zmq4"
)

type job struct {
	jobID   string
	request []byte
	reply   chan []byte
}

type wire interface {
	loadZMQ(msg [][]byte)
	sendZMQ(socket *zmq.Socket) (total int, err error)
}

/*
 * The partitionRequest is the message sent from this (the session manager)
 * over ZMQ to the scheduler/job partitioner, which decides what data to
 * retrieve from storage (manifest-server in current vocabulary)
 */
type partitionRequest struct {
	// Return-address that flows through to make sure that data is returned to
	// the correct node that manages the session
	address string
	jobID string
	request []byte
}

/*
 * The make/send functions are stupid helpers to help formalise the protocol
 * for communication with other parts of oneseismic, and provide a canonical
 * way of formatting messages for both the wire (over ZMQ) and over go channels
 */
func newPartitionRequest(j *job, address string) *partitionRequest {
	return &partitionRequest {
		address: address,
		jobID: j.jobID,
		request: j.request,
	}
}

func (p *partitionRequest) sendZMQ(socket *zmq.Socket) (total int, err error) {
	return socket.SendMessage(p.address, p.jobID, p.request)
}

/*
 * Parse a partition request as it is delivered in a ZMQ multipart message.
 * While this currently has no error checking, or really does anything
 * sophisticated, it's the canonical way to obtain a partitionRequest from a
 * multipart message, and *the* go reference for what the messages from the
 * fragment/worker looks like.
 */
func (p *partitionRequest) loadZMQ(msg [][]byte) {
	p.address = string(msg[0])
	p.jobID = string(msg[1])
	p.request = msg[2]
}

func multiplexer(jobs chan job, address string, reqNdpt string, repNdpt string) {
	req, err := zmq.NewSocket(zmq.PUSH)

	if err != nil {
		log.Fatal(err)
	}

	req.Bind(reqNdpt)

	rep := make(chan [][]byte)
	go func() {
		r, err := zmq.NewSocket(zmq.DEALER)

		if err != nil {
			log.Fatal(err)
		}

		r.SetIdentity(address)
		r.Bind(repNdpt)

		for {
			m, err := r.RecvMessageBytes(0)

			if err != nil {
				log.Fatal(err)
			}

			rep <- m
		}
	}()

	// TODO: Clean up in case a reply never arrives?
	replyChnls := make(map[string]chan []byte)

	for {
		select {
		case r := <-rep:
			jobID := string(r[len(r)-2])
			msg := r[len(r)-1]
			rc := replyChnls[jobID]

			rc <- msg
			delete(replyChnls, jobID)
		case j := <-jobs:
			part := newPartitionRequest(&j, address)
			replyChnls[j.jobID] = j.reply
			part.sendZMQ(req)
		}
	}
}
