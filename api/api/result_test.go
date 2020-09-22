package api

import (
	"context"
	"encoding/binary"
	"testing"
	"time"
)

type containerSuccess struct {}

func (c *containerSuccess) download(
	ctx context.Context,
	id string,
) ([]byte, *dlerror) {
	return []byte(id), nil
}

type containerFailure struct {}
func (c *containerFailure) download(
	ctx context.Context,
	id string,
) ([]byte, *dlerror) {
	return nil, downloaderror(500, id)
}

func TestSuccessWritesToChannel(t *testing.T) {
	/*
	 * Since downloadToChannel() writes to a channel it must either be
	 * buffered, or be run in a goroutine, in order for it not to deadlock
	 */
	success  := make(chan []byte, 1)
	failure  := make(chan *dlerror, 1)
	storage  := &containerSuccess{}
	identity := "blob-id"
	downloadToChannel(storage, identity, success, failure)

	select {
		case f := <-failure:
			t.Errorf("Unexpected failure - expected %v; got %v", nil, f)
		default:
			/* No failure - good */
	}

	select {
		case s := <-success:
			if string(s) != identity  {
				t.Errorf("Expected '%s'; got '%s'", identity, string(s))
			}
		default:
			t.Errorf("Did not receive success")
	}
}

func TestFailureWritesToChannel(t *testing.T) {
	success  := make(chan []byte, 1)
	failure  := make(chan *dlerror, 1)
	storage  := &containerFailure{}
	identity := "blob-id"
	downloadToChannel(storage, identity, success, failure)

	select {
		case s := <-success:
			t.Errorf("Unexpected message on chan success; got '%s'", string(s))
		default:
			/* No success - good, failure is expected */
	}

	select {
		case f := <-failure:
			if f.status != 500 {
				t.Errorf("Expected failure.status = 500; got %v", f.status)
			}
		default:
			t.Errorf("Did not receive failure failure")
	}
}

func TestCollectPrependsMsgpackArray(t *testing.T) {
	parts := 5
	success := make(chan []byte, parts)
	failure := make(chan *dlerror)
	timeout := 500 * time.Millisecond

	for i := 0; i < parts; i++ {
		success <- []byte("some-message")
	}

	result, err := collect(parts, success, failure, timeout)
	if err != nil {
		t.Fatalf("collect failed; %v", err)
	}

	// array-type, given by the msgpack spec
	if result[0] != 0xDD {
		t.Fatalf("collect() result not msgpack array; tag was %x", result[0])
	}

	length := binary.BigEndian.Uint32(result[1:5])
	if length != 5 {
		t.Errorf("Expected array of length 5; got %d", length)
	}
}

type containerSleep struct {}

func (c *containerSleep) download(
	ctx context.Context,
	id string,
) ([]byte, *dlerror) {
	time.Sleep(2 * time.Second)
	return []byte(id), nil
}

func TestTimeoutCancelsCollect(t *testing.T) {
	success := make(chan []byte)
	failure := make(chan *dlerror)
	timeout := 50 * time.Millisecond
	// Note: Nothing is ever *sent* to either success or failure, so this test
	// should always time out.
	//
	// Go runs on a pretty long timeout for channels by default, so if this
	// test is buggy then the test suite could hang. Pass -timeout=2000ms or so
	// to set a lower timeout
	_, err := collect(1, success, failure, timeout)
	if err == nil {
		t.Errorf("collect() did not time out like it should")
	}
}
