package server

import (
	"log"
	"testing"

	"github.com/equinor/oneseismic/api/oneseismic"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestMakeSliceRequest(t *testing.T) {
	bytes, err := makeSliceRequest("", "", 0, 0, "", "")
	assert.Nil(t, err)
	sr := oneseismic.ApiRequest{}
	err = proto.Unmarshal(bytes, &sr)
	assert.Nil(t, err)
	assert.NotNil(t, sr.Shape)
	assert.NotNil(t, sr.GetSlice())
}

func TestSliceModel(t *testing.T) {
	jobs := make(chan job)

	go func() {
		job := <-jobs
		fr := oneseismic.FetchResponse{
			Requestid: job.jobID,
		}
		ar := oneseismic.ApiRequest{}
		err := proto.Unmarshal([]byte(job.request), &ar)
		if err == nil {
			fr.Function = &oneseismic.FetchResponse_Slice{
				Slice: &oneseismic.SliceResponse{},
			}
		}

		bytes, err := proto.Marshal(&fr)
		if err == nil {
			job.reply <- bytes
		} else {
			job.reply <- []byte("")
		}
	}()

	sl := slicer{storageURL: "%s.some.url", jobs: jobs}
	slice, err := sl.fetchSlice("root", "guid", 0, 0, "requestid", "token")

	assert.Nil(t, err)
	assert.NotNil(t, slice)
}

func TestModelMissingSlice(t *testing.T) {
	jobs := make(chan job)

	go func() {
		job := <-jobs
		fr := oneseismic.FetchResponse{}
		bytes, err := proto.Marshal(&fr)
		if err != nil {
			log.Fatalln("Failed to encode:", err)
		}
		job.reply <- bytes
	}()
	sl := slicer{storageURL: "%s.some.url", jobs: jobs}
	_, err := sl.fetchSlice("root", "guid", 0, 0, "requestid", "token")

	assert.NotNil(t, err)
}
