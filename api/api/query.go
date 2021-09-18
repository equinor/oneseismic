package api

// #cgo LDFLAGS: -loneseismic
// #include <stdlib.h>
// #include "query.h"
import "C"
import "unsafe"

import (
	"fmt"

	"github.com/equinor/oneseismic/api/internal/message"
)

type QueryPlan struct {
	header []byte
	plan   [][]byte
}

type QueryError struct {
	msg    string
}

func (qe *QueryError) Error() string {
	return qe.msg
}

/*
 * The Query Engine struct, which mostly just wraps the C++ functionality and
 * gives it a go interface. This makes an "executable plan" (set of job
 * descriptions) from a query, e.g. sliceByIndex.
 */
type QueryEngine struct {
	tasksize int
}

func (qe *QueryEngine) PlanQuery(query *message.Query) (*QueryPlan, error) {
	msg, err := query.Pack()
	if err != nil {
		return nil, fmt.Errorf("pack error: %w", err)
	}
	// TODO: exhaustive error check including those from C++ exceptions
	csched := C.mkschedule(
		(*C.char)(unsafe.Pointer(&msg[0])),
		C.int(len(msg)),
		C.int(qe.tasksize),
	)
	defer C.cleanup(&csched)
	if csched.err != nil {
		return nil, &QueryError {
			msg: C.GoString(csched.err),
		}
	}

	ntasks := int(csched.len)
	result := make([][]byte, 0, ntasks)
	sizes  := (*[1 << 30]C.int)(unsafe.Pointer(csched.sizes))[:ntasks:ntasks]

	this := uintptr(unsafe.Pointer(csched.tasks))
	for _, size := range sizes {
		result = append(result, C.GoBytes(unsafe.Pointer(this), size))
		this += uintptr(size)
	}

	headerindex := len(result) - 1
	return &QueryPlan {
		header: result[headerindex],
		plan:   result[:headerindex],
	}, nil
}
