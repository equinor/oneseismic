package controller

import (
	"bytes"
	goctx "context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/kataras/iris"
	irisCtx "github.com/kataras/iris/context"
	"github.com/stretchr/testify/mock"
)

func NewTestingSurfaceStore() (store.SurfaceStore, error) {
	return store.NewSurfaceStore(map[string][]byte{})
}

func NewTestingManifestStore() (store.ManifestStore, error) {
	return store.NewManifestStore(map[string][]byte{
		"12345": JSONManifest(NewTestManifest()),
		"man-1": JSONManifest(NewTestManifest())})
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
	Stitch             MockStitch
	Ctx                irisCtx.Context
	Recorder           *httptest.ResponseRecorder
}

func NewTestSetup() *TestSetup {
	stitch := MockStitch{}
	ctx := NewTestingContext()
	ms, _ := NewTestingManifestStore()
	recorder := httptest.NewRecorder()
	ss, _ := NewTestingSurfaceStore()
	sc := NewSurfaceController(ss)
	mc := NewManifestController(ms)

	return &TestSetup{ss, ms, sc, mc, stitch, ctx, recorder}
}

func JSONManifest(m store.Manifest) []byte {
	b, err := json.Marshal(&m)
	if err != nil {
		fmt.Println("Could not marshal manifest", err)
	}
	return b
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
	runTests := m.Run()
	os.Exit(runTests)
}
