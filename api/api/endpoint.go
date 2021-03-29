package api

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"

	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/message"
	"github.com/equinor/oneseismic/api/internal/util"
)

type BasicEndpoint struct {
	endpoint string // e.g. https://oneseismic-storage.blob.windows.net
	keyring  *auth.Keyring
	tokens   auth.Tokens
	sched    scheduler
}

func MakeBasicEndpoint(
	keyring *auth.Keyring,
	endpoint string,
	storage  redis.Cmdable,
) BasicEndpoint {
	return BasicEndpoint {
		endpoint: endpoint,
		keyring: keyring,
		/*
		 * Scheduler should probably be exported (and in internal/?) and be
		 * constructed directly by the caller.
		 */
		sched:   newScheduler(storage),
	}
}

func (be *BasicEndpoint) MakeTask(
	pid       string,
	guid      string,
	token     string,
	manifest  []byte,
	shape     []int32,
	shapecube []int32,
) *message.Task {
	return &message.Task {
		Pid:             pid,
		Token:           token,
		Guid:            guid,
		StorageEndpoint: be.endpoint,
		Manifest:        string(manifest),
		Shape:           shape,
		ShapeCube:       shapecube,
	}
}

func (be *BasicEndpoint) Root(ctx *gin.Context) {
	pid := ctx.GetString("pid")

	guid := ctx.Param("guid")
	if guid == "" {
		log.Printf("pid=%s, guid empty", pid)
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	m, err := util.GetManifest(ctx, be.endpoint, guid)
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

func (be *BasicEndpoint) List(ctx *gin.Context) {
	pid := ctx.GetString("pid")
	endpoint, err := url.Parse(be.endpoint)
	if err != nil {
		log.Printf("pid=%s %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	token := ctx.GetString("Token")
	cubes, err := util.ListCubes(ctx, endpoint, token)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
	}

	links := make(map[string]string)
	for _, cube := range cubes {
		links[cube] = fmt.Sprintf("query/%s", cube)
	}

	ctx.JSON(http.StatusOK, gin.H {
		"links": links,
	})
}

func (s *BasicEndpoint) Entry(ctx *gin.Context) {
	pid := ctx.GetString("pid")

	guid := ctx.Param("guid")
	if guid == "" {
		log.Printf("pid=%s, guid empty", pid)
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	m, err := util.GetManifest(ctx, s.endpoint, guid)
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
			"slice":   fmt.Sprintf("query/%s/slice",   guid),
			"curtain": fmt.Sprintf("query/%s/curtain", guid),
		},
		"dimensions": dims,
		"pid": pid,
	})
}
