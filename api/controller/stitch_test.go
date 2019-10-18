package controller

import (
	goctx "context"
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
	return args.String(0), err
}

func NewGarbageRequest() *http.Request {
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
	req := httptest.NewRequest("GET", "/stitch", ioutil.NopCloser(strings.NewReader(have)))
	return req
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

func TestStitchControllerSuccess(t *testing.T) {
	l.SetLogSink(os.Stdout, events.DebugLevel)
	ctx := NewTestingContext()

	manifest := NewTestManifest()

	stitch := MockStitch{}
	stitch.On("Stitch", mock.Anything, manifest, mock.Anything, mock.Anything).Return("", nil)

	ms, _ := NewTestingManifestStore(map[string]store.Manifest{"12345": manifest})

	stitchController := StitchController(ms, stitch)

	ctx.BeginRequest(httptest.NewRecorder(), NewGarbageRequest())
	ctx.Params().Set("manifestID", "12345")

	stitchController(ctx)

	if code := ctx.GetStatusCode(); code != 200 {
		t.Errorf("Expected Status Code 200 got %v", code)
	}

	ctx.EndRequest()
}

func TestStitchMissingManifestNotFoundCode(t *testing.T) {
	l.SetLogSink(os.Stdout, events.DebugLevel)
	ctx := NewTestingContext()

	stitch := MockStitch{}

	ms, _ := NewTestingManifestStore(map[string]store.Manifest{})

	ctx.BeginRequest(httptest.NewRecorder(), NewGarbageRequest())
	ctx.Params().Set("manifestID", "12345")
	stitchController := StitchController(ms, stitch)
	stitchController(ctx)

	if code := ctx.GetStatusCode(); code != 404 {
		t.Errorf("Expected Status Code 404 got %v", code)
	}

	ctx.EndRequest()
}

func TestStitchSurfaceController(t *testing.T) {
	l.SetLogSink(os.Stdout, events.DebugLevel)
	manifest := NewTestManifest()

	ms, err := NewTestingManifestStore(map[string]store.Manifest{"man-1": manifest})

	stitch := MockStitch{}
	stitch.On("Stitch", mock.Anything, manifest, mock.Anything, mock.Anything).Return("", nil)

	ss, err := NewTestingSurfaceStore()

	if err != nil {
		t.Errorf("Error making TestingSurfaceStore: %v", err)
		return
	}

	ssC := StitchSurfaceController(ms, ss, stitch)

	ctx := NewTestingContext()
	ss.Upload(goctx.Background(), "surf-1", "test-user", strings.NewReader("SURFACE"))

	ctx.BeginRequest(httptest.NewRecorder(), NewGarbageRequest())
	ctx.Params().Set("manifestID", "man-1")
	ctx.Params().Set("surfaceID", "surf-1")

	ssC(ctx)

	ctx.EndRequest()
}
