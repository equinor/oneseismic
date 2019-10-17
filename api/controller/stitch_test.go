package controller

import (
	"bytes"
	goctx "context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
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

type MockManifestStore struct {
	mock.Mock
}

func (m *MockManifestStore) Fetch(id string) (store.Manifest, error) {
	args := m.Called(id)
	return args.Get(0).(store.Manifest), args.Error(1)
}

type MockWriter struct {
	header http.Header
	buffer *bytes.Buffer
}

func NewMockWriter() MockWriter {
	return MockWriter{http.Header{}, bytes.NewBuffer(make([]byte, 0))}
}

func (m MockWriter) Write(b []byte) (int, error) {
	return m.buffer.Write(b)
}

func (m MockWriter) Header() http.Header {
	return m.header
}

func (m MockWriter) WriteHeader(statusCode int) {}

type MockStitch struct {
	mock.Mock
}

func (m MockStitch) Stitch(ctx goctx.Context, ms store.Manifest, out io.Writer, in io.Reader) (string, error) {
	_, err := io.Copy(out, in)
	args := m.Called(ctx, ms, out, in)
	return args.String(0), err
}

func NewMockContext() irisCtx.Context {
	return irisCtx.NewContext(iris.Default())
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
	req, _ := http.NewRequest("GET", "test.com", ioutil.NopCloser(strings.NewReader(have)))
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
	ctx := NewMockContext()

	manifest := NewTestManifest()

	stitch := MockStitch{}
	stitch.On("Stitch", mock.Anything, manifest, mock.Anything, mock.Anything).Return("", nil)

	ms := &MockManifestStore{}
	ms.On("Fetch", "12345").Return(manifest, nil)

	stitchController := StitchController(ms, stitch)

	ctx.BeginRequest(NewMockWriter(), NewGarbageRequest())
	ctx.Params().Set("manifestID", "12345")

	stitchController(ctx)

	if code := ctx.GetStatusCode(); code != 200 {
		t.Errorf("Expected Status Code 200 got %v", code)
	}

	ctx.EndRequest()
}

func TestStitchMissingManifestNotFoundCode(t *testing.T) {
	l.SetLogSink(os.Stdout, events.DebugLevel)
	ctx := NewMockContext()

	stitch := MockStitch{}

	ms := &MockManifestStore{}
	ms.On("Fetch", "12345").Return(store.Manifest{}, errors.New(""))

	ctx.BeginRequest(NewMockWriter(), NewGarbageRequest())
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

	ms := &MockManifestStore{}
	ms.On("Fetch", "man-1").Return(manifest, nil)

	stitch := MockStitch{}
	stitch.On("Stitch", mock.Anything, manifest, mock.Anything, mock.Anything).Return("", nil)

	ss, err := NewTestingSurfaceStore()

	if err != nil {
		t.Errorf("Error making TestingSurfaceStore: %v", err)
		return
	}

	ssC := StitchSurfaceController(ms, ss, stitch)

	ctx := NewMockContext()
	ss.Upload(goctx.Background(), "surf-1", "test-user", strings.NewReader("SURFACE"))

	ctx.BeginRequest(NewMockWriter(), NewGarbageRequest())
	ctx.Params().Set("manifestID", "man-1")
	ctx.Params().Set("surfaceID", "surf-1")

	ssC(ctx)

	ctx.EndRequest()
}
