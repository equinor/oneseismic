package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pebbe/zmq4"
	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/message"
	"github.com/equinor/oneseismic/api/internal/util"
)

type process struct {
	pid     string
	request []byte
}

func (proc *process) sendZMQ(socket *zmq4.Socket) (total int, err error) {
	return socket.SendMessage(proc.pid, proc.request)
}

type Slice struct {
	endpoint string // e.g. https://oneseismic-storage.blob.windows.net
	queue    *zmq4.Socket
	keyring  *auth.Keyring
}

func MakeSlice(
	keyring *auth.Keyring,
	endpoint string,
	queue *zmq4.Socket,
) Slice {
	return Slice {
		endpoint: endpoint,
		queue: queue,
		keyring: keyring,
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
	params *sliceParams,
) message.Task {
	return message.Task {
		Pid:   pid,
		Token: token,
		Guid:  params.guid,
		StorageEndpoint: s.endpoint,
		Manifest: string(manifest),
		Shape: []int32 { 64, 64, 64, },
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
		log.Printf("%s guid empty", pid)
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	m, err := util.GetManifest(ctx, s.endpoint, guid)
	if err != nil {
		log.Printf("%s %v", pid, err)
		return
	}

	dims := make([]map[string]int, len(m.Dimensions))
	for i := 0; i < len(m.Dimensions); i++ {
		dims[i] = map[string]int {
			"size":      len(m.Dimensions[i]),
			"dimension": i,
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

func (s *Slice) Get(ctx *gin.Context) {
	pid := ctx.GetString("pid")

	params, err := parseSliceParams(ctx)
	if err != nil {
		log.Printf("%s %v", pid, err)
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	m, err := util.GetManifest(ctx, s.endpoint, params.guid)
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
	token := ctx.GetString("OBOJWT")
	msg := s.makeTask(pid, token, manifest, params)
	req, err := msg.Pack()
	if err != nil {
		log.Printf("%s pack error: %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	key, err := s.keyring.Sign(pid)
	if err != nil {
		log.Printf("%s %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	proc := process {
		pid: pid,
		request: req,
	}
	log.Printf("Scheduling %s", pid)
	proc.sendZMQ(s.queue)
	ctx.JSON(http.StatusOK, gin.H {
		"location": fmt.Sprintf("result/%s", pid),
		"status":   fmt.Sprintf("result/%s/status", pid),
		"authorization": key,
	})
}
