package controller

import (
	"bytes"
	goctx "context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/equinor/seismic-cloud/api/events"
	l "github.com/equinor/seismic-cloud/api/logger"
	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/kataras/iris"
	irisCtx "github.com/kataras/iris/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func NewTestingSurfaceStore() (store.SurfaceStore, error) {
	return store.NewSurfaceStore(map[string][]byte{})
}

func NewTestingManifestStore(m map[string]store.Manifest) (store.ManifestStore, error) {
	return store.NewManifestStore(m)
}

func NewTestingContext() irisCtx.Context {
	return irisCtx.NewContext(iris.Default())
}

type MockStitch struct {
	mock.Mock
}

func (m MockStitch) Stitch(ctx goctx.Context, ms store.Manifest, out io.Writer, in io.Reader) (string, error) {
	_, err := io.Copy(out, in)
	args := m.Called(ctx, ms, out, in)
	if err != nil {
		return "", err
	}
	return args.String(0), args.Error(1)
}

func NewSurfaceTestGetRequest(surfaceData []byte) *http.Request {
	return httptest.NewRequest("GET", "/surface", ioutil.NopCloser(bytes.NewReader(surfaceData)))
}

func NewTestManifest() store.Manifest {
	return store.Manifest{
		Basename:   "mock",
		Cubexs:     1,
		Cubeys:     1,
		Cubezs:     1,
		Fragmentxs: 1,
		Fragmentys: 1,
		Fragmentzs: 1,
	}
}

type TestSetup struct {
	SurfaceStore      store.SurfaceStore
	ManifestStore     store.ManifestStore
	SurfaceController *SurfaceController
	Manifests         map[string]store.Manifest
	Stitch            MockStitch
	Ctx               irisCtx.Context
	Recorder          *httptest.ResponseRecorder
}

func NewTestSetup() *TestSetup {
	stitch := MockStitch{}
	ctx := NewTestingContext()
	manifests := map[string]store.Manifest{}
	ms, _ := NewTestingManifestStore(manifests)
	recorder := httptest.NewRecorder()
	ss, _ := NewTestingSurfaceStore()
	c := NewSurfaceController(ss)

	return &TestSetup{ss, ms, c, manifests, stitch, ctx, recorder}
}

func (ts *TestSetup) AddManifest(manifestID string, m store.Manifest) {
	ts.Manifests[manifestID] = m
}

func (ts *TestSetup) AddSurface(surfaceID string, userID string, surface io.Reader) {
	ts.SurfaceStore.Upload(goctx.Background(), surfaceID, userID, surface)
}

func (ts *TestSetup) BeginRequest(r *http.Request) {
	r.ParseForm()
	ts.Ctx.BeginRequest(ts.Recorder, r)
}

func (ts *TestSetup) EndRequest() {
	ts.Ctx.EndRequest()
}

func (ts *TestSetup) SetParam(id string, v string) {
	ts.Ctx.Params().Set(id, v)
}

func (ts *TestSetup) OnStitch(v ...interface{}) *mock.Call {
	return ts.Stitch.On("Stitch", v...)
}

func (ts *TestSetup) Result() *http.Response {
	return ts.Recorder.Result()
}

func TestStitchControllerSuccess(t *testing.T) {
	l.SetLogSink(os.Stdout, events.DebugLevel)

	manifest := NewTestManifest()
	ts := NewTestSetup()
	ts.AddManifest("12345", manifest)
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

	StitchController(ts.ManifestStore, ts.Stitch)(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 200, "Should give ok status code")
	got, _ := ioutil.ReadAll(ts.Result().Body)
	assert.Equal(t, string(got), have, "garbage in, garbage out")

	ts.EndRequest()
}

func TestStitchMissingManifestNotFoundCode(t *testing.T) {
	ts := NewTestSetup()

	req := httptest.NewRequest("POST", "/stitch/12345", ioutil.NopCloser(strings.NewReader("")))
	ts.BeginRequest(req)
	ts.SetParam("manifestID", "12345")

	StitchController(ts.ManifestStore, ts.Stitch)(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 404, "Should give not found status code")

	ts.EndRequest()
}

func TestStitchControllerStitchError(t *testing.T) {
	manifest := NewTestManifest()
	ts := NewTestSetup()
	ts.AddManifest("12345", manifest)
	ts.OnStitch(mock.Anything, manifest, mock.Anything, mock.Anything).Return("", errors.New("Stitch failed"))

	req := httptest.NewRequest("POST", "/stitch/12345", ioutil.NopCloser(strings.NewReader("")))
	ts.BeginRequest(req)
	ts.SetParam("manifestID", "12345")

	StitchController(ts.ManifestStore, ts.Stitch)(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 500, "Should give internal Error")

	ts.EndRequest()
}

func TestStitchSurfaceControllerSuccess(t *testing.T) {
	manifest := NewTestManifest()

	ts := NewTestSetup()
	ts.AddSurface("surf-1", "test-user", strings.NewReader("SURFACE"))
	ts.AddManifest("man-1", manifest)
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
	manifest := NewTestManifest()

	ts := NewTestSetup()
	ts.AddSurface("surf-1", "test-user", strings.NewReader("SURFACE"))
	ts.OnStitch(mock.Anything, manifest, mock.Anything, mock.Anything).Return("", nil)

	req := httptest.NewRequest("POST", "/stitch/man-1/surf-1", ioutil.NopCloser(strings.NewReader("")))
	ts.BeginRequest(req)
	ts.SetParam("manifestID", "man-1")
	ts.SetParam("surfaceID", "surf-1")

	StitchSurfaceController(ts.ManifestStore, ts.SurfaceStore, ts.Stitch)(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 404, "Should give not found status code")

	ts.EndRequest()
}

func TestStitchSurfaceControllerNoSurface(t *testing.T) {
	manifest := NewTestManifest()

	ts := NewTestSetup()
	ts.AddManifest("man-1", manifest)
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
	manifest := NewTestManifest()

	ts := NewTestSetup()
	ts.AddSurface("surf-1", "test-user", strings.NewReader("SURFACE"))
	ts.AddManifest("man-1", manifest)
	ts.OnStitch(mock.Anything, manifest, mock.Anything, mock.Anything).Return("", errors.New("Stitch failed"))

	req := httptest.NewRequest("POST", "/stitch/man-1/surf-1", ioutil.NopCloser(strings.NewReader("")))
	ts.BeginRequest(req)
	ts.SetParam("manifestID", "man-1")
	ts.SetParam("surfaceID", "surf-1")

	StitchSurfaceController(ts.ManifestStore, ts.SurfaceStore, ts.Stitch)(ts.Ctx)

	assert.Equal(t, ts.Ctx.GetStatusCode(), 500, "Should give server error status code")

	ts.EndRequest()
}
