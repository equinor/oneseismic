package controller

import (
	"github.com/equinor/seismic-cloud/api/service/store"

	"github.com/kataras/iris"
)

// @Description get profiling numbers
// @Produce  application/json
// @Param   some_id     path    string     true        "Some ID"
// @Success 200 {file} controller.fileBytes OK
// @Failure 404 {object} controller.APIError "Profile not found"
// @Failure 500 {object} controller.APIError "Internal Server Error"
// @Router /profile/{profile_id} [get]
func ProfileController(ps store.ProfileStore) func(ctx iris.Context) {

	return func(ctx iris.Context) {
		profileID := ctx.Params().Get("profileID")
		pd, err := ps.Fetch(profileID)
		if err != nil {
			ctx.NotFound()
			return
		}
		ctx.JSON(pd)
		return
	}
}
