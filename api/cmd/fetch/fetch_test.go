package main

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/Azure/azure-pipeline-go/pipeline"
	"github.com/Azure/azure-storage-blob-go/azblob"
	
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func testpipeline() pipeline.Pipeline {
	return azblob.NewPipeline(
		azblob.NewAnonymousCredential(),
		azblob.PipelineOptions{},
	)
}

func testurl() url.URL {
	addr, _ := url.Parse("https://example.com")
	return *addr
}

func TestCancelledDownloadErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	blob := azblob.NewBlobURL(testurl(), testpipeline())
	_, err := fetchblob(ctx, blob)
	if err == nil {
		t.Errorf("expected fetchblob() to fail; err was nil")
	}
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

	tasks     := make(chan task, 1)
	fragments := make(chan fragment, 1)
	errors    := make(chan error, 1)
	tasks <- task {
		index: 0,
		blob: azblob.NewBlobURL(testurl(), testpipeline()),
	}
	// *don't* close the tasks channel - the fetch() loop should terminate with
	// the message posted on the error channel, so keeping it open from the
	// producer side means another layer covered in test.
	// close(tasks)
	fetch(ctx, tasks, fragments, errors)

	select {
	case <-tasks:
		t.Errorf("Pending message on tasks; should be drained by fetch()")
	case <-fragments:
		t.Errorf("Pending message on fragments; should be error")
	case <-errors:
	default:
		t.Errorf("No pending messages; should be error")
	}
}

func TestMessageOnErrorCancelsGather(t *testing.T) {
	fragments := make(chan fragment, 1)
	errors    := make(chan error, 1)

	mniredis, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer mniredis.Close()

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

	errors <- fmt.Errorf("Test error")
	// Pretend that there are 2 fragments to be fetched. None will be sent, but
	// it increases the confidence that the worker loop is aborted immediately
	// rather than waiting for more data.
	cli := redis.NewClient(&redis.Options{ Addr: mniredis.Addr(), })
	proc.gather(cli, 2, fragments, errors)
	select {
	case <-ctx.Done():
	default:
		t.Errorf("Expected context to be cancelled, but it is not")
	}
}
