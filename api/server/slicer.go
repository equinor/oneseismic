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
	getProcIO(
		guid string,
		dim int32,
		lineno int32,
		requestid string,
		token string,
) (*procIO, error)
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
	io, err := sc.slicer.getProcIO(guid, dim, lineno, requestid, token)
	if err != nil {
		log.Error().Err(err)
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	for failure := range io.err {
		e := newFailure(failure)
		ctx.StatusCode(e.status())
		ctx.WriteString(failure)
		return
	}

	ctx.Header("Content-Type", "application/json")
	ctx.Header("Transfer-Encoding", "chunked,gzip")
	_, err = ctx.WriteString("[")

	m := protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true}

	count := 0

	for partial := range io.out {
		expectedCount := partial.m
		if count != 0 {
			_, err = ctx.WriteString(",")
		}

		fr := oneseismic.FetchResponse{}

		err = proto.Unmarshal(partial.payload, &fr)
		if err != nil {
			ctx.StatusCode(http.StatusInternalServerError)
			msg := fmt.Sprintf("could not create FetchResponse: %v", err.Error())
			log.Error().Err(err).Msg(msg)
			ctx.WriteString(msg)
			return
		}

		slice := fr.GetSlice()
		if slice == nil {
			switch x := fr.Function.(type) {
			default:
				msg := "%s Expected FetchResponse.Function = %T; was %T"
				log.Error().Msgf(msg, requestid, slice, x)
				ctx.StatusCode(http.StatusInternalServerError)
				ctx.WriteString("internal error")
				return
			}
		}

		js, err := m.Marshal(fr.GetSlice())
		if err != nil {
			log.Error().Err(err)
			return
		}
		_, err = ctx.WriteString(string(js))
		if err != nil {
			log.Error().Err(err)
			return
		}
		if count == expectedCount -1 {
			_, err = ctx.WriteString("]")
		}
		ctx.ResponseWriter().Flush()
		count = count +1
	}

	return
}

type slicer struct {
	endpoint string
	sessions *sessions
}

func (s *slicer) getProcIO(
	guid string,
	dim int32,
	lineno int32,
	requestid string,
	token string,
) (*procIO, error) {

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

	io := s.sessions.Schedule(&proc)
	return &io, nil
}
