package controller

import (
	"errors"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStitchControllerSuccess(t *testing.T) {
	ts := NewTestSetup()
	manifest := NewTestManifest()
	ts.OnStitch(mock.Anything, manifest, mock.Anything, mock.Anything).Return("", nil)

	have :=
		`VLiFrhfjz7O5Zt1VD0Wd
		MBECw6JWO0oEsbkz4Qqv
		pEHK1urgtb8SC5gGs3po
		D5wzMivWXHiDvqHIKE4s
		djHkWdeZUB8JsacIhbnK
		HoTYPAQZ7ZoXAL2YVvoT
		j1sDu7eF9m1DNXFBy5cf
		TiAdXYPNBfNkqzi5nBRk
		S0wpZgBZYp5HK1dCF9sL
		kcmmZTNurGRSYkOJS9xn`
	req := httptest.NewRequest("POST", "/stitch/12345", ioutil.NopCloser(strings.NewReader(have)))
	ts.BeginRequest(req)
	ts.SetParam("manifestID", "12345")
	st := StitchController(ts.ManifestStore, ts.Stitch)
	st(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 200, "Should give ok status code")
	got, _ := ioutil.ReadAll(ts.Result().Body)
	assert.Equal(t, string(got), have, "garbage in, garbage out")

	ts.EndRequest()
}

func TestStitchMissingManifestNotFoundCode(t *testing.T) {
	ts := NewTestSetup()
	req := httptest.NewRequest("POST", "/stitch/notexist", ioutil.NopCloser(strings.NewReader("")))
	ts.BeginRequest(req)
	ts.SetParam("manifestID", "notexist")

	StitchController(ts.ManifestStore, ts.Stitch)(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 404, "Should give not found status code")

	ts.EndRequest()
}

func TestStitchControllerStitchError(t *testing.T) {
	ts := NewTestSetup()
	manifest := NewTestManifest()
	ts.OnStitch(mock.Anything, manifest, mock.Anything, mock.Anything).Return("", errors.New("Stitch failed"))

	req := httptest.NewRequest("POST", "/stitch/12345", ioutil.NopCloser(strings.NewReader("")))
	ts.BeginRequest(req)
	ts.SetParam("manifestID", "12345")

	StitchController(ts.ManifestStore, ts.Stitch)(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 500, "Should give internal Error")

	ts.EndRequest()
}

func TestStitchSurfaceControllerSuccess(t *testing.T) {
	ts := NewTestSetup()
	manifest := NewTestManifest()

	ts.AddSurface("surf-1", "test-user", strings.NewReader("SURFACE"))
	ts.OnStitch(mock.Anything, manifest, mock.Anything, mock.Anything).Return("", nil)

	req := httptest.NewRequest("POST", "/stitch/man-1/surf-1", ioutil.NopCloser(strings.NewReader("")))
	ts.BeginRequest(req)
	ts.SetParam("manifestID", "man-1")
	ts.SetParam("surfaceID", "surf-1")

	StitchSurfaceController(ts.ManifestStore, ts.SurfaceStore, ts.Stitch)(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 200, "Should give ok status code")

	ts.EndRequest()
}

func TestStitchSurfaceControllerNoManifest(t *testing.T) {
	ts := NewTestSetup()

	manifest := NewTestManifest()

	ts.AddSurface("surf-1", "test-user", strings.NewReader("SURFACE"))
	ts.OnStitch(mock.Anything, manifest, mock.Anything, mock.Anything).Return("", nil)

	req := httptest.NewRequest("POST", "/stitch/notexist/surf-1", ioutil.NopCloser(strings.NewReader("")))
	ts.BeginRequest(req)
	ts.SetParam("manifestID", "notexist")
	ts.SetParam("surfaceID", "surf-1")

	StitchSurfaceController(ts.ManifestStore, ts.SurfaceStore, ts.Stitch)(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 404, "Should give not found status code")

	ts.EndRequest()
}

func TestStitchSurfaceControllerNoSurface(t *testing.T) {
	ts := NewTestSetup()
	manifest := NewTestManifest()

	ts.OnStitch(mock.Anything, manifest, mock.Anything, mock.Anything).Return("", nil)

	req := httptest.NewRequest("POST", "/stitch/man-1/surf-1", ioutil.NopCloser(strings.NewReader("")))
	ts.BeginRequest(req)
	ts.SetParam("manifestID", "man-1")
	ts.SetParam("surfaceID", "surf-1")

	StitchSurfaceController(ts.ManifestStore, ts.SurfaceStore, ts.Stitch)(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 404, "Should give not found status code")

	ts.EndRequest()
}

func TestStitchSurfaceControllerStitchFailed(t *testing.T) {
	ts := NewTestSetup()
	manifest := NewTestManifest()

	ts.AddSurface("surf-1", "test-user", strings.NewReader("SURFACE"))
	ts.OnStitch(mock.Anything, manifest, mock.Anything, mock.Anything).Return("", errors.New("Stitch failed"))

	req := httptest.NewRequest("POST", "/stitch/man-1/surf-1", ioutil.NopCloser(strings.NewReader("")))
	ts.BeginRequest(req)
	ts.SetParam("manifestID", "man-1")
	ts.SetParam("surfaceID", "surf-1")

	StitchSurfaceController(ts.ManifestStore, ts.SurfaceStore, ts.Stitch)(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 500, "Should give server error status code")

	ts.EndRequest()
}
