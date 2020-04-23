package server

import (
	"log"
	zmq "github.com/pebbe/zmq4"
)

type job struct {
	jobId string
	request string
	reply chan string
}

func multiplexer(jobs chan job, address string, reqNdpt string, repNdpt string) {
	req, err := zmq.NewSocket(zmq.PUSH)

	if err != nil {
		log.Fatal(err)
	}

	req.Bind(reqNdpt)

	rep := make(chan []string)
	go func() {
		r, err := zmq.NewSocket(zmq.DEALER)

		if err != nil {
			log.Fatal(err)
		}

		r.SetIdentity(address)
		r.Bind(repNdpt)

		for {
			m, err := r.RecvMessage(0)

			if err != nil {
				log.Fatal(err)
			}

			rep <- m
		}
	}()

	// TODO: Clean up in case a reply never arrives?
	replyChnls := make(map[string]chan string)

	for {
		select {
		case r := <-rep:
			jobId := string(r[len(r)-2])
			msg := r[len(r)-1]
			rc := replyChnls[jobId]

			rc <- msg
			delete(replyChnls, jobId)
		case j := <-jobs:
			replyChnls[j.jobId] = j.reply
			req.Send(address, zmq.SNDMORE)
			req.Send(j.jobId, zmq.SNDMORE)
			req.SendMessage(j.request)
		}
	}
}
