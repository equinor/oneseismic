package server

import (
	"fmt"
	"log"

	"github.com/equinor/oneseismic/api/oneseismic"
	"google.golang.org/protobuf/proto"
)

type messageMultiplexer interface {
	root() string
	endpoint() string
}

type slicer struct {
	mm messageMultiplexer
	sessions *sessions
}

func makeSliceRequest(
	storageEndpoint string,
	root string,
	guid string,
	dim int32,
	lineno int32,
	requestid string) ([]byte, error) {

	req := oneseismic.ApiRequest{
		Requestid:       requestid,
		Guid:            guid,
		Root:            root,
		StorageEndpoint: storageEndpoint,
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
	return proto.Marshal(&req)
}

type mMultiplexer struct {
	storageEndpoint string
	storageRoot     string
}

func (m *mMultiplexer) root() string         { return m.storageRoot }
func (m *mMultiplexer) endpoint() string     { return m.storageEndpoint }

func (s *slicer) fetchSlice(
	guid string,
	dim int32,
	lineno int32,
	requestid string) (*oneseismic.SliceResponse, error) {

	req, err := makeSliceRequest(s.mm.endpoint(), s.mm.root(), guid, dim, lineno, requestid)
	if err != nil {
		return nil, fmt.Errorf("could not make slice request: %w", err)
	}

	proc := process{pid: requestid, request: req}
	fr := oneseismic.FetchResponse{}

	replyChannel := s.sessions.Schedule(&proc)

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
	for partial := range replyChannel {
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
				log.Printf(msg, requestid, slice, x)
				return nil, fmt.Errorf("internal error")
			}
		}

		tiles = append(tiles, slice.GetTiles()...)
	}

	fr.GetSlice().Tiles = tiles
	return fr.GetSlice(), nil
}
