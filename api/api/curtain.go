package api

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/message"
	"github.com/equinor/oneseismic/api/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type Curtain struct {
	BasicEndpoint
}

func MakeCurtain(
	keyring  *auth.Keyring,
	endpoint string,
	storage  redis.Cmdable,
) *Curtain {
	return &Curtain {
		MakeBasicEndpoint(
			keyring,
			endpoint,
			storage,
		),
	}
}

func (c *Curtain) MakeTask(
	pid       string,
	guid      string,
	endpoint  string,
	token     string,
	manifest  []byte,
	shape     []int32,
	shapecube []int32,
	params    *message.CurtainParams,
) *message.Task {
	task := c.BasicEndpoint.MakeTask(
		pid,
		guid,
		endpoint,
		token,
		manifest,
		shape,
		shapecube,
	)
	task.Function = "curtain"
	task.Params   = params
	return task
}

/*
 * The path is a tiny helper to parse the input "parameter" (the request body).
 * Eventually this will parse some normalised format, either from a previously
 * parsed input, or from some curtain storage system.
 */
type path struct {
	Intersections [][2]int `json:intersections`
}

func (p *path) toCurtainParams(
	manifest *message.Manifest,
) (*message.CurtainParams, error) {
	xs := make([]int, len(p.Intersections))
	ys := make([]int, len(p.Intersections))
	for i, xy := range p.Intersections {
		xs[i] = xy[0]
		ys[i] = xy[1]
	}
	return &message.CurtainParams{ Dim0s: xs, Dim1s: ys }, nil
}

func (c *Curtain) Get(ctx *gin.Context) {
	pid := ctx.GetString("pid")
	guid := ctx.Param("guid")

	m, err := util.GetManifest(ctx, c.endpoint, guid)
	if err != nil {
		log.Printf("pid=%s %v", pid, err)
		return
	}

	path := path {}
	err = ctx.ShouldBindJSON(&path)
	if err != nil {
		log.Printf("pid=%s %v", pid, err)
		return
	}

	params, err := path.toCurtainParams(m)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	token := ctx.GetString("Token")
	if err != nil {
		// No further recovery is tried - GetManifest should already have fixed
		// a broken token, so this should be readily cached. If it is
		// just-about to expire then the process will fail pretty soon anyway,
		// so just give up.
		log.Printf("pid=%s, %v", pid, err)
		return
	}

	manifest, err := m.Pack()
	if err != nil {
		log.Printf("pid=%s %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	cubeshape := make([]int32, 0, len(m.Dimensions))
	for i := 0; i < len(m.Dimensions); i++ {
		cubeshape = append(cubeshape, int32(len(m.Dimensions[i])))
	}
	endpoint, err := util.WithSASToken(ctx, c.endpoint)
	if err != nil {
		log.Printf("pid=%s %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	msg := c.MakeTask(
		pid,
		guid,
		endpoint.String(),
		token,
		manifest,
		[]int32{ 64, 64, 64 },
		cubeshape,
		params,
	)

	key, err := c.keyring.Sign(pid)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	query, err := c.sched.MakeQuery(msg)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		qe := err.(*QueryError)
		if qe.Status() != 0 {
			ctx.AbortWithStatus(qe.Status())
		} else {
			ctx.AbortWithStatus(http.StatusInternalServerError)
		}
		return
	}
	go func() {
		err := c.sched.Schedule(context.Background(), pid, query)
		if err != nil {
			/*
			 * Make scheduling errors fatal to detect them for debugging.
			 * Eventually this should log, maybe cancel the process, and
			 * continue.
			 */
			log.Fatalf("pid=%s, %v", pid, err)
		}
	}()

	ctx.JSON(http.StatusOK, gin.H {
		"location": fmt.Sprintf("result/%s", pid),
		"status":   fmt.Sprintf("result/%s/status", pid),
		"authorization": key,
	})
}
