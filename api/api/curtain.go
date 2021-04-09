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
	tokens   auth.Tokens,
) *Curtain {
	return &Curtain {
		MakeBasicEndpoint(
			keyring,
			endpoint,
			storage,
			tokens,
		),
	}
}

func (c *Curtain) MakeTask(
	pid       string,
	guid      string,
	token     string,
	manifest  []byte,
	shape     []int32,
	shapecube []int32,
	params    *message.CurtainParams,
) *message.Task {
	task := c.BasicEndpoint.MakeTask(
		pid,
		guid,
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
 *
 * It comes with a single method, the zeroindexed() which returns the
 * zero-index'd path, to keep that noise out of the handler body.
 */
type path struct {
	Intersections [][2]int `json:intersections`
}

func (p *path) zeroindexed(
	manifest *message.Manifest,
) (*message.CurtainParams, error) {
	// binary-search instead?
	// since we know lines are sorted, use bin-search in contains()
	dim0 := make(map[int]int, len(manifest.Dimensions[0]))
	dim1 := make(map[int]int, len(manifest.Dimensions[1]))
	for i, x := range manifest.Dimensions[0] {
		dim0[x] = i + 1
	}
	for i, x := range manifest.Dimensions[1] {
		dim1[x] = i + 1
	}
	xs := make([]int, 0, len(p.Intersections))
	ys := make([]int, 0, len(p.Intersections))
	for _, xy := range p.Intersections {
		in0 := dim0[xy[0]]
		if in0 == 0 {
			return nil, fmt.Errorf("%d not in dimensions[0]", xy[0])
		}
		in1 := dim1[xy[1]]
		if in1 == 0 {
			return nil, fmt.Errorf("%d not in dimensions[1]", xy[1])
		}
		xs = append(xs, in0 - 1)
		ys = append(ys, in1 - 1)
	}
	return &message.CurtainParams{ Dim0s: xs, Dim1s: ys }, nil
}

func (c *Curtain) Get(ctx *gin.Context) {
	pid := ctx.GetString("pid")
	guid := ctx.Param("guid")

	m, err := util.GetManifest(ctx, c.tokens, c.endpoint, guid)
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

	params, err := path.zeroindexed(m)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		return
	}

	authorization := ctx.GetHeader("Authorization")
	token, err := c.tokens.GetOnbehalf(authorization)
	if err != nil {
		// No further recovery is tried - GetManifest should already have fixed
		// a broken token, so this should be readily cached. If it is
		// just-about to expire then the process will fail pretty soon anyway,
		// so just give up.
		log.Printf("pid=%s, %v", pid, err)
		auth.AbortContextFromToken(ctx, err)
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
	msg := c.MakeTask(
		pid,
		guid,
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
		ctx.AbortWithStatus(http.StatusInternalServerError)
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
