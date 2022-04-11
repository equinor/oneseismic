package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"net/url"

	"github.com/equinor/oneseismic/api/internal/util"

	"github.com/go-redis/redis/v8"
	"github.com/pborman/getopt/v2"
)

type opts struct {
	redis      string
	group      string
	stream     string
	consumerid string
	jobs       int
	retries    int
}

func parseopts() opts {
	help := getopt.BoolLong("help", 0, "print this help text")
	opts := opts {
		group:  "fetch",
		stream: "jobs",
	}
	getopt.FlagLong(
		&opts.redis,
		"redis",
		'R',
		"Address to redis, e.g. host[:port]",
		"addr",
	).Mandatory()
	getopt.FlagLong(
		&opts.group,
		"group",
		'G',
		"Consumer group. " +
		    "All workers should belong to the same group for " +
		    "fair distribution of work. " +
		    "You should normally not need to change this.",
		"name",
	)
	getopt.FlagLong(
		&opts.stream,
		"stream",
		'S',
		"Stream ID to read tasks from. Must be consistent with the producer. " +
		    "You should normally not need to change this.",
		"name",
	)
	getopt.FlagLong(
		&opts.consumerid,
		"consumer-id",
		'C',
		"Consumer ID of this worker. This should be unique among all the " +
			"workers in the consumer group. If no name is specified, " +
			"a random ID will be generated. You should normally not need " +
			"to specify a consumer ID.",
		"id",
	)
	jobs := getopt.IntLong(
		"jobs",
		'j',
		30,
		"Allow N concurrent connections at once. Defaults to 30",
		"N",
	)
	retries := getopt.IntLong(
		"retries",
		'r',
		0,
		"Max attempted retries when fetching from blobstore. Defaults to 0",
		"N",
	)
	getopt.Parse()

	if *help {
		getopt.Usage()
		os.Exit(0)
	}

	if opts.consumerid == "" {
		opts.consumerid = fmt.Sprintf("consumer:%s", util.MakePID())
	}
	opts.jobs = *jobs
	opts.retries = *retries
	return opts
}

func run(
	storage redis.Cmdable,
	fetch   *fetch,
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
	/*
	 * Build the container-URL early, in case it should be broken,
	 * so that no goroutines are scheduled before any sanity
	 * checking of input.
	 */
	container, err := proc.container()
	if err != nil {
		log.Printf("%s dropping bad process %v", proc.logpid(), err)
		return
	}

	fragments := proc.fragments()
	blobs := make([]*url.URL, len(fragments))
	for i, id := range fragments {
		blob, err := proc.blob(container, id)
		if err != nil {
			log.Printf("%s dropping bad process %v", proc.logpid(), err)
			return
		}
		blobs[i] = blob
	}

	fq := fetch.mkqueue()
	go proc.gather(storage, len(fragments), fq)
	fetch.enqueue(proc.ctx, fq, blobs)
}

func main() {
	opts := parseopts()

	storage := redis.NewClient(&redis.Options {
		Addr: opts.redis,
		DB: 0,
	})
	// TODO: err?
	defer storage.Close()

	ctx := context.Background()
	/*
	 * Always try to create the group and stream on start-up. The stream may
	 * have already been created, but that is a soft error to be discarded. In
	 * fact, the stream and group *probably* exists already because nodes
	 * connect in parallel.
	 *
	 * The XGroupCreate command is really just a try-create and fits well here,
	 * it offloads all the concurrency issues to redis. Consequently, this
	 * program can immediately go into the work loop assuming that the stream
	 * and group exists, without having to do any chatter or sync.
	 */
	err := storage.XGroupCreateMkStream(ctx, opts.stream, opts.group, "0").Err()
	if err != nil {
		 // Check if the response is a redis error (= BUSYGROUP), which just
		 // means the group already exists and nothing happens, or if it is a
		 // network error or something
		_, busygroup := err.(interface{RedisError()});
		if !busygroup {
			log.Fatalf(
				"Unable to create group %s for stream %s: %v",
				opts.group,
				opts.stream,
				err,
			)
		}
	}
	log.Printf(
		"consumer %s in group %s connecting to stream %s",
		opts.consumerid,
		opts.group,
		opts.stream,
	)

	// TODO: destroy consumers on shutdown
	// All reads can re-use the same group-args
	// NoAck is turned on - we can afford to fail requests and lose messages
	// should a node crash.
	args := redis.XReadGroupArgs {
		Group:    opts.group,
		Consumer: opts.consumerid,
		Streams:  []string { opts.stream, ">", },
		Count:    1,
		NoAck:    true,
	}

	fetch := newFetch(opts.jobs)
	fetch.startWorkers()

	for {
		msgs, err := storage.XReadGroup(ctx, &args).Result()
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
			err := storage.XDel(ctx, opts.stream, ids...).Err()
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
				run(storage, fetch, opts.retries, message.Values)
			}
		}
	}
}
