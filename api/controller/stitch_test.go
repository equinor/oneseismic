package controller

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

type MockResponseWriter struct {
	io.Writer
}

func NewMock(w io.Writer) MockResponseWriter {
	return MockResponseWriter{w}
}

func (m MockResponseWriter) Header() http.Header {
	return http.Header{}
}

func (m MockResponseWriter) WriteHeader(statusCode int) {
	return
}

func TestStitch(t *testing.T) {

	echoCtx := context.NewContext(nil)
	want :=
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
	echoReq.Body = ioutil.NopCloser(strings.NewReader(want))
	buf := bytes.NewBuffer([]byte{})
	echoWriter := NewMock(buf)
	echoCtx.BeginRequest(echoWriter, echoReq)

	type args struct {
		ctx iris.Context
	}

	tests := []struct {
		name string
		args args
		resp MockResponseWriter
	}{
		{name: "Echo little ", args: args{echoCtx}},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			Stitch(tt.args.ctx)
			got, err := ioutil.ReadAll(buf)
			if err != nil {
				t.Errorf("Stitch() Readall err %v", err)
				return
			}
			if string(got) != want {
				t.Errorf("Stitch() got = %v, want %v", got, want)
			}
			tt.args.ctx.EndRequest()

		})
	}
}
