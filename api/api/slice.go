package api

// #cgo LDFLAGS: -loneseismic -lfmt
// #include <stdlib.h>
// #include "slice.h"
import "C"
import "unsafe"

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/message"
	"github.com/equinor/oneseismic/api/internal/util"
)

type process struct {
	pid     string
	request []byte
}

type Slice struct {
	endpoint string // e.g. https://oneseismic-storage.blob.windows.net
	keyring  *auth.Keyring
	storage  redis.Cmdable
	tokens   auth.Tokens
}

func MakeSlice(
	keyring *auth.Keyring,
	endpoint string,
	storage  redis.Cmdable,
	tokens   auth.Tokens,
) Slice {
	return Slice {
		endpoint: endpoint,
		keyring: keyring,
		storage: storage,
		tokens:  tokens,
	}
}

type sliceParams struct {
	guid      string
	dimension int
	lineno    int
}

/*
 * Helper function to parse the parameters of a request, to keep int
 * parsing boilerplate out of interesting logic
 */
func parseSliceParams(ctx *gin.Context) (*sliceParams, error) {
	guid := ctx.Param("guid")
	if guid == "" {
		return nil, fmt.Errorf("guid empty")
	}

	dim := ctx.Param("dimension")
	dimension, err := strconv.Atoi(dim)
	if err != nil {
		return nil, fmt.Errorf("error parsing dimension: %w", err)
	}

	lno := ctx.Param("lineno")
	lineno, err := strconv.Atoi(lno)
	if err != nil {
		return nil, fmt.Errorf("error parsing lineno: %w", err)
	}

	return &sliceParams {
		guid: guid,
		dimension: dimension,
		lineno: lineno,
	}, nil
}

/*
 * Make the process prototype for the slice. This is really just a stupid
 * copy-parameter-into-struct function, but is a hook for sanity checks,
 * hard-coded values etc (such as the function parameter).
 */
func (s *Slice) makeTask(
	pid string,
	token string,
	manifest []byte,
	shape     []int32,
	shapecube []int32,
	params *sliceParams,
) message.Task {
	return message.Task {
		Pid:   pid,
		Token: token,
		Guid:  params.guid,
		StorageEndpoint: s.endpoint,
		Manifest: string(manifest),
		Shape: shape,
		ShapeCube: shapecube,
		Function: "slice",
		Params: &message.SliceParams {
			Dim:    params.dimension,
			Lineno: params.lineno,
		},
	}
}

func contains(haystack []int, needle int) bool {
	for _, x := range haystack {
		if x == needle {
			return true
		}
	}
	return false
}

