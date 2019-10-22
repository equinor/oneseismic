package controller

import (
	"encoding/json"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/stretchr/testify/assert"
)

func TestManifestControllerList(t *testing.T) {
	ts := NewTestSetup()

	req := httptest.NewRequest("GET", "/manifest", nil)
	ts.BeginRequest(req)

	ts.ManifestController.List(ts.Ctx)
	assert.Equal(t, ts.Ctx.GetStatusCode(), 200)

	ts.EndRequest()

	gotManifests, err := ioutil.ReadAll(ts.Result().Body)
	assert.Nil(t, err)

	readManifests := make([]store.Manifest, 0)
	err = json.Unmarshal(gotManifests, &readManifests)
	assert.Nil(t, err)
	expected := []store.Manifest{NewTestManifest(), NewTestManifest()}
	assert.Equal(t, expected, readManifests)
}

func TestManifestControllerFetch(t *testing.T) {
	manifest := NewTestManifest()
	ts := NewTestSetup()

	req := httptest.NewRequest("GET", "/manifest/12345", nil)
	ts.BeginRequest(req)
	ts.SetParam("manifestID", "12345")

	ts.ManifestController.Fetch(ts.Ctx)
	assert.Equal(t, ts.Ctx.GetStatusCode(), 200)

	ts.EndRequest()

	gotManifests, err := ioutil.ReadAll(ts.Result().Body)
	assert.Nil(t, err)

	readManifests := store.Manifest{}
	err = json.Unmarshal(gotManifests, &readManifests)
	assert.Nil(t, err)

	assert.Equal(t, manifest, readManifests)
}

func TestManifestControllerFetchMissing(t *testing.T) {
	ts := NewTestSetup()
	req := httptest.NewRequest("GET", "/manifest/notexist", nil)
	ts.BeginRequest(req)
	ts.SetParam("manifestID", "notexist")

	ts.ManifestController.Fetch(ts.Ctx)
	assert.Equal(t, ts.Ctx.GetStatusCode(), 404)

	ts.EndRequest()
}
