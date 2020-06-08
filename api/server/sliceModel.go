package server

import (
	"fmt"

	"github.com/equinor/oneseismic/api/oneseismic"
	"google.golang.org/protobuf/proto"
)

type sliceModel interface {
	fetchSlice(
		root string,
		guid string,
		dim int32,
		lineno int32,
		requestid string,
		token string) (*oneseismic.SliceResponse, error)
}

type slicer struct {
	jobs       chan job
	storageURL string
}

func makeSliceRequest(
	storageEndpoint string,
	guid string,
	dim int32,
	lineno int32,
	requestid string,
	token string,
) ([]byte, error) {

	req := oneseismic.ApiRequest{
		Requestid: requestid,
		Guid:      guid,
		Root:      storageEndpoint,
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

func (s *slicer) fetchSlice(
	root string,
	guid string,
	dim int32,
	lineno int32,
	requestid string,
	token string,
) (*oneseismic.SliceResponse, error) {
	_, err := parseStorageURL(root, s.storageURL)
	if err != nil {
		return nil, err
	}
	req, err := makeSliceRequest(root, guid, dim, lineno, requestid, token)
	if err != nil {
		return nil, fmt.Errorf("could not make slice request: %w", err)
	}

	replyChannel := make(chan []byte)
	job := job{requestid, req, replyChannel}

	fr := oneseismic.FetchResponse{}

	s.jobs <- job
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
