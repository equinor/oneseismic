package server

import (
	"strconv"
	"testing"
	"log"

	zmq "github.com/pebbe/zmq4"
	"github.com/stretchr/testify/assert"
)

/*
 * Emulate the core pipeline, but instead of producing an artifact of seismic,
 * just echo the payload. Even though the payload processing is the identity
 * function, the messages going in and coming out must be properly structured.
 */
func echoAsWorker(tasks int) {
	in, _ := zmq.NewSocket(zmq.PULL)
	in.Connect("inproc://req1")
	in.Connect("inproc://req2")

	out, _ := zmq.NewSocket(zmq.ROUTER)
	out.SetRouterMandatory(1)
	out.Connect("inproc://rep1")
	out.Connect("inproc://rep2")

	for {
		m, _ := in.RecvMessageBytes(0)
		proc := process{}
		err := proc.loadZMQ(m)

		if err != nil {
			msg := "Broken process (loadZMQ) in core emulation: %s"
			log.Fatalf(msg, err.Error())
		}

		for i := 0; i < tasks; i++ {
			partial := routedPartialResult {
				address: proc.address,
				partial: partialResult {
					pid: proc.pid,
					n: i,
					m: tasks,
					payload: proc.request,
				},
			}

			/*
			* There is an awkward race condition in this test Connect() does
			* not block, and it can happen that the source is available with
			* messages waiting before the sink. In those cases, the sink will
			* be unreachable, but the sink is an inproc queue, so host
			* unreachable is somewhat non-sensical. Just re-try sending until
			* it actually completes.
			*
			* In the presence of super bad bugs, this could lead to a difficult
			* to diagnose infinite loop
			*/
			_, err = partial.sendZMQ(out)
			for err == zmq.EHOSTUNREACH {
				_, err = partial.sendZMQ(out)
			}
		}
	}
}

func verifyCorrectReply(t *testing.T, i int, s *sessions, done chan struct{}) {
	id := strconv.Itoa(i)
	msg := []byte("message from " + id)
	job := process{address: "", pid: id, request: msg}
	io := s.Schedule(&job)

	for result := range io.out {
		assert.Equal(t, result.payload, msg)
	}
	done <- struct{}{}
}

func TestMultiplexer(t *testing.T) {
	s1 := newSessions()
	s2 := newSessions()
	go s1.Run("inproc://req1", "inproc://rep1")
	go s2.Run("inproc://req2", "inproc://rep2")

	go echoAsWorker(1)
	go echoAsWorker(2)
	go echoAsWorker(3)

	done1 := make(chan struct{})
	done2 := make(chan struct{})
	for i := 0; i < 50; i++ {
		go verifyCorrectReply(t, i, s1, done1)
		go verifyCorrectReply(t, i, s2, done2)
	}

	for i := 0; i < 50; i++ {
		<-done1
		<-done2
	}
}
