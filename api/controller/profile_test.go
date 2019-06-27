package controller

import (
	"bytes"
	"net/http"
	"reflect"
	"testing"

	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func TestProfileController(t *testing.T) {

	ps := &store.ProfileInMemoryStore{}
	ps.Append("exist", map[string]string{"foo": "bar"})
	c := ProfileController(ps)

	tests := []struct {
		name      string
		sessionID string
		want      int
	}{
		{"Existing profile", "exist", 200},
		{"Not existing profile", "not-exist", 404},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.NewContext(iris.Default())

			ctx.BeginRequest(NewMockWriter(bytes.NewBuffer(make([]byte, 0))), &http.Request{})

			ctx.Params().Set("profileID", tt.sessionID)
			c(ctx)

			if got := ctx.GetStatusCode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProfileController() = %v, want %v", got, tt.want)
			}

			ctx.EndRequest()
		})
	}
}
