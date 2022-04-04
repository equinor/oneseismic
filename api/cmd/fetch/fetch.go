package main

// #cgo LDFLAGS: -loneseismic -lfmt
// #include <stdlib.h>
// #include "tasks.h"
import "C"
import "unsafe"

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/equinor/oneseismic/api/internal/message"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/go-redis/redis/v8"
)

/*
 * A handle for the process currently being worked on. The task is just the
 * parsed message as received from the scheduler.
 *
 * The raw task is stored to be passed along to C++, which in turn will parse
 * it. This means the task will be parsed twice, which simplifies some
 * interfaces greatly, and reduces the number of parameters that has to be sent
 * to C++ functions.
 */
type process struct {
	/*
	 * The part of the system-wide process, formatted as n/m where n < m. This
	 * is probably only useful for logging, and helps identify tasks and track
	 * a process through the system.
	 */
	pid  string
	part string
	/*
	 * The parsed and raw task specification, as read from the input message
	 * queue.
	 */
	task    message.Task
	rawtask []byte
	/*
	 * The azblob API uses a context to communicate status to the caller, which
	 * in turn can be shared between multiple concurrent downloads. Useful for
	 * signalling failures to cancel pending downloads.
	 */
	ctx    context.Context
	cancel context.CancelFunc
	/*
	 * A pointer to the corresponding C++ object. The go part of this program
	 * handles sessions and I/O (tokens, requests, http requests and redis
	 * writes), whereas parsing bytes, understanding geometry, and constructing
	 * results are done in C++.
	 *
	 * The C++ implementation is aware of dimensions and generalised to N
	 * dimensions, is reasonably well tested, and leverages the compiler to
	 * ensure correctness. Using the C++ library like this introduces some
	 * syntactical overhead and build complexity, but is worth it to keep
	 * geometry and maths in a language that is arguably more suited for it.
	 * This way, go handles all I/O, scheduling, and concurrency, where C++
	 * maps between bytes and oneseismic concepts.
	 */
	cpp *C.struct_proc
}

/*
 * Automation - return the pid=, proc= log line prefix to make log formatting
 * less noisy.
 *
 * This is the only function allowed to call on a process if exec() returns an
 * error.
 */
func (p *process) logpid() string {
	return fmt.Sprintf("pid=%s, part=%s", p.pid, p.part)
}

/*
 * Convert the last-set error from C++ into a go error.
 */
func (p *process) c_error() error {
	return fmt.Errorf(C.GoString(C.errmsg(p.cpp)))
}

/*
 * exec a new process from a task description. It lifts its name from the POSIX
 * API (since oneseismic uses the process metaphor), but otherwise has no
 * relation to POSIX.
 *
 * > The exec() family of functions replaces the current process image with a
 * > new process image.
 */
func exec(msg [][]byte) (*process, error) {
	ctx, cancel := context.WithCancel(context.Background())
	proc := &process {
		pid:     string(msg[0]),
		part:    string(msg[1]),
		rawtask: msg[2],
		ctx:     ctx,
		cancel:  cancel,
	}
	_, err := proc.task.Unpack(proc.rawtask)
	if err != nil {
		return nil, err
	}

	kind := C.CString(proc.task.Function)
	defer C.free(unsafe.Pointer(kind))
	proc.cpp = C.newproc(kind);
	if proc.cpp == nil {
		msg := "%s unable to new() proc of kind %s"
		return proc, fmt.Errorf(msg, proc.logpid(), proc.task.Function)
	}
	buffer := unsafe.Pointer(&proc.rawtask[0])
	length := C.int(len(proc.rawtask))
	ok := C.init(proc.cpp, buffer, length)
	if !ok {
		return proc, proc.c_error()
	}
	return proc, nil
}

/*
 * Clean up a process, i.e. call the cleanup functions for the (unmanaged) C++
 * objects and cancel the context.
 */
func (p *process) cleanup() {
	C.cleanup(p.cpp)
	p.cpp = nil
	p.cancel()
}

/*
 * Get the list of fragment IDs [1] for this process. This call is probably
 * *somewhat* expensive and shouldn't be called in a loop.
 *
 * This function is *not* thread safe, and should not be invoked from multiple
 * goroutines.
 *
 * [1] e.g. ['src/64-64-64/0-0-1.f32', 'src/64-64-64/4-1-2.f32' ...]
 */
