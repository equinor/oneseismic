package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/equinor/seismic-cloud-api/api/tests"
	"github.com/stretchr/testify/assert"
)

func TestManifestControllerFetchMissing(t *testing.T) {
	ts := tests.NewTestServiceSetup()
	req := httptest.NewRequest("GET", "/manifest/not-exists", nil)
	ts.BeginRequest(req)
	ts.SetParam("manifestID", "not-exists")
	mc := ManifestController{ts.ManifestStore}

	mc.Download(ts.Ctx)
	assert.Equal(t, ts.Ctx.GetStatusCode(), 404)

	ts.EndRequest()
}
