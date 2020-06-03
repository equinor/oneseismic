package auth

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
)

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

	validator := ValidateIssuer("valid_issuer")
	testApp := iris.Default()
	tests := []struct {
		name   string
		fn     func(ctx context.Context)
		claims jwt.MapClaims
		want   int
	}{
		{
			"Valid token",
			validator,
			jwt.MapClaims{
				"iss": "valid_issuer"},
			200},
		{
			"Invalid issuer",
			validator,
			jwt.MapClaims{
				"iss": "invalid_issuer"},
			401},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.NewContext(testApp)

			ctx.BeginRequest(newMockWriter(bytes.NewBuffer([]byte{})), new(http.Request))
			ctx.Values().Set("jwt", jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), tt.claims))
			tt.fn(ctx)
			ctx.EndRequest()
			got := ctx.GetStatusCode()
			if got != tt.want {
				t.Errorf("Status code got %d != want %d", got, tt.want)
			}
		})
	}
}
