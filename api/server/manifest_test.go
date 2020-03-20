package server

import (
	"context"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/stretchr/testify/assert"
)

type mockStore struct {
	manifests []string
}

func (mbs *mockStore) list(ctx context.Context) ([]string, error) {
	return mbs.manifests, nil
}

func TestEmptyList(t *testing.T) {
	hs := HTTPServer{
		app: iris.Default(),
		mc:  &manifestController{ms: &mockStore{[]string{}}},
	}

	err := Configure(&hs)

	assert.NoError(t, err)

	e := httptest.New(t, hs.app)
	j := e.GET("/").
		Expect().
		Status(httptest.StatusOK).
		JSON().Raw().([]interface{})
	assert.Empty(t, j)
}

func TestList(t *testing.T) {
	m := []string{}
	hs := HTTPServer{
		app: iris.Default(),
		mc:  &manifestController{ms: &mockStore{manifests: m}},
	}

	err := Configure(&hs)

	assert.NoError(t, err)

	e := httptest.New(t, hs.app)
	e.GET("/").
		Expect().
		Status(httptest.StatusOK).
		JSON().Equal(m)
}
