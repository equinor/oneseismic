package controller

import (
	"encoding/json"
	"io/ioutil"
	"net/http/httptest"
	"testing"
	"os"

	"github.com/equinor/seismic-cloud/api/events"
	l "github.com/equinor/seismic-cloud/api/logger"
	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/stretchr/testify/assert"
)

func TestManifestControllerList(t *testing.T) {
	l.SetLogSink(os.Stdout, events.DebugLevel)
	manifest := NewTestManifest()
	ts := NewTestSetup()
	ts.AddManifest("12345", manifest)

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

	assert.Equal(t, readManifests[0], manifest)
}
