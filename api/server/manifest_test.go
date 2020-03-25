package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/stretchr/testify/assert"
)

type MockEmptyStore struct{}

func (mbs *MockEmptyStore) list(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func TestEmptyList(t *testing.T) {
	hs := HTTPServer{
		app:                iris.Default(),
		manifestController: &manifestController{&MockEmptyStore{}},
	}

	hs.RegisterEndpoints()
	e := httptest.New(t, hs.app)
	j := e.GET("/").
		Expect().
		Status(httptest.StatusOK).
		JSON().Raw().([]interface{})
	assert.Empty(t, j)
}

type MockStore struct{}

func (mbs *MockStore) list(ctx context.Context) ([]string, error) {
	return []string{""}, nil
}

func TestList(t *testing.T) {
	hs := HTTPServer{
		app:                iris.Default(),
		manifestController: &manifestController{&MockStore{}},
	}

	hs.RegisterEndpoints()

	e := httptest.New(t, hs.app)
	e.GET("/").
		Expect().
		Status(httptest.StatusOK).
		JSON().Array().NotEmpty()
}

type MockStoreMissing struct{}

func (mbs *MockStoreMissing) list(ctx context.Context) ([]string, error) {
	return nil, fmt.Errorf("")
}

func TestMissing(t *testing.T) {
	hs := HTTPServer{
		app:                iris.Default(),
		manifestController: &manifestController{&MockStoreMissing{}},
	}

	hs.RegisterEndpoints()

	e := httptest.New(t, hs.app)
	e.GET("/").
		Expect().
		Status(httptest.StatusNotFound)
}
