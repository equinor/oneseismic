package server

import (
	"context"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/stretchr/testify/assert"
)

type MockStore struct {
	manifests []string
}

func (mbs *MockStore) list(ctx context.Context) ([]string, error) {
	manifests := make([]string, len(mbs.manifests))
	copy(manifests, mbs.manifests)
	return manifests, nil
}

func TestEmptyList(t *testing.T) {
	app := iris.Default()

	mc := manifestController{ms: &MockStore{}}
	app.Get("/", mc.list)

	e := httptest.New(t, app)
	j := e.GET("/").
		Expect().
		Status(httptest.StatusOK).
		JSON().Raw().([]interface{})
	assert.Empty(t, j)
}

func TestList(t *testing.T) {
	m := []string{"a", "b"}

	app := iris.Default()
	mc := manifestController{ms: &MockStore{m}}
	app.Get("/", mc.list)

	e := httptest.New(t, app)
	e.GET("/").
		Expect().
		Status(httptest.StatusOK).
		JSON().Equal(m)
}
