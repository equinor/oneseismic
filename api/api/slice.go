package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pebbe/zmq4"
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
}

func MakeSlice(endpoint string, queue *zmq4.Socket) Slice {
	return Slice {
		endpoint: endpoint,
		queue: queue,
	}
}

type sliceParams struct {
	guid      string
	dimension int
	lineno    int
}

type Task struct {
	Pid             string  `json:"pid"`
	Token           string  `json:"token"`
	Guid            string  `json:"guid"`
	StorageEndpoint string  `json:"storage_endpoint"`
	Shape           []int32 `json:"shape"`
	Function        string  `json:"function"`
	Params          interface {} `json:"params"`
}

type SliceParams struct {
	Dim    int `json:"dim"`
	Lineno int `json:"lineno"`
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

	msg := Task {
		Pid:   pid,
		Token: token,
		Guid:  params.guid,
		StorageEndpoint: s.endpoint,
		Shape: []int32 { 64, 64, 64, },
		Function: "slice",
		Params: &SliceParams {
			Dim:    params.dimension,
			Lineno: params.lineno,
		},
	}

	req, err := json.Marshal(msg)
	if err != nil {
		log.Printf("%s marshalling error: %v", pid, err)
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
	})
}
