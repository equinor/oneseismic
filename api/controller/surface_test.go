package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/stretchr/testify/assert"
)

func TestSurfaceControllerUpload(t *testing.T) {
	surfaceData := []byte("blob blob, I'm a fish!\n")
	ts := NewTestSetup()
	req := httptest.NewRequest("POST", "/surface/testblob", ioutil.NopCloser(bytes.NewReader(surfaceData)))

	ts.BeginRequest(req)
	ts.SetParam("surfaceID", "testblob")

	ts.SurfaceController.Upload(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 200)

	ts.EndRequest()

	buf, err := ts.SurfaceStore.Download(context.Background(), "testblob")
	assert.Nil(t, err)

	gotSurface, err := ioutil.ReadAll(buf)
	assert.Nil(t, err)

	assert.Equal(t, gotSurface, surfaceData)
}

func TestSurfaceControllerList(t *testing.T) {
	surfaceData := []byte("blob blob, I'm a fish!\n")

	surfaces := make([]store.SurfaceMeta, 0)
	surfaces = append(surfaces, store.SurfaceMeta{
		SurfaceID: "blobtest",
		Link:      "blobtest",
	}, store.SurfaceMeta{
		SurfaceID: "blobtest_2",
		Link:      "blobtest_2",
	})

	ts := NewTestSetup()

	for _, ms := range surfaces {
		ts.AddSurface(ms.SurfaceID, "test-user", bytes.NewReader(surfaceData))
	}

	req := httptest.NewRequest("GET", "/surface", nil)
	ts.BeginRequest(req)

	ts.SurfaceController.List(ts.Ctx)
	assert.Equal(t, ts.Ctx.GetStatusCode(), 200)

	ts.EndRequest()

	gotSurfaces, err := ioutil.ReadAll(ts.Result().Body)
	assert.Nil(t, err)

	surf := make([]store.SurfaceMeta, 0)
	err = json.Unmarshal(gotSurfaces, &surf)
	assert.Nil(t, err)

	assert.Equal(t, surf, surfaces)
}

func TestSurfaceControllerDownload(t *testing.T) {
	surfaceData := []byte("blob blob, I'm a Fish!\n")

	ts := NewTestSetup()
	ts.AddSurface("blobtest", "test-user", bytes.NewReader(surfaceData))

	req := httptest.NewRequest("GET", "/surface/blobtest", nil)
	ts.BeginRequest(req)
	ts.SetParam("surfaceID", "blobtest")

	ts.SurfaceController.Download(ts.Ctx)

	ts.EndRequest()
	gotData, err := ioutil.ReadAll(ts.Result().Body)
	assert.Nil(t, err)

	assert.Equal(t, gotData, surfaceData)
}

func TestSurfaceControllerDownloadMissingSurface(t *testing.T) {
	ts := NewTestSetup()

	req := httptest.NewRequest("GET", "/surface/blobtest", nil)
	ts.BeginRequest(req)
	ts.SetParam("surfaceID", "blobtest")

	ts.SurfaceController.Download(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 404, "Should give not found status code")

	ts.EndRequest()
}
