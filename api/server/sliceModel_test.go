package server

import (
	"log"
	"testing"

	"github.com/equinor/oneseismic/api/oneseismic"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

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

		job.io.out <- pr
		close(job.io.out)
		close(job.io.err)
	}()

	sl := slicer{
		endpoint: "",
		sessions: sessions,
	}
	slice, err := sl.fetchSlice("guid", 0, 0, "requestid", "token")

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
		job.io.out <- pr
		close(job.io.out)
		close(job.io.err)
	}()
	sl := slicer{
		endpoint: "",
		sessions: sessions,
	}
	_, err := sl.fetchSlice("guid", 0, 0, "requestid", "token")

	assert.NotNil(t, err)
}
