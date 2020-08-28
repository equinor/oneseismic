package server

import (
	"testing"

	"github.com/google/uuid"
	zmq "github.com/pebbe/zmq4"
	"github.com/stretchr/testify/assert"
)

/*
 * Emulate the core pipeline, but instead of producing an artifact of seismic,
 * just echo the payload. Even though the payload processing is the identity
 * function, the messages going in and coming out must be properly structured.
 */
func mockCoreEcho(req, rep string) {
	in, _ := zmq.NewSocket(zmq.PULL)
	in.Connect(req)

	out, _ := zmq.NewSocket(zmq.ROUTER)
	out.SetRouterMandatory(1)
	out.Connect(rep)

	go func() {
		for {
			bytes, _ := in.RecvMessageBytes(0)
			out.SendMessage(
				bytes[0],
				bytes[1],
				"0/1",
				bytes[2],
			)
		}
	}()
}

func TestMultiplexer(t *testing.T) {
	req := "inproc://" + uuid.New().String()
	rep := "inproc://" + uuid.New().String()
	fail := "inproc://" + uuid.New().String()

	mockCoreEcho(req, rep)

	s := newSessions()
	go s.Run(req, rep, fail)

	msg := []byte("random message")
	io := s.schedule(uuid.New().String(), msg)
	for result := range io.out {
		assert.Equal(t, result.payload, msg)
	}
}

func mockCoreFail(req, failAddr, msg string) {
	in, _ := zmq.NewSocket(zmq.PULL)
	in.Connect(req)

	f, _ := zmq.NewSocket(zmq.PUSH)
	f.Connect(failAddr)

	go func() {
		for {
			bytes, _ := in.RecvMessageBytes(0)
			f.SendMessage(
				bytes[1],
				msg,
			)
		}
	}()
}

func TestCoreFail(t *testing.T) {
	reqaddr := "inproc://" + uuid.New().String()
	repaddr := "inproc://" + uuid.New().String()
	failaddr := "inproc://" + uuid.New().String()
	session := newSessions()
	go session.Run(reqaddr, repaddr, failaddr)

	msg := "expected error message"
	mockCoreFail(reqaddr, failaddr, msg)

	io := session.schedule("pid", []byte("random message"))

	i := 0
	for f := range io.err {
		assert.Equal(t, msg, f)
		if i > 0 {
			t.Fatal("failure channel should be closed after failure")
		}
		i = i + 1
	}

	for range io.out {
		t.Fatal("failure should close out")
	}

}

// Sends incomplete output, then a failure
func mockCoreAfterOutputFail(req, rep, failAddr string) {
	in, _ := zmq.NewSocket(zmq.PULL)
	in.Connect(req)

	out, _ := zmq.NewSocket(zmq.ROUTER)
	out.SetRouterMandatory(1)
	out.Connect(rep)

	f, _ := zmq.NewSocket(zmq.PUSH)
	f.Connect(failAddr)

	go func() {
		for {
			bytes, _ := in.RecvMessageBytes(0)
			out.SendMessage(
				bytes[0],
				bytes[1],
				"0/2",
				bytes[2],
			)
			f.SendMessage(
				bytes[1],
				"a random error message",
			)
		}
	}()
}

// This test is really that both err and out will be closed on failure
// We do not really know anything else since any order of receiving and handling
// messages, both output and errors, are undefined. And should be.
// Output might
//  - never start,
//  - be interrupted
//  - complete
// If any error is sent, we can close connections etc.
// The client somehow needs to be told that it has not received all
// Errors will be logged
// Refactoring to impose artificial ordering for testing should not be done
func TestCoreAfterOutputFail(t *testing.T) {
	reqaddr := "inproc://" + uuid.New().String()
	repaddr := "inproc://" + uuid.New().String()
	failaddr := "inproc://" + uuid.New().String()
	session := newSessions()
	go session.Run(reqaddr, repaddr, failaddr)

	mockCoreAfterOutputFail(reqaddr, repaddr, failaddr)

	io := session.schedule("pid", []byte("random message"))

	// Should not hang
	for range io.err {
	}

	// Should not hang
	for range io.out {
	}
}
