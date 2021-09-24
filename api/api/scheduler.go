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

type cppscheduler struct {
	storage  redis.Cmdable
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
	return &cppscheduler{
		storage:  storage,
	}
}

func (sched *cppscheduler) Schedule(
	ctx  context.Context,
	pid  string,
	plan *QueryPlan,
) error {
	/*
	 * TODO: This mixes I/O with parsing and building the plan. This could very
	 * well be split up into sub structs and functions which can then be
	 * dependency-injected for some customisation and easier testing.
	 */
	sched.storage.Set(
		ctx,
		fmt.Sprintf("%s/header.json", pid),
		plan.header,
		10 * time.Minute,
	)
	ntasks := len(plan.plan)
	for i, task := range plan.plan {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		part := fmt.Sprintf("%d/%d", i, ntasks)
		values := []interface{} {
			"pid",  pid,
			"part", part,
			"task", task,
		}
		args := redis.XAddArgs{Stream: "jobs", Values: values}
		_, err := sched.storage.XAdd(ctx, &args).Result()
		if err != nil {
			msg := "part=%v unable to put in storage; %w"
			return fmt.Errorf(msg, part, err)
		}
	}
	return nil
}
