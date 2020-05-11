package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/stretchr/testify/assert"
)

type mockStore struct {
	blobs []string
}

func (ms *mockStore) list(ctx context.Context) ([]string, error) {
	blobs := make([]string, len(ms.blobs))
	copy(blobs, ms.blobs)
	return blobs, nil
}

func TestEmptyList(t *testing.T) {
	app := iris.Default()

	mc := storeController{&mockStore{}}
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
	mc := storeController{&mockStore{m}}
	app.Get("/", mc.list)

	e := httptest.New(t, app)
	e.GET("/").
		Expect().
		Status(httptest.StatusOK).
		JSON().Equal(m)
}

type mockStoreNotFound struct{}

func (ms *mockStoreNotFound) list(ctx context.Context) ([]string, error) {
	return nil, fmt.Errorf("any error should give a NotFound")
}

func TestListNotFound(t *testing.T) {
	app := iris.Default()
	mc := storeController{&mockStoreNotFound{}}
	app.Get("/", mc.list)

	e := httptest.New(t, app)
	e.GET("/").
		Expect().
		Status(httptest.StatusNotFound)
}
