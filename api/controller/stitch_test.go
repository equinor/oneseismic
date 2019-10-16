package controller

import (
	"bytes"
	goctx "context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/equinor/seismic-cloud/api/events"
	l "github.com/equinor/seismic-cloud/api/logger"
	"github.com/equinor/seismic-cloud/api/service/store"
	irisCtx "github.com/kataras/iris/context"
	"github.com/stretchr/testify/mock"
)

type MockWriter struct {
	io.Writer
	mock.Mock
}

type MockManifestStore struct {
	mock.Mock
}

func (m *MockManifestStore) Fetch(id string) (store.Manifest, error) {
	args := m.Called(id)
	return args.Get(0).(store.Manifest), args.Error(1)
}

func NewMockWriter(w io.Writer) MockWriter {
	return MockWriter{w, mock.Mock{}}
}

func (m MockWriter) Header() http.Header {
	args := m.Called()
	return args.Get(0).(http.Header)
}

func (m MockWriter) WriteHeader(statusCode int) {
	m.Called(statusCode)
}

type MockStitch struct {
	mock.Mock
}

func (m MockStitch) Stitch(ctx goctx.Context, ms store.Manifest, out io.Writer, in io.Reader) (string, error) {
	_, err := io.Copy(out, in)
	args := m.Called(ctx, ms, out, in)
	return args.String(0), err
}

type MockContext struct {
	mock.Mock
	irisCtx.Context
}

func (ctx *MockContext) StatusCode(statusCode int) {
	ctx.Called(statusCode)
	ctx.ResponseWriter().WriteHeader(statusCode)
}

func TestStitch(t *testing.T) {
	l.SetLogSink(os.Stdout, events.DebugLevel)
	ctx := irisCtx.NewContext(nil)
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

	req := &http.Request{}
	req.Body = ioutil.NopCloser(strings.NewReader(have))
	buf := bytes.NewBuffer([]byte{})
	writer := NewMockWriter(buf)
	writer.On("Header").Return(http.Header{})
	writer.On("WriteHeader", 200).Return().Once()
	ctx.BeginRequest(writer, req)
	ctx.Params().Set("manifestID", "12345")
	manifest := store.Manifest{
		Basename:   "mock",
		Cubexs:     1,
		Cubeys:     1,
		Cubezs:     1,
		Fragmentxs: 1,
		Fragmentys: 1,
		Fragmentzs: 1,
	}

	stitch := MockStitch{}
	stitch.On("Stitch", mock.Anything, manifest, mock.Anything, mock.Anything).Return("", nil)

	ms := &MockManifestStore{}
	ms.On("Fetch", "12345").Return(manifest, nil)
	stitchController := StitchController(
		ms,
		stitch)
	type args struct {
		ctx irisCtx.Context
	}

	tests := []struct {
		name string
		args args
	}{
		{name: "Echo little ", args: args{ctx}},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			stitchController(tt.args.ctx)
			got, err := ioutil.ReadAll(buf)
			if err != nil {
				t.Errorf("Stitch() Readall err %v", err)
				return
			}
			if string(got) != have {
				t.Errorf("Stitch() got = %v, want %v", string(got), have)
			}
			tt.args.ctx.EndRequest()

		})
	}
}

func TestStitchSurfaceController(t *testing.T) {
	l.SetLogSink(os.Stdout, events.DebugLevel)
	type args struct {
		manID  string
		surfID string
	}
	manifest := store.Manifest{
		Basename:   "mock",
		Cubexs:     1,
		Cubeys:     1,
		Cubezs:     1,
		Fragmentxs: 1,
		Fragmentys: 1,
		Fragmentzs: 1,
	}
	ms := &MockManifestStore{}
	ms.On("Fetch", "man-1").Return(manifest, nil)
	stitch := MockStitch{}
	stitch.On("Stitch", mock.Anything, manifest, mock.Anything, mock.Anything).Return("", nil)
	ss, err := store.NewSurfaceStore(map[string][]byte{"surf-2": []byte("SURFACE")})
	if err != nil {
		t.Errorf("StitchSurfaceController() = Error making new Surface: %v", err)
		return
	}
	ssC := StitchSurfaceController(ms, ss, stitch)
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			"Reply from exisiting manifest and surface",
			args{manID: "man-1", surfID: "surf-2"},
			[]byte("SURFACE"),
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			ctx := irisCtx.NewContext(nil)
			buf := &bytes.Buffer{}
			writer := NewMockWriter(buf)
			writer.On("Header").Return(http.Header{})
			writer.On("WriteHeader", 200).Return().Once()
			ctx.BeginRequest(writer, &http.Request{})
			ctx.Params().Set("manifestID", tt.args.manID)
			ctx.Params().Set("surfaceID", tt.args.surfID)

			ssC(ctx)

			got, err := ioutil.ReadAll(buf)
			if err != nil {
				t.Errorf("StitchSurfaceController() = Error %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StitchSurfaceController() = %v, want %v", got, tt.want)
			}
			ctx.EndRequest()
		})

	}
}

func TestStitchMissingManifest(t *testing.T) {
	req := &http.Request{}
	l.SetLogSink(os.Stdout, events.DebugLevel)
	buf := &bytes.Buffer{}
	writer := NewMockWriter(buf)
	writer.On("Header").Return(http.Header{})
	writer.On("WriteHeader", 404).Return().Once()
	ctx := &MockContext{mock.Mock{}, irisCtx.NewContext(nil)}
	ctx.BeginRequest(writer, req)
	ctx.Params().Set("manifestID", "12345")
	ctx.On("StatusCode", 404).Return().Once()
	stitch := MockStitch{}
	ms := &MockManifestStore{}
	ms.On("Fetch", "12345").Return(store.Manifest{}, errors.New(""))

	stitchController := StitchController(
		ms,
		stitch)
	stitchController(ctx)
	ctx.AssertExpectations(t)
}
