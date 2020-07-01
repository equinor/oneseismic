package server

import (
	"log"
	"testing"

	"github.com/equinor/oneseismic/api/oneseismic"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestMakeSliceRequest(t *testing.T) {
	bytes, err := makeSliceRequest("", "", "", 0, 0, "")
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
	sl := slicer{&mMultiplexer{"", "", jobs}}
	slice, err := sl.fetchSlice("guid", 0, 0, "requestid")

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
	sl := slicer{&mMultiplexer{"", "", jobs}}
	_, err := sl.fetchSlice("guid", 0, 0, "requestid")

	assert.NotNil(t, err)
}
