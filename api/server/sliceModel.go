package server

import (
	"fmt"

	"github.com/equinor/oneseismic/api/oneseismic"
	"google.golang.org/protobuf/proto"
)

type messageMultiplexer interface {
	jobChannel() chan job
	root() string
	endpoint() string
}

type slicer struct {
	mm messageMultiplexer
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
	jobs            chan job
}

func (m *mMultiplexer) root() string         { return m.storageRoot }
func (m *mMultiplexer) jobChannel() chan job { return m.jobs }
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

	replyChannel := make(chan []byte)
	job := job{requestid, req, replyChannel}

	fr := oneseismic.FetchResponse{}

	s.mm.jobChannel() <- job
	err = proto.Unmarshal([]byte(<-replyChannel), &fr)
	if err != nil {
		return nil, fmt.Errorf("could not create slice response: %w", err)
	}

	slice := fr.GetSlice()
	if slice == nil {
		return nil, fmt.Errorf("slice not found")

	}

	return fr.GetSlice(), nil
}
