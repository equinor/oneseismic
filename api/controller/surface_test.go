package controller

import (
	"bytes"
	gocontext "context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

type MockSurface struct{}

type MockSurfaceStore struct{}

func mockSurfaces() []store.Surface {
	surfaces := make([]store.Surface, 0)
	surfaces = append(surfaces, store.Surface{
		SurfaceID:    "blobtest",
		Link:         "azure.container.blobstore",
		LastModified: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)})
	surfaces = append(surfaces, store.Surface{
		SurfaceID:    "blobtest_2",
		Link:         "azure.container.blobstore",
		LastModified: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)})
	return surfaces
}

func (*MockSurfaceStore) List(gocontext.Context) ([]store.Surface, error) {
	return mockSurfaces(), nil
}
func (*MockSurfaceStore) Download(gocontext.Context, string) (io.Reader, error) {
	return bytes.NewReader([]byte("surface")), nil
}
func (*MockSurfaceStore) Upload(gocontext.Context, string, string, io.Reader) (string, error) {
	return "blobtest", nil
}

func TestSurfaceControllerUpload(t *testing.T) {

	store := &MockSurfaceStore{}
	c := NewSurfaceController(store)

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

	mockstore := &MockSurfaceStore{}
	c := NewSurfaceController(mockstore)

	tests := []struct {
		name         string
		wantSurfaces []store.Surface
		wantStatus   int
	}{
		{"List", mockSurfaces(), 200},
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

			surf := make([]store.Surface, 0)
			err = json.Unmarshal(gotSurfaces, &surf)

			if !reflect.DeepEqual(surf, tt.wantSurfaces) {
				t.Errorf("SurfaceController.List : Surfaces = %v, want %v", surf, tt.wantSurfaces)
			}

			ctx.EndRequest()
		})
	}
}

func TestSurfaceControllerDownload(t *testing.T) {

	store := &MockSurfaceStore{}
	c := NewSurfaceController(store)

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
