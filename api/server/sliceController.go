package server

import (
	"fmt"
	"net/http"

	"github.com/equinor/oneseismic/api/oneseismic"
	"github.com/google/uuid"
	"github.com/kataras/golog"
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
		golog.Errorf("unknown failure; key = %s", f.key)
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
	io, err := sc.slicer.fetchSlice(guid, dim, lineno, requestid, token)
	if err != nil {
		switch e := err.(type) {
		case *failure:
			ctx.StatusCode(e.status())

		default:
			golog.Error(e)
			ctx.StatusCode(http.StatusInternalServerError)
		}
		return
	}

	ctx.Header("Content-Type", "application/json")
	ctx.Header("Transfer-Encoding", "chunked")
	m := protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true}

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
	 * TODO: update comment above
	 *
	 * The memory usage when marshalling the entire FetchResponse to json is too large
	 * Rather send each slice in chunks to the client
	 *
	 * TODO: This gives weak failure handling, and Session needs a way to
	 * signal failed processes
	 */

	 // TODO: handle errors 
	// for err := range io.err {
	// 	if err == "404" {
	// 		ctx.StatusCode(http.StatusNotFound)
	// 		return
	// 	}
	// 	ctx.StatusCode(http.StatusInternalServerError)
	// 	return
	// }

	ctx.WriteString("[")
	first := true
	for partial := range io.out {
		fr := oneseismic.FetchResponse{}
		err = proto.Unmarshal(partial.payload, &fr)
		slice := fr.GetSlice()
		js, err := m.Marshal(slice)
		if err != nil {
			ctx.StatusCode(http.StatusInternalServerError)
			return
		}
		if first {
			ctx.Write(js)
			first = false
		} else {
			ctx.WriteString(",")
			ctx.Write(js)
		}
		ctx.ResponseWriter().Flush()
	}
	ctx.WriteString("]")
}

type slicer struct {
	root     string
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
		Root:            s.root,
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
