package api

/*
 * This module implements the go-interface to the scheduler module. The
 * scheduler module is access through cgo, but it conceptually be a separate
 * service running on a different computer somewhere. It should be viewed as
 * starting an external program and parsing its output. In fact, it is
 * explicitly designed to communicate with messages (passed as byte arrays) so
 * that if scaling or design needs mandate it, scheduling can be moved out of
 * this process and into a separate.
 */

import(
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type redisScheduler struct {
	queue redis.Cmdable
	/*
	 * Time-to-live for the response, i.e. after this duration results will be
	 * cleaned up
	 */
	ttl   time.Duration
}

/*
 * This interface does feel superfluous, and should probably not need to be
 * exported. Using an interface makes testing a lot easier though, and unless
 * it should introduce an absurd performance penalty it's worth keeping around
 * for now.
 */
type scheduler interface {
	Schedule(context.Context, string, *QueryPlan) error
}

func NewScheduler(storage redis.Cmdable) scheduler {
	return &redisScheduler {
		queue: storage,
		ttl:   10 * time.Minute,
	}
}

func (rs *redisScheduler) Schedule(
	ctx  context.Context,
	pid  string,
	plan *QueryPlan,
) error {
	err := rs.queue.Set(
		ctx,
		fmt.Sprintf("%s/header.json", pid),
		plan.header,
		rs.ttl,
	).Err()
	if err != nil {
		return err
	}
	values := []interface{} {
		"pid",  pid,
		"part", nil,
		"task", nil,
	}
	args := &redis.XAddArgs{Stream: "jobs", Values: values}
	ntasks := len(plan.plan)
	for i, task := range plan.plan {
		part := fmt.Sprintf("%d/%d", i, ntasks)
		values[3] = part
		values[5] = task
		err := rs.queue.XAdd(ctx, args).Err()
		if err != nil {
			msg := "pid=%s, part=%v, unable to schedule: %w"
			return fmt.Errorf(msg, pid, part, err)
		}
	}
	return nil
}
