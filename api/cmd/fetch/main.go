package main

import (
	"log"
	"os"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/go-redis/redis"
	"github.com/pborman/getopt/v2"
	"github.com/pebbe/zmq4"
)

type opts struct {
	source    string
	redis     string
	heartbeat time.Duration
	jobs      int
}

func parseopts() opts {
	help := getopt.BoolLong("help", 0, "print this help text")
	opts := opts {}
	getopt.FlagLong(
		&opts.source,
		"source",
		's',
		"Address to source, e.g. tcp://host[:port]",
		"addr",
	).Mandatory()
	getopt.FlagLong(
		&opts.redis,
		"redis",
		'R',
		"Address to redis, e.g. host[:port]",
		"addr",
	).Mandatory()
	heartbeat := getopt.DurationLong(
		"heartbeat",
		'H',
		60 * time.Second,
		"time between READY messages to source",
		"interval",
	)
	jobs := getopt.IntLong(
		"jobs",
		'j',
		10,
		"Allow N concurrent connections at once. Defaults to 10",
		"N",
	)
	getopt.Parse()

	if *help {
		getopt.Usage()
		os.Exit(0)
	}

	opts.heartbeat = *heartbeat
	opts.jobs = *jobs
	return opts
}

type task struct {
	index int
	blob  azblob.BlobURL
}

func main() {
	opts := parseopts()

	source, err := zmq4.NewSocket(zmq4.DEALER)
	if err != nil {
		log.Fatalf("Unable to create socket: %v", err)
	}
	err = source.Connect(opts.source)
	if err != nil {
		log.Fatalf("Unable to connect queue to %s: %v", opts.source, err)
	}
	defer source.Close()

	storage := redis.NewClient(&redis.Options {
		Addr: opts.redis,
		DB: 0,
	})
	// TODO: err?
	defer storage.Close()

	poll := zmq4.NewPoller()
	poll.Add(source, zmq4.POLLIN)
	for {
		/*
		 * There should only be new assignments after a READY message is sent,
		 * so it should be performed unconditionally at the start of the loop.
		 */
		_, err := source.Send("READY", 0)
		if err != nil {
			switch zmq4.AsErrno(err) {
				/*
				 * Add handlers for recoverable errors here
				 */
			}

			/*
			 * Take down the service if the source socket breaks. This is a bit
			 * heavy handed, but I have no good grasp right now on what might
			 * go wrong or is recoverable, so the right call is to promptly
			 * kill anything that breaks the socket and fix issues when they
			 * come up.
			 *
			 * There should be an automated log alert for this error.
			 */
			log.Fatalf("socket broken: %v", err)
		}

		active, err := poll.Poll(opts.heartbeat)
		if err != nil {
			/*
			 * Take down the service if the poll breaks. This is a bit heavy
			 * handed, but I have no good grasp right now on what might go
			 * wrong, and what is recoverable, so the right call is to properly
			 * kill anything that breaks and fix issues as they come up.
			 *
			 * There should be an automated log alert for this error.
			 */
			log.Fatalf("poll broken: %v", err)
		}

		/*
		 * Process pending requests
		 *
		 * There's some syntactical noise here with the range-of-sockets and
		 * then switch. It is quite unnecessary for as long as there's only one
		 * queue in use, but there's a high probability there will be more, and
		 * with for-all infrastructure set up it should be easy to integrate
		 * new handlers into the loop.
		 */
		for _, socket := range active {
			switch s := socket.Socket; s {
			case source:
				msg, err := source.RecvMessageBytes(0)
				if err != nil {
					log.Fatalf("recvmessage broken: %v", err)
				}
				proc, err := exec(msg)
				if err != nil {
					log.Printf("%s dropping bad process %v", proc.logpid(), err)
					break
				}
				/*
				 * Build the container-URL early, in case it should be broken,
				 * so that no goroutines are scheduled before any sanity
				 * checking of input.
				 */
				container, err := proc.container()
				if err != nil {
					log.Printf("%s dropping bad process %v", proc.logpid(), err)
					break
				}
				log.Printf("%s being processed", proc.logpid())
				// TODO: share job between processes, but private use
				// output/error queue? Then buffered-elements would control
				// number of pending jobs Rather than it now continuing once
				// the last task has been scheduled and possibly spawning N
				// more goroutines.
				frags  := make(chan fragment)
				tasks  := make(chan task)
				errors := make(chan error)
				for i := 0; i < opts.jobs; i++ {
					go fetch(proc.ctx, tasks, frags, errors)
				}
				fragments := proc.fragments()
				go proc.gather(storage, len(fragments), frags, errors)
				for i, id := range fragments {
					tasks <- task { index: i, blob: container.NewBlobURL(id) }
				}
				/*
				 * The goroutines must be signalled that there are no more
				 * data, or they will leak. defer close() cannot be used since
				 * it works on a function scope, unless the rest of the body is
				 * wrapped in a func(){}()
				 *
				 * https://stackoverflow.com/questions/49456943/why-is-golang-defer-scoped-to-function-not-lexical-enclosure
				 */
				close(tasks)

			default:
				log.Fatalf("unhandled message: %v", s)
			}
		}
	}
}
