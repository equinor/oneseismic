package controller

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

type MockWriter struct {
	io.Writer
}

type MockManifestStore struct{}

func (*MockManifestStore) Fetch(id string) ([]byte, error) {
	return []byte("MANIFEST"), nil
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

func (es EchoStitch) Stitch(out io.Writer, in io.Reader) (string, error) {

	_, err := io.Copy(out, in)
	return string(es), err
}

func TestStitch(t *testing.T) {

	echoCtx := context.NewContext(nil)
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

	want := "M:" + string([]byte{8, 0, 0, 0}) + "MANIFEST" + have

	echoReq := &http.Request{}
	echoReq.Body = ioutil.NopCloser(strings.NewReader(have))
	buf := bytes.NewBuffer([]byte{})
	echoWriter := NewMockWriter(buf)
	echoCtx.BeginRequest(echoWriter, echoReq)
	echoStitch := StitchController(
		new(MockManifestStore),
		EchoStitch("Echo Stitch"),
		log.New(os.Stdout, "MockLog", log.Ldate))
	type args struct {
		ctx iris.Context
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
			if string(got) != want {
				t.Errorf("Stitch() got = %v, want %v", string(got), want)
			}
			tt.args.ctx.EndRequest()

		})
	}
}
