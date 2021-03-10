package api

// #cgo LDFLAGS: -loneseismic -lfmt
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
	"fmt"

	"github.com/equinor/oneseismic/api/internal/message"
)

type cppscheduler struct {
	tasksize int
}

/*
 * This interface does feel superfluous, and should probably not need to be
 * exported. Using an interface makes testing a lot easier though, and unless
 * it should introduce an absurd performance penalty it's worth keeping around
 * for now.
 */
type scheduler interface {
	Schedule(*message.Task) ([][]byte, error)
	ScheduleRaw([]byte) ([][]byte, error)
}

func newScheduler() scheduler {
	return &cppscheduler{
		tasksize: 10,
	}
}

func (sched *cppscheduler) Schedule(task *message.Task) ([][]byte, error) {
	req, err := task.Pack()
	if err != nil {
		return nil, fmt.Errorf("pack error: %w", err)
	}
	return sched.ScheduleRaw(req)
}

func (sched *cppscheduler) ScheduleRaw(task []byte) ([][]byte, error) {
	csched := C.mkschedule(
		(*C.char)(unsafe.Pointer(&task[0])),
		C.int(len(task)),
		C.int(sched.tasksize),
	)
	defer C.cleanup(csched)

	result := make([][]byte, 0, int(csched.size))
	size := unsafe.Sizeof(C.struct_task {})
	base := uintptr(unsafe.Pointer(csched.tasks))
	for i := uintptr(0); i < uintptr(csched.size); i++ {
		this := (*C.struct_task)(unsafe.Pointer(base + (size * i)))
		result = append(result, C.GoBytes(this.task, this.size))
	}
	return result, nil
}