func (s *Slice) Entry(ctx *gin.Context) {
	pid := ctx.GetString("pid")

	guid := ctx.Param("guid")
	if guid == "" {
		log.Printf("pid=%s, guid empty", pid)
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	m, err := util.GetManifest(ctx, s.tokens, s.endpoint, guid)
	if err != nil {
		log.Printf("%s %v", pid, err)
		return
	}

	dims := make([]message.DimensionDescription, len(m.Dimensions))
	for i := 0; i < len(m.Dimensions); i++ {
		dims[i] = message.DimensionDescription {
			Dimension: i,
			Size: len(m.Dimensions[i]),
			Keys: m.Dimensions[i],
		}
	}

	ctx.JSON(http.StatusOK, gin.H {
		"functions": gin.H {
			"slice": fmt.Sprintf("query/%s/slice", guid),
		},
		"dimensions": dims,
		"pid": pid,
	})
}

func (s *Slice) About(ctx *gin.Context) {
	pid := ctx.GetString("pid")
	guid := ctx.Param("guid")
	if guid == "" {
		log.Printf("%s guid empty", pid)
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	m, err := util.GetManifest(ctx, s.tokens, s.endpoint, guid)
	if err != nil {
		log.Printf("%s %v", pid, err)
		return
	}
	
	links := make(map[string]string)
	for i := 0; i < len(m.Dimensions); i++ {
		key := fmt.Sprintf("%d", i)
		links[key] = fmt.Sprintf("query/%s/slice/%d", guid, i)
	}

	ctx.JSON(http.StatusOK, gin.H {
		"links": links,
		"pid": pid,
	})
}

func makeSchedule(request []byte) [][]byte {
	raw := C.mkschedule(
		(*C.char)(unsafe.Pointer(&request[0])),
		C.int(len(request)),
		10, /* task-size */
	)
	defer C.cleanup(raw)

	result := make([][]byte, 0)
	size := unsafe.Sizeof(C.struct_task {})
	base := uintptr(unsafe.Pointer(raw.tasks))
	for i := uintptr(0); i < uintptr(raw.size); i++ {
		this := (*C.struct_task)(unsafe.Pointer(base + (size * i)))
		result = append(result, C.GoBytes(this.task, this.size))
	}
	return result
}

func (s *Slice) Get(ctx *gin.Context) {
	pid := ctx.GetString("pid")

	params, err := parseSliceParams(ctx)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	m, err := util.GetManifest(ctx, s.tokens, s.endpoint, params.guid)
	if err != nil {
		log.Printf("%s %v", pid, err)
		return
	}

	if !(params.dimension < len(m.Dimensions)) {
		msg := fmt.Sprintf(
			"param.dimension (= %d) not in [0, %d)",
			params.dimension,
			len(m.Dimensions),
		)
		log.Printf("%s %s in cube %s", pid, msg, params.guid)
		ctx.String(http.StatusNotFound, msg)
		ctx.Abort()
		return
	}
	if !contains(m.Dimensions[params.dimension], params.lineno) {
		msg := fmt.Sprintf("param.lineno (= %d) not in cube", params.lineno)
		log.Printf("%s %s %s", pid, msg, params.guid)
		ctx.String(http.StatusNotFound, msg)
		ctx.Abort()
		return
	}

	manifest, err := m.Pack()
	if err != nil {
		log.Printf("%s %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	/*
	 * Embedding a json doc as a string works (surprisingly) well, since the
	 * Pack()/encoding escapes all nested quotes. It might be reasonable at
	 * some point to change the underlying representation to messagepack, or
	 * even send the messages gzipped, but so for now strings and embedded
	 * documents should do fine.
	 *
	 * This opens an opportunity for the manifest forwarded not being quite
	 * faithful to what's stored in blob, i.e. information can be stripped out
	 * or added.
	 */
	authorization := ctx.GetHeader("Authorization")
	token, err := s.tokens.GetOnbehalf(authorization)
	if err != nil {
		// No further recovery is tried - GetManifest should already have fixed
		// a broken token, so this should be readily cached. If it is
		// just-about to expire then the process will fail pretty soon anyway,
		// so just give up.
		log.Printf("pid=%s, %v", pid, err)
		auth.AbortContextFromToken(ctx, err)
		return
	}

	cubeshape := make([]int32, 0, len(m.Dimensions))
	for i := 0; i < len(m.Dimensions); i++ {
		cubeshape = append(cubeshape, int32(len(m.Dimensions[i])))
	}
	msg := s.makeTask(
		pid,
		token,
		manifest,
		[]int32{ 64, 64, 64 },
		cubeshape,
		params,
	)
	req, err := msg.Pack()
	if err != nil {
		log.Printf("pid=%s, pack error: %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	key, err := s.keyring.Sign(pid)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	tasks := makeSchedule(req)
	ntasks := len(tasks)

	ctxcopy := ctx.Copy()
	go func() {
		/*
		 * Do the I/O in a go-routine to respond to clients faster.
		 *
		 * The context must be copied for this to not be broken
		 * https://pkg.go.dev/github.com/gin-gonic/gin#Context.Copy
		 */
		s.storage.Set(
			ctxcopy,
			fmt.Sprintf("%s:header.json", pid),
			fmt.Sprintf("{\"parts\": %d }", ntasks),
			10 * time.Minute,
		)
		for i, task := range tasks {
			values := []interface{} {
				"pid",  pid,
				"part", fmt.Sprintf("%d/%d", i, ntasks),
				"task", task,
			}
			args := redis.XAddArgs{Stream: "jobs", Values: values}
			_, err = s.storage.XAdd(ctxcopy, &args).Result()
			if err != nil {
				log.Fatalf("Unable to put %d in storage; %v", i, err)
			}
		}
	}()

	ctx.JSON(http.StatusOK, gin.H {
		"location": fmt.Sprintf("result/%s", pid),
		"status":   fmt.Sprintf("result/%s/status", pid),
		"authorization": key,
	})
}

func (s *Slice) List(ctx *gin.Context) {
	pid := ctx.GetString("pid")
	endpoint, err := url.Parse(s.endpoint)
	if err != nil {
		log.Printf("%s %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	authorization := ctx.GetHeader("Authorization")
	cubes, err := util.WithOnbehalfAndRetry(
		s.tokens,
		authorization,
		func (tok string) (interface{}, error) {
			return util.ListCubes(ctx, endpoint, tok)
		},
	)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		auth.AbortContextFromToken(ctx, err)
	}

	links := make(map[string]string)
	for _, cube := range cubes.([]string) {
		links[cube] = fmt.Sprintf("query/%s", cube)
	}

	ctx.JSON(http.StatusOK, gin.H {
		"links": links,
	})
}
