package controller

import (
	"bytes"

	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/equinor/seismic-cloud/api/tests"

	"github.com/stretchr/testify/assert"
)

func TestSurfaceControllerUpload(t *testing.T) {
	ts := tests.NewTestServiceSetup()
	surfaceData := []byte("blob blob, I'm a fish!\n")
	req := httptest.NewRequest("POST", "/surface/testblob", ioutil.NopCloser(bytes.NewReader(surfaceData)))

	ts.BeginRequest(req)
	ts.SetParam("surfaceID", "testblob")
	sc := SurfaceController{ts.SurfaceStore}
	sc.Upload(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 200)

	ts.EndRequest()

}

func TestSurfaceControllerList(t *testing.T) {
	ts := tests.NewTestServiceSetup()

	req := httptest.NewRequest("GET", "/surface", nil)
	ts.BeginRequest(req)
	sc := SurfaceController{ts.SurfaceStore}
	sc.List(ts.Ctx)
	assert.Equal(t, ts.Ctx.GetStatusCode(), 200)

	ts.EndRequest()

}

func TestSurfaceControllerDownload(t *testing.T) {
	ts := tests.NewTestServiceSetup()

	req := httptest.NewRequest("GET", "/surface/blobtest", nil)
	ts.BeginRequest(req)
	ts.SetParam("surfaceID", "blobtest")
	sc := SurfaceController{ts.SurfaceStore}
	sc.Download(ts.Ctx)

	ts.EndRequest()

}

func TestSurfaceControllerDownloadMissingSurface(t *testing.T) {
	ts := tests.NewTestServiceSetup()

	req := httptest.NewRequest("GET", "/surface/not-exists", nil)
	ts.BeginRequest(req)
	ts.SetParam("surfaceID", "not-exists")
	sc := SurfaceController{ts.SurfaceStore}
	sc.Download(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 404, "Should give not found status code")

	ts.EndRequest()
}
