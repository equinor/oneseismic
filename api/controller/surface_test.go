package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"

	l "github.com/equinor/seismic-cloud/api/logger"

	"github.com/equinor/seismic-cloud/api/service/store"
)

func NewSurfaceTestGetRequest(surfaceData []byte) (*http.Request, error) {
	return http.NewRequest("GET", "equinorseismiccloud.com", ioutil.NopCloser(bytes.NewReader(surfaceData)))
}

func TestSurfaceControllerUpload(t *testing.T) {

	ss, _ := NewTestingSurfaceStore()

	surfaceData := []byte("blob blob, I'm a fish!\n")
	surfaceID := "testblob"
	userID := "test-user"

	ssc := NewSurfaceController(ss)
	ctx := NewMockContext()

	req, _ := NewSurfaceTestGetRequest(surfaceData)
	ctx.BeginRequest(NewMockWriter(), req)
	ctx.Params().Set("userID", userID)
	ctx.Params().Set("surfaceID", surfaceID)

	ssc.Upload(ctx)

	if gotStatus := ctx.GetStatusCode(); gotStatus != 200 {
		t.Errorf("SurfaceController.Upload : Status = %v, want %v", gotStatus, 200)
	}

	buf, err := ss.Download(context.Background(), surfaceID)
	if err != nil {
		t.Errorf("surface store download failed err %v", err)
		return
	}

	ctx.EndRequest()

	gotSurface, err := ioutil.ReadAll(buf)
	if err != nil {
		t.Errorf("SurfaceController.Upload Readall err %v", err)
		return
	}
	if strings.HasSuffix(string(gotSurface), surfaceID) {
		t.Errorf("SurfaceController.Upload : SurfaceID = %v, want %v", string(gotSurface), surfaceID)
	}
}

func TestSurfaceControllerList(t *testing.T) {
	userID := "test-user"
	surfaceData := []byte("blob blob, I'm a Fish!\n")
	surfaces := make([]store.SurfaceMeta, 0)
	surfaces = append(surfaces, store.SurfaceMeta{
		SurfaceID: "blobtest",
		Link:      "blobtest",
	}, store.SurfaceMeta{
		SurfaceID: "blobtest_2",
		Link:      "blobtest_2",
	})

	ss, _ := NewTestingSurfaceStore()

	c := NewSurfaceController(ss)
	ctx := NewMockContext()

	writer := NewMockWriter()
	req, _ := NewSurfaceTestGetRequest(surfaceData)
	ctx.BeginRequest(writer, req)
	ctx.Params().Set("userID", userID)

	for _, ms := range surfaces {
		ss.Upload(context.Background(), ms.SurfaceID, userID, bytes.NewReader(surfaceData))
	}

	c.List(ctx)

	if gotStatus := ctx.GetStatusCode(); gotStatus != 200 {
		t.Errorf("SurfaceController.List : Status = %v, want %v", gotStatus, 200)
	}

	gotSurfaces, err := ioutil.ReadAll(writer.buffer)
	if err != nil {
		t.Errorf("SurfaceController.List Readall err %v", err)
		return
	}
	surf := make([]store.SurfaceMeta, 0)
	l.LogI("test", fmt.Sprintf("Got: %v\n", string(gotSurfaces)))
	err = json.Unmarshal(gotSurfaces, &surf)
	if err != nil {
		t.Errorf("SurfaceController.List json unmarshal err %v", err)
		return
	}

	if !reflect.DeepEqual(surf, surfaces) {
		t.Errorf("SurfaceController.List : Surfaces = %v, want %v", surf, surfaces)
	}

	ctx.EndRequest()
}

func TestSurfaceControllerDownload(t *testing.T) {
	userID := "test-user"
	surfaceData := []byte("blob blob, I'm a Fish!\n")
	surfaceID := "blobtest"

	ss, _ := NewTestingSurfaceStore()

	c := NewSurfaceController(ss)
	ctx := NewMockContext()

	writer := NewMockWriter()
	req, _ := NewSurfaceTestGetRequest(surfaceData)
	ctx.BeginRequest(writer, req)
	ctx.Params().Set("userID", userID)
	ctx.Params().Set("surfaceID", surfaceID)

	ss.Upload(context.Background(), surfaceID, userID, bytes.NewReader(surfaceData))

	c.Download(ctx)

	if gotStatus := ctx.GetStatusCode(); gotStatus != 200 {
		t.Errorf("SurfaceController.Download : Status = %v, want %v", gotStatus, 200)
	}

	gotData, err := ioutil.ReadAll(writer.buffer)
	if err != nil {
		t.Errorf("SurfaceController.Download Readall err %v", err)
		return
	}
	if !reflect.DeepEqual(gotData, surfaceData) {
		t.Errorf("SurfaceController.Download : data = %v, want %v", string(gotData), string(surfaceData))
	}

	ctx.EndRequest()

}
