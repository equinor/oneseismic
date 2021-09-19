package api

// #cgo LDFLAGS: -loneseismic
// #include <stdlib.h>
// #include "query.h"
import "C"
import "unsafe"

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/equinor/oneseismic/api/internal"
	"github.com/equinor/oneseismic/api/internal/message"
)

type QueryPlan struct {
	header []byte
	plan   [][]byte
}

/*
 * The Query Engine struct, which mostly just manages a pool of Session objects
 * that wrap C++ functionality and gives it a go interface.
 */
type QueryEngine struct {
	tasksize int
	pool     sync.Pool
}

/*
 * A QuerySession wraps C++ functionality and caches parsed messages, re-uses
 * buffers etc.
 */
type QuerySession struct {
	csession *C.struct_session
	tasksize int
}

func NewQuerySession() *QuerySession {
	return &QuerySession{
		csession: C.session_new(),
	}
}

func DefaultQueryEnginePool() sync.Pool {
	return sync.Pool {
		New: func() interface{} {
			return NewQuerySession()
		},
	}
}

func (qe *QueryEngine) Get() *QuerySession {
	q := qe.pool.Get().(*QuerySession)
	q.tasksize = qe.tasksize
	return q
}

func (qe *QueryEngine) Put(q *QuerySession) {
	qe.pool.Put(q)
}

func (q *QuerySession) InitWithManifest(doc []byte) error {
	cdoc := (*C.char)(unsafe.Pointer(&doc[0]))
	clen := C.int(len(doc))
	err := C.session_init(q.csession, cdoc, clen)
	if err != nil {
		/*
		 * This should only fail if JSON document is not a JSON document, or if
		 * it is not a manifest. Either way there is no recovery, and a healthy
		 * system should have no such documents.
		 */
		defer C.free(unsafe.Pointer(err))
		errstr := C.GoString(err)
		return internal.InternalError(errstr)
	}
	return nil
}

func (q *QuerySession) PlanQuery(query *message.Query) (*QueryPlan, error) {
	msg, err := query.Pack()
	if err != nil {
		return nil, fmt.Errorf("pack error: %w", err)
	}

	// TODO: exhaustive error check including those from C++ exceptions
	csched := C.session_plan_query(
		q.csession,
		(*C.char)(unsafe.Pointer(&msg[0])),
		C.int(len(msg)),
		C.int(q.tasksize),
	)
	defer C.plan_delete(&csched)
	if csched.err != nil {
		return nil, internal.QueryError(C.GoString(csched.err))
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

/*
 * QueryManifest is essentially a JSON pointer [1] interface to the manifest
 * object. It is intended to support simple read-field-in-manifest resolvers,
 * without having to parse the manifest every time.
 *
 * It is written through cgo and the query engine in order to make go a pure
 * I/O layer, and to contain all parsing, validation, and lookup in the same
 * module. This means certain classes of errors can be handled centrally too,
 * rather than having to take up space in every single block
 *
 * This does make a single query lookup *significantly* more complicated than
 * it would be in pure go, but it makes adding new calls very cheap, and it
 * makes for a single feature that must be understood.
 *
 * This function returns a JSON encoded byte array (if not an error). Ideally
 * graphql could just write this directly, but the resolver system expects to
 * find a return type of the resolver that matches the schema type. An easy fix
 * is to parse the RawMessage into the "destination" type. The overhead is of
 * course outrageous, but it can work until the upstream graphql support can
 * handle json.rawmessage or similar.
 *
 * [1] https://rapidjson.org/md_doc_pointer.html
 */
func (q *QuerySession) QueryManifest(path string) (json.RawMessage, error) {
	pathb    := []byte(path)
	pathcstr := (*C.char)(unsafe.Pointer(&pathb[0]))
	pathlen  := C.int(len(path))

	result := C.session_query_manifest(q.csession, pathcstr, pathlen)
	defer C.query_result_delete(&result)
	if result.err != nil {
		errstr := C.GoString(result.err)
		return nil, fmt.Errorf("QueryManifest: %v", errstr)
	}
	return C.GoBytes(unsafe.Pointer(result.body), result.size), nil
}
