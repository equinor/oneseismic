package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/equinor/oneseismic/api/internal/datastorage"
	"github.com/equinor/oneseismic/api/internal/util"
	"github.com/equinor/oneseismic/api/internal/fetch"

	"github.com/go-redis/redis/v8"
	"github.com/pborman/getopt/v2"
)

type opts struct {
	redis       string
	group       string
	stream      string
	consumerid  string
	jobs        int
	retries     int

	storageKind string	
	storageUrl  string
}

func parseopts() opts {
	help := getopt.BoolLong("help", 0, "print this help text")
	opts := opts {
		group:        "fetch",
		stream:       "jobs",
		storageKind:  os.Getenv("STORAGE_KIND"),
		storageUrl:   os.Getenv("STORAGE_URL"),
	}
	getopt.FlagLong(
		&opts.redis,
		"redis",
		'R',
		"Address to redis, e.g. host[:port]",
		"addr",
	).Mandatory()
	getopt.FlagLong(
		&opts.storageKind,
		"storage-kind",
		'K',
		"Kind of storage. " +
			  " Set this together with storage-url if you want to " +
			  " specify the storage-backend on startup." +
			  " Otherwise, storage-backend is derived from each task.",
		"name",
	)
	getopt.FlagLong(
		&opts.storageUrl,
		"storage-url",
		0,
		"Storage URL, e.g. https://<account>.blob.core.windows.net",
		"url",
	)
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
		10,
		"Allow N concurrent connections at once. Defaults to 10",
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

func main() {
	opts := parseopts()

	if opts.storageKind != "" && opts.storageUrl != "" {
		fetch.StorageSingleton = datastorage.CreateStorage(opts.storageKind, opts.storageUrl)
	}

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

	for {
		fetch.Run(ctx,
			          storage,
					  &args,
					  opts.jobs,
					  opts.retries)
	}
}
