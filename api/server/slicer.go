package server

import (
	"net/http"
	"fmt"

	"github.com/equinor/oneseismic/api/oneseismic"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/kataras/iris/v12"
	"google.golang.org/protobuf/encoding/protojson"
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

	ctx.Header("Content-Type", "application/json")
	m := protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true}
	js, err := m.Marshal(slice)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}
	_, err = ctx.Write(js)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	return
}

type slicer struct {
	endpoint string
	sessions *sessions
}

func (s *slicer) fetchSlice(
	guid string,
	dim int32,
	lineno int32,
	requestid string,
	token string,
) (*oneseismic.SliceResponse, error) {

	msg := oneseismic.ApiRequest {
		Requestid:       requestid,
		Token:           token,
		Guid:            guid,
		StorageEndpoint: s.endpoint,
		Shape: &oneseismic.FragmentShape{
			Dim0: 64,
			Dim1: 64,
			Dim2: 64,
		},
		Function: &oneseismic.ApiRequest_Slice{
			Slice: &oneseismic.ApiSlice{
				Dim:    dim,
				Lineno: lineno,
			},
		},
	}

	req, err := proto.Marshal(&msg)
	if err != nil {
		return nil, fmt.Errorf("Marshalling oneseisimc.ApiRequest: %w", err)
	}

	proc := process{pid: requestid, request: req}
	fr := oneseismic.FetchResponse{}

	io := s.sessions.Schedule(&proc)

	/*
	 * Read and parse messages as they come, and consider the process complete
	 * when the reply-channel closes.
	 *
	 * Right now, the result is assembled here and returned in one piece to
	 * users, so it never looks like a parallelised job. This is so that we can
	 * experiment with chunk sizes, worker nodes, load etc. without having to
	 * be bothered with a more complex protocol between API and users, and so
	 * that previously-written clients still work. In the future, this will
	 * probably change and partial results will be transmitted.
	 *
	 * TODO: This gives weak failure handling, and Session needs a way to
	 * signal failed processes
	 */
	var tiles []*oneseismic.SliceTile
	for partial := range io.out {
		err = proto.Unmarshal(partial.payload, &fr)

		if err != nil {
			return nil, fmt.Errorf("could not create slice response: %w", err)
		}

		slice := fr.GetSlice()
		// TODO: cancel job on failure channel
		if slice == nil {
			switch x := fr.Function.(type) {
			default:
				msg := "%s Expected FetchResponse.Function = %T; was %T"
				log.Error().Msgf(msg, requestid, slice, x)
				return nil, fmt.Errorf("internal error")
			}
		}

		tiles = append(tiles, slice.GetTiles()...)
	}

	/*
	 * On successful runs, there are no messages on this channel, and the loop
	 * turns into a no-op.
	 */
	 for failure := range io.err {
		return nil, newFailure(failure)
	}

	fr.GetSlice().Tiles = tiles
	return fr.GetSlice(), nil
}
