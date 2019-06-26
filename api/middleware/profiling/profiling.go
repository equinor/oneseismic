package profiling

import (
	"time"

	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/google/uuid"
	"github.com/kataras/iris/context"
)

func Init(ps store.ProfileStore) func(context.Context) {
	return func(ctx context.Context) {
		timeStart := time.Now()
		sessionID := uuid.New().String()
		ctx.ResponseWriter().Header().Set("Trailer", "Duration")
		ctx.ResponseWriter().Header().Set("Session-Id", sessionID)
		ctx.Next()

		deltaT := time.Since(timeStart)
		ctx.ResponseWriter().Header().Set("Duration", deltaT.String())
		ctx.OnClose(func() {

			ps.Append(sessionID,
				map[string]string{
					"sessionDuration": deltaT.String(),
					"stitchProfile":   ctx.Values().GetStringTrim("StitchInfo")})

		})
	}

}
