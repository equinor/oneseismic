package controller

import (
	"bytes"
	goctx "context"
	"github.com/equinor/seismic-cloud/api/events"
	l "github.com/equinor/seismic-cloud/api/logger"
	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/kataras/iris"
	irisCtx "github.com/kataras/iris/context"
	"github.com/stretchr/testify/mock"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
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
	SurfaceStore       store.SurfaceStore
	ManifestStore      store.ManifestStore
	SurfaceController  *SurfaceController
	ManifestController *ManifestController
	Manifests          map[string]store.Manifest
	Stitch             MockStitch
	Ctx                irisCtx.Context
	Recorder           *httptest.ResponseRecorder
}

func NewTestSetup() *TestSetup {
	stitch := MockStitch{}
	ctx := NewTestingContext()
	manifests := map[string]store.Manifest{}
	ms, _ := NewTestingManifestStore(manifests)
	recorder := httptest.NewRecorder()
	ss, _ := NewTestingSurfaceStore()
	c := NewSurfaceController(ss)
	mc := NewManifestController(ms)

	return &TestSetup{ss, ms, c, mc, manifests, stitch, ctx, recorder}
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

func TestMain(m *testing.M) {
	l.SetLogSink(os.Stdout, events.DebugLevel)
	runTests := m.Run()
	os.Exit(runTests)
}
