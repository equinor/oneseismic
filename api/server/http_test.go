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

func (mbs *MockStore) List(ctx context.Context) ([]string, error) {
	manifests := make([]string, len(mbs.manifests))
	copy(manifests, mbs.manifests)
	return manifests, nil
}

func TestEmptyList(t *testing.T) {
	hs := HTTPServer{
		app:           iris.Default(),
		manifestStore: &MockStore{},
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
	m := []string{"a", "b"}
	hs := HTTPServer{
		app:           iris.Default(),
		manifestStore: &MockStore{m},
	}

	err := Configure(&hs)

	assert.NoError(t, err)

	e := httptest.New(t, hs.app)
	e.GET("/").
		Expect().
		Status(httptest.StatusOK).
		JSON().Equal(m)
}
