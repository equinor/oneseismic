package server

import (
	"strconv"
	"testing"

	zmq "github.com/pebbe/zmq4"
	"github.com/stretchr/testify/assert"
)

func msgLoopback() {
	in, _ := zmq.NewSocket(zmq.PULL)
	in.Connect("inproc://req1")
	in.Connect("inproc://req2")

	out, _ := zmq.NewSocket(zmq.ROUTER)
	out.SetRouterMandatory(1)
	out.Connect("inproc://rep1")
	out.Connect("inproc://rep2")

	for {
		m, _ := in.RecvMessage(0)
		_, err := out.SendMessage(m)

		for err == zmq.EHOSTUNREACH {
			_, err = out.SendMessage(m)
		}
	}
}

func verifyCorrectReply(t *testing.T, i int, jobs chan job, done chan struct{}) {
	id := strconv.Itoa(i)
	repChnl := make(chan []byte)
	msg := []byte("message from " + id)
	job := job{id, msg, repChnl}
	jobs <- job

	rep := <-repChnl

	assert.Equal(t, rep, msg)
	done <- struct{}{}
}

func TestMultiplexer(t *testing.T) {
	jobs := make(chan job)
	go multiplexer(jobs, "mplx1", "inproc://req1", "inproc://rep1")
	go multiplexer(jobs, "mplx2", "inproc://req2", "inproc://rep2")

	go msgLoopback()
	go msgLoopback()
	go msgLoopback()

	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go verifyCorrectReply(t, i, jobs, done)
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}
