package fetch

// #cgo LDFLAGS: -loneseismic -lfmt
// #include <stdlib.h>
// #include "tasks.h"
import "C"
import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unsafe"

	"github.com/equinor/oneseismic/api/api"
	"github.com/equinor/oneseismic/api/internal/datastorage"
	"github.com/equinor/oneseismic/api/internal/message"
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
* The storage can be a global singleton in this module.
* This means the driver can override it if we decide this is the best
* overall solution (i.e driver assumes responsibility of its storage-
* backend). See also TODOs in api.AbstractStorage.
*
* If not set by driver, each task will look it up from a map where kind
* and endpoint is combined to key.
*/
var StorageSingleton api.AbstractStorage = nil
var storageCache map[string]api.AbstractStorage = make(map[string]api.AbstractStorage)
func getStorage(kind string, endpoint string) *api.AbstractStorage {
	if StorageSingleton != nil {
		return &StorageSingleton
	}
	key := fmt.Sprintf("%s::%s", kind, endpoint)
	retval, ok := storageCache[key]; if !ok {
		retval = datastorage.CreateStorage(kind, endpoint)
		storageCache[key] = retval
	}
	return &retval
}

type task struct {
	index int
	id string
	blobStorage  *api.AbstractStorage
	credentials string
}

/*
* This method retrieves all available incoming jobs in the specified
* Redis-group and executes them. The value of "args.Block" decides
* whether the method blocks or times out when no jobs are available.
*
* In a driver/worker-process it can be called in a loop, alternatively
* terminating on some condition (max number of iterations, max memory-usage,
* max age, etc, etc) before exiting.
*
* A test will just call it to process whatever jobs has been triggered.
*/
func Run(ctx context.Context,
			 storage *redis.Client,
			 args *redis.XReadGroupArgs,
			 njobs int, retries int) {
		msgs, err := storage.XReadGroup(ctx, args).Result()
		if err != nil {
			log.Fatalf("Unable to read from redis: %v", err)
		}

		go func() {
			/*
			 * Send a request-for-delete once the message has been read, in
			 * order to stop the infinite growth of the job queue.
			 *
			 * This is the simplest solution that is correct [1] - the node
			 * that gets a job also deletes it, which emulates a
			 * fire-and-forget job queue. Unfortunately it also means more
			 * traffic back to the central job queue node. In redis6.2 the
			 * XTRIM MINID strategy is introduced, which opens up some
			 * interesting strategies for cleaning up the job queue. This is
			 * work for later though.
			 *
			 * [1] except in some crashing scenarios
			 */
			ids := make([]string, 0, 3)
			for _, xmsg := range msgs {
				for _, msg := range xmsg.Messages {
					ids = append(ids, msg.ID)
				}
			}
			err := storage.XDel(ctx, args.Streams[0], ids...).Err()
			if err != nil {
				log.Fatalf("Unable to XDEL: %v", err)
			}
		}()

		/*
		 * The redis interface is designed for asking for a set of messages per
		 * XReadGroup command, but we really only ask for one [1]. The redis-go
		 * API is is aware of this which means the message structure must be
		 * unpacked with nested loops. As a consequence, the read-count can
		 * be increased with little code change, but it also means more nested
		 * loops.
		 *
		 * For ease of understanding, the loops can be ignored.
		 *
		 * [1] Instead opting for multiple fragments to download per message.
		 *     This is a design decision from before redis streams, but it
		 *     works well with redis streams too.
		 */
		for _, xmsg := range msgs {
			for _, message := range xmsg.Messages {
				// TODO: graceful shutdown and/or cancellation
				doProcess(storage, njobs, retries, message.Values)
			}
		}

}
func doProcess(
	storage redis.Cmdable,
	njobs   int,
	retries int,
	process map[string]interface{},
) {
	/*
	 * Curiously, the XReadGroup/XStream values end up being map[string]string
	 * effectively. This is detail of the go library where it uses ReadLine()
	 * internally - redis uses byte strings as strings anyway.
	 *
	 * The type assertion is not checked and will panic. This is a good thing
	 * as it should only happen when the redis library is updated to no longer
	 * return strings, and crash oneseismic properly. This should catch such a
	 * change early.
	 */
	pid  := process["pid" ].(string)
	part := process["part"].(string)
	body := process["task"].(string)
	msg  := [][]byte{ []byte(pid), []byte(part), []byte(body) }
	proc, err := exec(msg)
	if err != nil {
		log.Printf("%s dropping bad process %v", proc.logpid(), err)
		return
	}

	// TODO: share job queue between processes, but use private
	// output/error queue? Then buffered-elements would control
	// number of pending jobs Rather than it now continuing once
	// the last task has been scheduled and possibly spawning N
	// more goroutines.
	frags  := make(chan fragment, njobs)
	tasks  := make(chan task, njobs)
	errors := make(chan error)
	/*
	 * The goroutines must be signalled that there are no more data, or
	 * they will leak.
	 */
	defer close(tasks)
	for i := 0; i < njobs; i++ {
		go fetch(proc.ctx, retries, tasks, frags, errors)
	}
	fragments := proc.fragments()
	guid := proc.task.Guid
	go proc.gather(storage, len(fragments), frags, errors)
	for i, id := range fragments {
		select {
		case tasks <- task { index: i,
							id: fmt.Sprintf("%s#%s", guid, id),
							blobStorage: getStorage(proc.task.StorageKind,
                                                    proc.task.StorageEndpoint),
							credentials: proc.task.Credentials,
							}:
		case <-proc.ctx.Done():
			msg := "%s cancelled after %d scheduling fragments; %v"
			log.Printf(msg, proc.logpid(), i, proc.ctx.Err())
			break
		}
	}
}

