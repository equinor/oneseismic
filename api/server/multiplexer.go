package server

import (
	"github.com/kataras/golog"
	zmq "github.com/pebbe/zmq4"
)

type job struct {
	jobID   string
	request []byte
	reply   chan []byte
}

func multiplexer(jobs chan job, address string, reqNdpt string, repNdpt string) {
	req, err := zmq.NewSocket(zmq.PUSH)
	if err != nil {
		golog.Fatal(err)
	}

	req.Bind(reqNdpt)

	rep := make(chan [][]byte)
	go func() {
		r, err := zmq.NewSocket(zmq.DEALER)
		if err != nil {
			golog.Fatal(err)
		}

		r.SetIdentity(address)
		r.Bind(repNdpt)

		for {
			m, err := r.RecvMessageBytes(0)
			//TODO do not crash, send error?
			if err != nil {
				golog.Fatal(err)
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
			replyChnls[j.jobID] = j.reply
			req.Send(address, zmq.SNDMORE)
			req.Send(j.jobID, zmq.SNDMORE)
			req.SendMessage(j.request)
		}
	}
}
