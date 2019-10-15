package controller

import (
	"bytes"
	goctx "context"
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
)

type MockWriter struct {
	io.Writer
}

type MockManifestStore struct{}

func (*MockManifestStore) Fetch(ctx goctx.Context, id string) (store.Manifest, error) {
	return store.Manifest{
		Basename:   "mock",
		Cubexs:     1,
		Cubeys:     1,
		Cubezs:     1,
		Fragmentxs: 1,
		Fragmentys: 1,
		Fragmentzs: 1,
	}, nil
}

func (*MockManifestStore) List(ctx goctx.Context) ([]store.Manifest, error) {
	return nil, nil
}

func NewMockWriter(w io.Writer) MockWriter {
	return MockWriter{w}
}

func (m MockWriter) Header() http.Header {
	return http.Header{}
}

func (m MockWriter) WriteHeader(statusCode int) {
	return
}

type EchoStitch string

func (es EchoStitch) Stitch(ctx goctx.Context, ms store.Manifest, out io.Writer, in io.Reader) (string, error) {

	_, err := io.Copy(out, in)
	return string("ECHO"), err
}

func TestStitch(t *testing.T) {
	l.SetLogSink(os.Stdout, events.DebugLevel)
	echoCtx := irisCtx.NewContext(nil)
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

	echoReq := &http.Request{}
	echoReq.Body = ioutil.NopCloser(strings.NewReader(have))
	buf := bytes.NewBuffer([]byte{})
	echoWriter := NewMockWriter(buf)
	echoCtx.BeginRequest(echoWriter, echoReq)
	echoCtx.Params().Set("manifestID", "12345")
	ms := &MockManifestStore{}
	// ms, _ := store.NewManifestStore(map[string][]byte{"12345": []byte("MANIFEST")})
	echoStitch := StitchController(
		ms,
		EchoStitch("Echo Stitch"))
	type args struct {
		ctx irisCtx.Context
	}

	tests := []struct {
		name string
		args args
		resp MockWriter
	}{
		{name: "Echo little ", args: args{echoCtx}},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			echoStitch(tt.args.ctx)
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
	ms := &MockManifestStore{}
	// ms, _ := store.NewManifestStore(map[string][]byte{"man-1": []byte("MANIFEST")})
	ss, err := store.NewSurfaceStore(map[string][]byte{"surf-2": []byte("SURFACE")})
	if err != nil {
		t.Errorf("StitchSurfaceController() = Error making new Surface: %v", err)
		return
	}
	ssC := StitchSurfaceController(ms, ss, EchoStitch("Echo Stitch"))
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
			ctx.BeginRequest(NewMockWriter(buf), &http.Request{})
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