/*
 * Automation - return the pid=, proc= log line prefix to make log formatting
 * less noisy.
 *
 * This is the only function allowed to call on a process if exec() returns an
 * error.
 */
func (p *process) logpid() string {
	return fmt.Sprintf("pid=%s, part=%s", api.GetPid(p.ctx), p.part)
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
	cache := make(map[string]interface{})
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "pid", string(msg[0]))
	ctx = context.WithValue(ctx, "cache", cache)
	proc := &process {
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
	C.cleanupProcess(p.cpp)
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
 * A reference to a downloaded fragment (as it is stored in blob)
 */
type fragment struct {
	index int
	chunk []byte
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
	fragments  chan fragment,
	errors     chan error,
) {
	defer p.cleanup()
	for i := 0; i < nfragments; i++ {
		select {
		case f := <-fragments:
			err := p.add(f)
			if err != nil {
				log.Fatalf("%s add failed: %v", p.logpid(), err)
			}
		case e := <-errors:
			log.Printf("%s download failed: %v", p.logpid(), e)
            GRAB_THE_REST_LOOP:
			for {
				// Grab the remaining available errors to log them, but don't
				// wait around for any new ones to come in
				select {
				case e := <-errors:
					log.Printf("%s download failed: %v", p.logpid(), e)
				default:
					break GRAB_THE_REST_LOOP
				}
			}
			// In order to signal error, fill the stream with error-msgs
			// and make sure there is one more msg than nfragments so
			// that result-handlers can determine this quickly
			args := redis.XAddArgs{Stream: api.GetPid(p.ctx),
				          Values: map[string]interface{}{"error": e.Error()}}
			log.Printf("%s signal error and return", p.logpid())
			n := 0
			for ii := i; ii < nfragments+1; ii++ {
				n++
				err := storage.XAdd(p.ctx, &args).Err()
				if err != nil {
					log.Printf("%s write error to storage failed: %v", p.logpid(), err)
				}
			}
			count, _ := storage.XLen(p.ctx, p.logpid()).Result()
			log.Printf("%s %d errors written to %v. len==%d", p.logpid(), n, args.Stream, count)

			return
		} // select
	}

	packed := p.pack()
	log.Printf("%s ready", p.logpid())
	args := redis.XAddArgs{
		Stream: api.GetPid(p.ctx),
		Values: map[string]interface{}{p.part: packed},
	}
	err := storage.XAdd(p.ctx, &args).Err()
	if err != nil {
		log.Printf("%s write to storage failed: %v", p.logpid(), err)
	}
	storage.Expire(p.ctx, api.GetPid(p.ctx), 10 * time.Minute)
	log.Printf("%s written to storage", p.logpid())
}

/*
 * Fetch fragments from the blob store, and write them to the fragments
 * channel. This is a simple worker loop, which will grab tasks until the input
 * channel is closed.
 */
func fetch(
	ctx        context.Context,
	maxRetries int,
	tasks      chan task,
	fragments  chan fragment,
	errorChan  chan error,
) {
	for task := range tasks {
		select {
		case <- ctx.Done():
			errorChan <- errors.New("operation was cancelled")
			return
		default:
			chunk, err := (*task.blobStorage).Get(ctx, task.credentials, task.id)
			if err != nil {
				errorChan <- err
				return
			}
			fragments <- fragment {
				index: task.index,
				chunk: chunk,
			}
		}
	}
}
