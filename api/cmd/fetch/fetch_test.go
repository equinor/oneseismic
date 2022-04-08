package main

import (
	"context"
	"fmt"
	"net/url"
	"testing"
)

func testurl() *url.URL {
	addr, _ := url.Parse("https://example.com")
	return addr
}

func TestCancelledDownloadPostsOnErrorChannel(t *testing.T) {
	/* 
	 * Cancel the context immediately, to emulating a failure from the process
	 * controller. This tests that the messages flow onto the right channel in
	 * the presence of cancelled sibling fetches, not the actual fetch being
	 * processed by this thread.
	 */
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fetch := newFetch(1)
	blobs := []*url.URL{testurl()}

	fq := fetch.mkqueue()
	fetch.enqueue(ctx, fq, blobs)
	close(fetch.requests)
	fetch.run()

	select {
	case <-fq.errors:
	case <-fq.fragments:
		t.Errorf("Pending message on fragments; should be message on error")
	}
}

func TestMessageOnErrorCancelsGather(t *testing.T) {
	o := fetchQueue {
		fragments: make(chan fragment, 1),
		errors:    make(chan error, 1),
	}

	ctx, cancel := context.WithCancel(context.Background())
	// directly construct the process by populating fields manually. This is to
	// avoid having to shuffle bytes about and init the C++ object, which
	// should not be used at all for this test. This is vulnerable to changes
	// in the struct layout, but such changes should probably be detected
	// compile time anyway, and this test is then easily updated.
	proc := process {
		ctx: ctx,
		cancel: cancel,
		cpp: nil,
	}

	o.errors <- fmt.Errorf("Test error")
	// Pretend that there are 2 fragments to be fetched. None will be sent, but
	// it increases the confidence that the worker loop is aborted immediately
	// rather than waiting for more data.
	proc.gather(nil, 2, o)
	select {
	case <-ctx.Done():
	default:
		t.Errorf("Expected context to be cancelled, but it is not")
	}
}

/*
 * Compare the cost of sending the (regular) payload with a smaller structure.
 * Sending blob objects as pointers is much faster, but might possibly
 * pessimize garbage collection or other parts of the program, so it must be
 * optimized with much care.
 *
 * So far it seems that the cost of sending by-value is so small that *any* gc
 * hit or other indirection will eat up the gain and more.
 *
 * BenchmarkSendTask-4             15701625                75.2 ns/op
 * BenchmarkSendSmallStruct-4      20805266                55.6 ns/op
 */
func BenchmarkSendTask(b *testing.B) {
	type payload = request
	p := payload {}
	c := make(chan payload, 100)
	for n := 0; n < b.N; n++ {
		c <- p
		<-c
	}
}

func BenchmarkSendSmallStruct(b *testing.B) {
	type payload struct {
		i int
		b []byte
		e chan error
	}
	p := payload {}
	c := make(chan payload, 100)
	for n := 0; n < b.N; n++ {
		c <- p
		<-c
	}
}
