package api

// #cgo LDFLAGS: -loneseismic
// #include <stdlib.h>
// #include "scheduler.h"
import "C"
import "unsafe"

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

	"github.com/equinor/oneseismic/api/internal/message"
)

type cppscheduler struct {
	tasksize int
	storage  redis.Cmdable
}

type Query struct {
	header []byte
	plan   [][]byte
}

/*
 * This interface does feel superfluous, and should probably not need to be
 * exported. Using an interface makes testing a lot easier though, and unless
 * it should introduce an absurd performance penalty it's worth keeping around
 * for now.
 */
type scheduler interface {
	MakeQuery(*message.Task) (*Query, error)
	Schedule(context.Context, string, *Query) error
}

func newScheduler(storage redis.Cmdable) scheduler {
	return &cppscheduler{
		storage:  storage,
		tasksize: 10,
	}
}

func (sched *cppscheduler) MakeQuery(msg *message.Task) (*Query, error) {
	task, err := msg.Pack()
	if err != nil {
		return nil, fmt.Errorf("pack error: %w", err)
	}
	// TODO: exhaustive error check including those from C++ exceptions
	csched := C.mkschedule(
		(*C.char)(unsafe.Pointer(&task[0])),
		C.int(len(task)),
		C.int(sched.tasksize),
	)
	defer C.cleanup(&csched)
	if csched.err != nil {
		return nil, fmt.Errorf(C.GoString(csched.err))
	}

	ntasks := int(csched.len)
	result := make([][]byte, 0, ntasks)
	sizes  := (*[1 << 30]C.int)(unsafe.Pointer(csched.sizes))[:ntasks:ntasks]

	this := uintptr(unsafe.Pointer(csched.tasks))
	for _, size := range sizes {
		result = append(result, C.GoBytes(unsafe.Pointer(this), size))
		this += uintptr(size)
	}

	return &Query {
		header: result[0],
		plan:   result[1:],
	}, nil
}

func (sched *cppscheduler) Schedule(
	ctx  context.Context,
	pid  string,
	plan *Query,
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
