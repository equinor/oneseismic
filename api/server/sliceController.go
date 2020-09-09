package server

import (
	"net/http"

	"github.com/equinor/oneseismic/api/oneseismic"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/kataras/iris/v12"
	"google.golang.org/protobuf/proto"
)

type failure struct {
	key string
}

// TODO: maybe test these keys to ensure they're always in sync
func (f *failure) status() int {
	switch f.key {
	case "manifest-not-found":
		return http.StatusNotFound

	case "fragment-not-found":
		/*
		 * If the fragment cannot be found, but the manifest already scheduled
		 * the process, something has gone wrong in the upload or maintainance
		 * of storage, and nothing is really wrong with the request itself, so
		 * this should probably be carefully investigated.
		 */
		return http.StatusInternalServerError

	case "bad-message":
		return http.StatusInternalServerError

	/*
	 * Azure blobs return 403 Forbidden also when the authentication string is
	 * broken, so there's no reasonable way for us to distinguish forbidden
	 * (access not allowed) and just a badly-formatted auth. This is something
	 * to maybe handle in the future, but is put on a TODO for now.
	 */
	case "manifest-not-authorized":
		return http.StatusUnauthorized

	case "fragment-not-authorized":
		return http.StatusUnauthorized

	default:
		log.Error().Msgf("unknown failure; key = %s", f.key)
		return http.StatusInternalServerError
	}
}

func (f *failure) Error() string {
	return f.key
}

func newFailure(key string) *failure {
	return &failure{key: key}
}

type sliceModel interface {
	fetchSlice(
		guid string,
		dim int32,
		lineno int32,
		requestid string,
		token string,
) (*oneseismic.SliceResponse, error)
}

type sliceController struct {
	slicer sliceModel
}

func (sc *sliceController) get(ctx iris.Context) {
	token, ok := ctx.Values().Get("jwt").(string)
	if !ok {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	guid := ctx.Params().GetString("guid")
	if len(guid) == 0 {
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	dim, err := ctx.Params().GetInt32("dim")
	if err != nil {
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	lineno, err := ctx.Params().GetInt32("lineno")
	if err != nil {
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	requestid := uuid.New().String()
	slice, err := sc.slicer.fetchSlice(guid, dim, lineno, requestid, token)
	if err != nil {
		switch e := err.(type) {
		case *failure:
			ctx.StatusCode(e.status())

		default:
			log.Error().Err(e)
			ctx.StatusCode(http.StatusInternalServerError)
		}
		return
	}

	ctx.Header("Content-Type", "application/x-protobuf")
	bytes, err := proto.Marshal(slice)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}
	_, err = ctx.Write(bytes)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	return
}
