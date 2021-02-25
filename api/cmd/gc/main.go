package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pborman/getopt/v2"
)

type opts struct {
	redis     string
	stream    string
	group     string
	threshold time.Duration
	dryrun    bool
}

func parseopts() opts {
	help := getopt.BoolLong("help", 0, "print this help text")
	opts := opts {
		stream: "jobs",
		group:  "fetch",
		threshold: 30 * time.Minute,
	}
	getopt.FlagLong(
		&opts.redis,
		"redis",
		'R',
		"Address to redis, e.g. host[:port]",
		"addr",
	).Mandatory()
	getopt.FlagLong(
		&opts.stream,
		"stream",
		'S',
		"Stream to garbage collect",
		"key",
	)
	getopt.FlagLong(
		&opts.group,
		"group",
		'G',
		"Consumer group to garbage collect",
		"group",
	)
	getopt.FlagLong(
		&opts.threshold,
		"threshold",
		't',
		"Idle duration before consumer is a candidate for garbage collection",
		"duration",
	)
	getopt.FlagLong(
		&opts.dryrun,
		"dry-run",
		'n',
		"Do not actually remove anything, just show what would be done",
	).SetFlag()
	getopt.Parse()

	if *help {
		getopt.Usage()
		os.Exit(0)
	}

	return opts
}

/*
 * The garbage collector addresses a fundamental issue with a scaling
 * distributed system.
 *
 * It is assumed that any node can crash at any time, and it cannot be assumed
 * that crashes will "clean up" after itself. For example, workers can added
 * and removed both explicitly by an operator, and dynamically by some runetime
 * (like kubernetes). Workers get jobs by pulling them from a redis-provided
 * task queue, but that also registeres them by storing a consumer ID. The set
 * of seen consumer IDs will grow indefinitely.
 *
 * The garbage collection program encodes and executes the know-how of cleaning
 * up orphaned stuff from the databases.
 */
func main() {
	opts := parseopts()

	storage := redis.NewClient(&redis.Options {
		Addr: opts.redis,
	})
	defer storage.Close()
	ctx := context.Background()

	cmd := storage.XInfoConsumers(ctx, opts.stream, opts.group)
	consumers, err := cmd.Result()
	if err != nil {
		log.Fatal(err)
	}

	garbage := []string{}
	for _, consumer := range consumers {
		if consumer.Idle > opts.threshold.Milliseconds() {
			garbage = append(garbage, consumer.Name)
		}
	}

	for _, id := range garbage {
		log.Printf(
			"Removing consumer %s from group %s in stream %s",
			id,
			opts.group,
			opts.stream,
		)
		if opts.dryrun {
			continue
		}
		/*
		 * The consumer could be idle both from being abandoned (e.g. the node
		 * scaled down or restarted) and there just not being any work, but if
		 * the node is still alive then the consumer will be re-iniated on the
		 * next available job and nothing will be lost. This is ok because jobs
		 * are fetched with NoAck so there are no pending-but-not-acked
		 * messages. This has been tested manually to work well, but I have not
		 * found a good reference with guarantees from redis, so this *might*
		 * come to bite us later.
		 */
		err := storage.XGroupDelConsumer(ctx, opts.stream, opts.group, id).Err()
		if err != nil {
			log.Fatalf("Could not delete consumer %s; %v", id, err)
		}
	}
}
