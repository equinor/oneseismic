package profiling

import (
	"time"

	"github.com/kataras/iris/context"
)

func Duration(ctx context.Context) {
	timeStart := time.Now()
	ctx.ResponseWriter().Header().Set("Trailer", "Duration")

	ctx.Next()

	deltaT := time.Since(timeStart)
	ctx.ResponseWriter().Header().Set("Duration", deltaT.String())
}