func (p *process) fragments() []string {
	cfrags := C.fragments(p.cpp)
	if cfrags == nil {
		msg := "%s unable to get fragment IDs: %w"
		log.Fatalf(msg, p.logpid(), p.c_error())
	}

	/*
	 * There are many ways of doing this, but since go <-> C interop is
	 * apparently quite expensive, it is implemented as a single function that
	 * returns a string. This string in turn should be
	 * parsed by go, since we want to end up with a nice structure that can be
	 * looped over without really knowing that it's powered by C++. If it turns
	 * out another approach is faster then this function could be rewritten.
	 *
	 * It is assumed that the generous allocation going both in C++ and go on
	 * is faster than doing multiple FFI trips.
	 *
	 * The delimiter ';' is chosen somewhat arbitrarily, but shouldn't really
	 * ever show up in oneseismic storage. The output from C++ does not have a
	 * trailing delimiter as it would mean string.Split() adds an empty string
	 * (!!) at the end, which in turn would build invalid URLs.
	 */
	gofrags := C.GoString(cfrags)
	return strings.Split(gofrags, ";")
}

/*
 * Register a downloaded fragment. This should be called *at least once* [1]
 * for every fragment in the task set [2] before the process is finalized with
 * pack().
 *
 * [1] really exactly once, although nothing bad *should* happen if it is
 * called multiple times for the same object.
 * [2] the list of IDs given by fragments()
 */
func (p *process) add(f fragment) error {
	buffer := unsafe.Pointer(&f.chunk[0])
	length := C.int(len(f.chunk))
	index  := C.int(f.index)
	ok := C.add(p.cpp, index, buffer, length)
	if !ok {
		return p.c_error()
	} else {
		return nil
	}
}

/*
 * Pack the result into a ready-to-store message. This should be called *at
 * least once* [1], when all fragments been given to add(). The packed result
 * is ready to be written to storage as-is, and does not need to be inspected
 * or handled in any way by go.
 *
 * This function is *not* thread safe, and should not be invoked from multiple
 * goroutines.
 *
 * For now this function raises a fatal error if it is not able to pack the
 * result - this is in order to immediately catch errors in the pack() logic.
 * Eventually this should be a softer error (e.g. log & continue), but for now
 * is a hard crash so that errors do not pass in silence.
 *
 * [1] really exactly once, although nothing bad *should* happen if it is
 * called multiple times for the same object.
 */
func (p *process) pack() []byte {
	packed := C.pack(p.cpp)
	if packed.err {
		msg := "%s unable to pack result: %v"
		log.Fatalf(msg, p.logpid(), p.c_error())
	}
	return C.GoBytes(packed.body, packed.size)
}

/*
 * Make a container URL. This is just a stupid helper to make calling prettier,
 * and it is somewhat inflexible by reading endpoint + guid from the input
 * task.
 */
func (p *process) container() (azblob.ContainerURL, error) {
	endpoint := p.task.StorageEndpoint
	guid     := p.task.Guid
	container, err := url.Parse(fmt.Sprintf("%s/%s", endpoint, guid))
	if err != nil {
		err = fmt.Errorf("Container URL would be malformed: %w", err)
		return azblob.ContainerURL{}, err
	}

	container.RawQuery = p.task.UrlQuery

	credentials := azblob.NewAnonymousCredential()
	pipeline := azblob.NewPipeline(credentials, azblob.PipelineOptions{})
	return azblob.NewContainerURL(*container, pipeline), nil
}

/*
 * Gather the result and write the result to storage. This must *be called
 * exactly once* since it also clears the process handle.
 *
 * The function will block until either a message is read from the error
 * channel, or until n fragments have been read from the fragments channel. In
 * real programs this should usually be called as a goroutine, although it
 * doesn't have to without introducing deadlocks if the channels are
 * sufficiently buffered.
 *
 * This function finalizes the process.
 */
func (p *process) gather(
	storage    redis.Cmdable,
	nfragments int,
	queue      fetchQueue,
) {
	defer p.cleanup()
	for i := 0; i < nfragments; i++ {
		select {
		case f := <-queue.fragments:
			err := p.add(f)
			if err != nil {
				log.Fatalf("%s add failed: %v", p.logpid(), err)
			}
		case e := <-queue.errors:
			log.Printf("%s download failed: %v", p.logpid(), e)
			for {
				// Grab the remaining available errors to log them, but don't
				// wait around for any new ones to come in
				select {
				case e := <-queue.errors:
					log.Printf("%s download failed: %v", p.logpid(), e)
				default:
					return
				}
			}
		}
	}

	packed := p.pack()
	log.Printf("%s ready", p.logpid())
	args := redis.XAddArgs{
		Stream: p.pid,
		Values: map[string]interface{}{p.part: packed},
	}
	err := storage.XAdd(p.ctx, &args).Err()
	if err != nil {
		log.Printf("%s write to storage failed: %v", p.logpid(), err)
	}
	storage.Expire(p.ctx, p.pid, 10 * time.Minute)
	log.Printf("%s written to storage", p.logpid())
}
