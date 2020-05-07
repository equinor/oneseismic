package claims

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

func TestNew(t *testing.T) {
	type args struct {
		audience   string
		issuer     string
		validators []ClaimsValidator
	}
	tests := []struct {
		name string
		args args
		want *Middleware
	}{
		{"No audience",
			args{issuer: "valid_issuer", audience: "valid_audience"},
			&Middleware{issuer: "valid_issuer", audience: "valid_audience"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.args.audience, tt.args.issuer, tt.args.validators...)
			if tt.args.audience != got.audience {
				t.Errorf("Invalid audience %s", got.audience)
			}
			if tt.args.issuer != got.issuer {
				t.Errorf("Invalid issuer %s", got.issuer)
			}
		})
	}
}

type mockWriter struct {
	io.Writer
	statusCode int
}

func newMockWriter(w io.Writer) mockWriter {
	return mockWriter{w, 0}
}

func (m mockWriter) Header() http.Header {
	return http.Header{}
}

func (m mockWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

func TestClaimsMiddleware_Validate(t *testing.T) {

	m := New("valid_audience", "valid_issuer")
	testApp := iris.Default()
	tests := []struct {
		name   string
		m      *Middleware
		claims jwt.MapClaims
		want   int
	}{
		{
			"Valid token",
			m,
			jwt.MapClaims{
				"aud": "valid_audience",
				"iss": "valid_issuer",
				"exp": float64(time.Now().Unix() + 1000)},
			200},
		{
			"Invalid audience",
			m,
			jwt.MapClaims{
				"aud": "invalid_audience",
				"iss": "valid_issuer",
				"exp": float64(time.Now().Unix() + 1000)},
			401},
		{
			"Invalid issuer",
			m,
			jwt.MapClaims{
				"aud": "valid_audience",
				"iss": "invalid_issuer",
				"exp": float64(time.Now().Unix() + 1000)},
			401},
		{
			"Invalid expiry",
			m,
			jwt.MapClaims{
				"aud": "valid_audience",
				"iss": "valid_issuer",
				"exp": float64(time.Now().Unix() - 1000)},
			401},
		{
			"Invalid not before",
			m,
			jwt.MapClaims{
				"aud": "valid_audience",
				"iss": "valid_issuer",
				"exp": float64(time.Now().Unix() + 2000),
				"nbf": float64(time.Now().Unix() + 1000)},
			401},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.NewContext(testApp)

			ctx.BeginRequest(newMockWriter(bytes.NewBuffer([]byte{})), new(http.Request))
			ctx.Values().Set("user-jwt", jwt.NewWithClaims(jwt.GetSigningMethod("RS256"), tt.claims))
			tt.m.Validate(ctx)
			ctx.EndRequest()
			got := ctx.GetStatusCode()
			if got != tt.want {
				t.Errorf("Status code got %d != want %d", got, tt.want)
			}
		})
	}
}
