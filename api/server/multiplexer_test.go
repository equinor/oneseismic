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
	go s1.Run("inproc://req1", "inproc://rep1", "inproc://fail1")
	go s2.Run("inproc://req2", "inproc://rep2", "inproc://fail2")

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

func makeSocket(addr string, socktype zmq.Type) *zmq.Socket {
	queue, err := zmq.NewSocket(socktype)
	if err != nil {
		msg := "Unable to create socket %s: %w"
		log.Fatalf(msg, addr, err.Error())
	}

	err = queue.Connect(addr)
	if err != nil {
		msg := "Unable to connect to %s: %w"
		log.Fatalf(msg, addr, err.Error())
	}

	return queue
}

func TestFailureCancelsProcess(t *testing.T) {
	reqaddr  := "inproc://req"  + "failure-cancels-process"
	repaddr  := "inproc://rep"  + "failure-cancels-process"
	failaddr := "inproc://fail" + "failure-cancels-process"
	session := newSessions()
	go session.Run(reqaddr, repaddr, failaddr)

	/*
	 * The queue must be connected, otherwise the Schedule() will hang until it
	 * is. Go will not compile as long as the variable is unused, so while the
	 * defer in.Close() shouldn't necessary (will be cleaned up on gc), it is
	 * to make the variable used somehow
	 */
	in := makeSocket(reqaddr, zmq.PULL)
	defer in.Close()
	fail := makeSocket(failaddr, zmq.PUSH)

	io := make([]procIO, 10)
	for i := 0; i < 10; i++ {
		job := process {
			address: "",
			pid: strconv.Itoa(i),
			request: []byte("should never arrive"),
		}
		io[i] = session.Schedule(&job)
	}

	// emulate failures from the workers
	for i := 0; i < 10; i++ {
		msg := []string {
			strconv.Itoa(i),
			strconv.Itoa(i) + " manual-failure",
		}
		_, err := fail.SendMessage(msg)
		for err == zmq.EHOSTUNREACH {
			_, err = fail.SendMessage(msg)
		}
	}

	for i, proc := range io {

		msg := <-proc.err
		expected := strconv.Itoa(i) + " manual-failure"
		if msg != expected {
			fmt := "Unexpected fail-message = '%s'; want '%s'"
			t.Errorf(fmt, msg, expected)
		}

		_, open := <-proc.err
		if open {
			t.Errorf("proc.err (pid = %d) not closed as it should be", i)
		}

		for m := range proc.out {
			fmt := "Unexpected message (%s) received - test is likely broken"
			t.Fatalf(fmt, m)
		}

		_, open = <-proc.out
		if open {
			t.Errorf("proc.out (pid = %d) not closed as it should be", i)
		}

	}
}

func TestMessagesToCancelledJobsAreDropped(t *testing.T) {
	session := newSessions()
	reqaddr  := "inproc://req"  + "cancelled-jobs-msg-dropped"
	repaddr  := "inproc://rep"  + "cancelled-jobs-msg-dropped"
	failaddr := "inproc://fail" + "cancelled-jobs-msg-dropped"
	go session.Run(reqaddr, repaddr, failaddr)

	in := makeSocket(reqaddr, zmq.PULL)
	defer in.Close()
	rep := makeSocket(repaddr, zmq.PUSH)
	fail := makeSocket(failaddr, zmq.PUSH)

	io := make([]procIO, 10)
	for i := 0; i < 10; i++ {
		job := process {
			address: "addr",
			pid: strconv.Itoa(i),
			request: []byte("late"),
		}
		io[i] = session.Schedule(&job)
	}

	/* pid 9 does not fail! */
	for i := 0; i < 9; i++ {
		msg := []string { strconv.Itoa(i), "msg-" + strconv.Itoa(i) }
		_, err := fail.SendMessage(msg)
		for err == zmq.EHOSTUNREACH {
			_, err = fail.SendMessage(msg)
		}
	}

	/*
	 * Workers send their messages *after* the failure has happened It's not
	 * the routedPartialResult because rep is a PUSH socket, not a ROUTER
	 *
	 * This test could've been more elegant if the reply-queue was exposed, and
	 * ZMQ was circumvented (as it really plays no part in what's being tested
	 * here). This is probably a good hint for refactoring.
	 */
	for i := 0; i < 10; i++ {
		partial := partialResult {
			pid: strconv.Itoa(i),
			n: 0,
			m: 1,
			payload: []byte("late"),
		}
		_, err := partial.sendZMQ(rep)
		for err == zmq.EHOSTUNREACH {
			_, err = partial.sendZMQ(rep)
		}
	}

	/* consume all processes, failed or otherwise */
	/*
	 * This test is quite racy - it wants to make sure that messages arriving
	 * after a process has failed does not crash. In that case, the output
	 * channels are already closed, and if the range-for consumes all channels
	 * before the message-to-be-dropped is scheduled, there's no way to catch
	 * it.
	 *
	 * The flaky nature of this particular test is another good clue for
	 * refactoring the sessions so that message ordering and sensitivity can be
	 * fleshed out, documented, reproduced, and tested.
	 */
	for _, proc := range io {
		for _ = range proc.err {}
		for _ = range proc.out {}
	}
}
