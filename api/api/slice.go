package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pebbe/zmq4"
	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/util"
	"github.com/equinor/oneseismic/api/internal/message"
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
	manifest string,
	params *sliceParams,
) message.Task {
	return message.Task {
		Pid:   pid,
		Token: token,
		Guid:  params.guid,
		StorageEndpoint: s.endpoint,
		Manifest: manifest,
		Shape: []int32 { 64, 64, 64, },
		Function: "slice",
		Params: &message.SliceParams {
			Dim:    params.dimension,
			Lineno: params.lineno,
		},
	}
}

func (s *Slice) Get(ctx *gin.Context) {
	pid := util.MakePID()

	token := ctx.GetString("OBOJWT")
	if token == "" {
		/*
		 * The OBOJWT should be set in the middleware pipeline, so it's a
		 * programming error if it's not set
		 */
		log.Printf("%s OBOJWT was not set on request as it should be", pid)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	params, err := parseSliceParams(ctx)
	if err != nil {
		log.Printf("%s %v", pid, err)
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	msg := s.makeTask(pid, token, "{}", params)
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
		"result": fmt.Sprintf("result/%s", pid),
		"authorization": key,
	})
}
