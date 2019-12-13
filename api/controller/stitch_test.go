package controller

import (
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/equinor/seismic-cloud-api/api/tests"
	"github.com/stretchr/testify/assert"
)

func TestStitchMissingManifestNotFoundCode(t *testing.T) {
	ts := tests.NewTestServiceSetup()
	req := httptest.NewRequest("POST", "/stitch/not-exists", ioutil.NopCloser(strings.NewReader("")))
	ts.BeginRequest(req)
	ts.SetParam("manifestID", "not-exists")

	StitchSurfaceController(ts.ManifestStore, ts.Stitch)(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 404, "Should give not found status code")

	ts.EndRequest()
}

func TestStitchSurfaceControllerNoManifest(t *testing.T) {
	ts := tests.NewTestServiceSetup()

	req := httptest.NewRequest("GET", "/stitch/not-exists/surf-1", ioutil.NopCloser(strings.NewReader("")))
	ts.BeginRequest(req)
	ts.SetParam("manifestID", "not-exists")
	ts.SetParam("surfaceID", "surf-1")

	StitchSurfaceController(ts.ManifestStore, ts.Stitch)(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 404, "Should give not found status code")

	ts.EndRequest()
}
