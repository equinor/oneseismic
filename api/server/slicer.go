package server

import (
	"encoding/binary"
	"fmt"
	"net/http"

	"github.com/equinor/oneseismic/api/oneseismic"
	"github.com/google/uuid"
	"github.com/kataras/iris/v12"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

type failure struct {
	key string
}

// TODO: maybe test these keys to ensure they're always in sync
// TODO: rather add code where these errors are generated?
// TODO: The failure [][]byte will then add a []byte for code
func (f *failure) statusCode() int {
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

type sliceModel interface {
	fetchSlice(
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
	log.Trace().Msgf("%v: new request", requestid)
	io, err := sc.slicer.fetchSlice(guid, dim, lineno, requestid, token)
	if err != nil {
		log.Error().Err(err)
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	for msg := range io.err {
		f := failure{msg}
		ctx.StatusCode(f.statusCode())
		return
	}

	ctx.Header("Content-Type", "application/x-protobuf")
	ctx.Header("Transfer-Encoding", "chunked")

	for partial := range io.out {
		fr := oneseismic.FetchResponse{}
		err = proto.Unmarshal(partial.payload, &fr)
		slice := fr.GetSlice()
		//TODO partial.n could be sent to the client to message the number of
		//chunks to get. Then the client can know if all has been transferred
		//or there has been an error

		bytes, err := proto.Marshal(slice)
		if err != nil {
			ctx.StatusCode(http.StatusInternalServerError)
			return
		}
		bs := make([]byte, 4)
		binary.LittleEndian.PutUint32(bs, uint32(len(bytes)))
		ctx.Write(bs)
		ctx.Write(bytes)
		ctx.ResponseWriter().Flush()
	}

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
) (*procIO, error) {

	msg := oneseismic.ApiRequest{
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
	log.Trace().Msgf("%v: scheduling", requestid)
	io := s.sessions.schedule(requestid, req)
	return &io, nil
}
