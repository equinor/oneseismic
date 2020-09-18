package server

import (
	"log"
	"testing"

	"github.com/equinor/oneseismic/api/oneseismic"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

type sliceMock struct {
	slices map[string]oneseismic.SliceResponse
}

func (s *sliceMock) fetchSlice(
	guid string,
	dim int32,
	lineno int32,
	requestid string,
	token string,
) (*oneseismic.SliceResponse, error) {
	l, ok := s.slices[guid]
	if ok {
		return &l, nil
	}
	return nil, newFailure("manifest-not-found")
}

func TestExistingSlice(t *testing.T) {
	app := iris.Default()
	app.Use(mockOboJWT())

	m := map[string]oneseismic.SliceResponse{"some_guid": {}}
	sc := &sliceController{&sliceMock{m}}

	app.Get("/{guid:string}/slice/{dim:int32}/{lineno:int32}", sc.get)

	e := httptest.New(t, app)
	e.GET("/some_guid/slice/0/0").
		Expect().
		Status(httptest.StatusOK).
		JSON()
}

func TestMissingSlice(t *testing.T) {
	app := iris.Default()
	app.Use(mockOboJWT())

	sc := &sliceController{&sliceMock{}}
	app.Get("/{guid:string}/slice/{dim:int32}/{lineno:int32}", sc.get)

	e := httptest.New(t, app)
	e.GET("/some_other_guid/slice/0/0").
		Expect().
		Status(httptest.StatusNotFound)
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
