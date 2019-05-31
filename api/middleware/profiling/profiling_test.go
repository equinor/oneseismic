package profiling

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/kataras/iris/context"
)

type mockWriter struct {
	io.Writer
	statusCode int
	header     http.Header
}

func newMockWriter(w io.Writer) mockWriter {
	m := mockWriter{w, 0, http.Header{}}
	return m

}

func (m mockWriter) Header() http.Header {
	return m.header
}

func (m mockWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
	return
}

// Test for Profiling middleware that checks
// that the Duration Trailer is present and not empty
func TestProfilingMiddlewareDuration(t *testing.T) {

	ctx := context.NewContext(nil)
	ctx.BeginRequest(newMockWriter(bytes.NewBuffer([]byte{})), new(http.Request))
	Duration(ctx)
	ctx.EndRequest()

	want := "Duration"
	got := ctx.ResponseWriter().Header().Get("Trailer")

	if got != want {
		t.Errorf("Header Trailer: %v != want %v", got, want)
	}

	got = ctx.ResponseWriter().Header().Get("Duration")
	if got == "" {
		t.Errorf("Header Duration is empty)
	}

}
