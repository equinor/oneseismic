package controller

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func TestSurfaceControllerUpload(t *testing.T) {

	ss, _ := store.NewSurfaceStore(make(map[string][]byte))

	c := NewSurfaceController(ss)

	tests := []struct {
		name          string
		userID        string
		data          []byte
		wantSurfaceID string
		wantStatus    int
	}{
		{"Upload with userID", "exist", []byte("blob blob im a fish\n"), "blobtest", 200},
		{"Upload without userID", "", []byte("blob blob im a fish\n"), "blobtest", 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.NewContext(iris.Default())

			req := &http.Request{}
			r := bytes.NewReader(tt.data)
			req.Body = ioutil.NopCloser(r)
			buf := bytes.NewBuffer(make([]byte, 0))
			writer := NewMockWriter(buf)
			ctx.BeginRequest(writer, req)

			ctx.Params().Set("userID", tt.userID)

			_, err := io.Copy(ctx.ResponseWriter(), r)
			if err != nil {
				t.Errorf("SurfaceController.Upload : Could not make testfile")
			}

			c.Upload(ctx)

			if gotStatus := ctx.GetStatusCode(); !reflect.DeepEqual(gotStatus, tt.wantStatus) {
				t.Errorf("SurfaceController.Upload : Status = %v, want %v", gotStatus, tt.wantStatus)
			}
			gotSurfaceID, err := ioutil.ReadAll(buf)
			if err != nil {
				t.Errorf("SurfaceController.Upload Readall err %v", err)
				return
			}
			if strings.HasSuffix(string(gotSurfaceID), tt.wantSurfaceID) {
				t.Errorf("SurfaceController.Upload : SurfaceID = %v, want %v", string(gotSurfaceID), tt.wantSurfaceID)
			}

			ctx.EndRequest()
		})
	}
}

func TestSurfaceControllerList(t *testing.T) {

	mockSurfaces := make([]store.SurfaceMeta, 0)
	mockSurfaces = append(mockSurfaces, store.SurfaceMeta{
		SurfaceID: "blobtest",
		Link:      "blobtest",
	}, store.SurfaceMeta{
		SurfaceID: "blobtest_2",
		Link:      "blobtest_2",
	})

	ss, _ := store.NewSurfaceStore(map[string][]byte{
		"blobtest":   []byte("blobtest"),
		"blobtest_2": []byte("blobtest_2"),
	})

	c := NewSurfaceController(ss)

	tests := []struct {
		name         string
		wantSurfaces []store.SurfaceMeta
		wantStatus   int
	}{
		{"List", mockSurfaces, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.NewContext(iris.Default())

			req := &http.Request{}
			buf := bytes.NewBuffer(make([]byte, 0))
			writer := NewMockWriter(buf)
			ctx.BeginRequest(writer, req)

			c.List(ctx)

			if gotStatus := ctx.GetStatusCode(); !reflect.DeepEqual(gotStatus, tt.wantStatus) {
				t.Errorf("SurfaceController.List : Status = %v, want %v", gotStatus, tt.wantStatus)
			}
			gotSurfaces, err := ioutil.ReadAll(buf)
			if err != nil {
				t.Errorf("SurfaceController.List Readall err %v", err)
				return
			}

			surf := make([]store.SurfaceMeta, 0)
			err = json.Unmarshal(gotSurfaces, &surf)

			if !reflect.DeepEqual(surf, tt.wantSurfaces) {
				t.Errorf("SurfaceController.List : Surfaces = %v, want %v", surf, tt.wantSurfaces)
			}

			ctx.EndRequest()
		})
	}
}

func TestSurfaceControllerDownload(t *testing.T) {

	surfaces := map[string][]byte{"blobtest": []byte("surface")}
	ss, _ := store.NewSurfaceStore(surfaces)

	c := NewSurfaceController(ss)

	tests := []struct {
		name          string
		wantSurfaceID string
		wantData      []byte
		wantStatus    int
	}{
		{"Download with valid surfaceID", "blobtest", []byte("surface"), 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.NewContext(iris.Default())

			req := &http.Request{}
			r := bytes.NewReader(nil)
			req.Body = ioutil.NopCloser(r)
			buf := bytes.NewBuffer(make([]byte, 0))
			writer := NewMockWriter(buf)
			ctx.BeginRequest(writer, req)

			ctx.Params().Set("surfaceID", tt.wantSurfaceID)

			_, err := io.Copy(ctx.ResponseWriter(), r)
			if err != nil {
				t.Errorf("SurfaceController.Download : Could not make testfile")
			}

			c.Download(ctx)

			if gotStatus := ctx.GetStatusCode(); !reflect.DeepEqual(gotStatus, tt.wantStatus) {
				t.Errorf("SurfaceController.Download : Status = %v, want %v", gotStatus, tt.wantStatus)
			}
			gotData, err := ioutil.ReadAll(buf)
			if err != nil {
				t.Errorf("SurfaceController.Download Readall err %v", err)
				return
			}
			if !reflect.DeepEqual(gotData, tt.wantData) {
				t.Errorf("SurfaceController.Download : SurfaceID = %v, want %v", gotData, tt.wantData)
			}

			ctx.EndRequest()
		})
	}
}
