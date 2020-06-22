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
	s.jobs <- job{requestid, req, replyChannel}

	fr := oneseismic.FetchResponse{}
	err = proto.Unmarshal([]byte(<-replyChannel), &fr)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal slice response: %w", err)
	}

	return fr.GetSlice(), nil
}
