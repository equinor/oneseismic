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
	sessions := newSessions()

	go func() {
		job := <-sessions.queue
		fr := oneseismic.FetchResponse{
			Requestid: job.pid,
		}
		ar := oneseismic.ApiRequest{}
		err := proto.Unmarshal([]byte(job.request), &ar)
		if err == nil {
			fr.Function = &oneseismic.FetchResponse_Slice{
				Slice: &oneseismic.SliceResponse{},
			}
		}

		bytes, _ := proto.Marshal(&fr)
		pr := partialResult {
			pid: "1",
			n: 0,
			m: 1,
			payload: bytes,
		}

		job.reply <- pr
		close(job.reply)
	}()

	sl := slicer{&mMultiplexer{"", ""}, sessions}
	slice, err := sl.fetchSlice("guid", 0, 0, "requestid")

	assert.Nil(t, err)
	assert.NotNil(t, slice)
}

func TestModelMissingSlice(t *testing.T) {
	sessions := newSessions()

	go func() {
		job := <-sessions.queue
		fr := oneseismic.FetchResponse{}
		bytes, err := proto.Marshal(&fr)
		if err != nil {
			log.Fatalln("Failed to encode:", err)
		}
		pr := partialResult {
			pid: "2",
			n: 0,
			m: 1,
			payload: bytes,
		}
		job.reply <- pr
		close(job.reply)
	}()
	sl := slicer{&mMultiplexer{"", ""}, sessions}
	_, err := sl.fetchSlice("guid", 0, 0, "requestid")

	assert.NotNil(t, err)
}
