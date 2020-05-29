package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"

	"github.com/kataras/iris/v12/httptest"
)

type mockStore struct {
	guids     []string
	dims      []int32
	mani      Manifest
	linesMock []int32
}

func (ms *mockStore) list(ctx context.Context, root, token string) ([]string, error) {
	return ms.guids, nil
}

func (ms *mockStore) manifest(ctx context.Context, root, guid, token string) (*Manifest, error) {
	return &ms.mani, nil
}

func (ms *mockStore) dimensions(ctx context.Context, root, guid, token string) ([]int32, error) {
	return ms.dims, nil
}

func (ms *mockStore) lines(ctx context.Context, root, guid string, dimension int32, token string) ([]int32, error) {
	return ms.linesMock, nil
}

func genKeySetAndJwt() (map[string]interface{}, string) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	kid := "a"

	sigKeySet := make(map[string]interface{})
	sigKeySet[kid] = privateKey.Public()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{})
	token.Header["kid"] = kid
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		panic(err)
	}

	return sigKeySet, tokenString

}

func TestList(t *testing.T) {
	sigKeySet, jwt := genKeySetAndJwt()
	c := Config{SigKeySet: sigKeySet}
	app := newApp(&c)

	guids := []string{"a", "b"}
	sc := storeController{&mockStore{guids: guids}}
	app.Get("/{root:string}", sc.list)

	e := httptest.New(t, app)

	e.GET("/a").
		WithHeader("Authorization", "Bearer "+jwt).
		Expect().
		Status(httptest.StatusOK).
		JSON().Equal(guids)
}

func TestContainerServices(t *testing.T) {
	sigKeySet, jwt := genKeySetAndJwt()
	c := Config{SigKeySet: sigKeySet}
	app := newApp(&c)

	sc := storeController{&mockStore{}}
	app.Get("/{root:string}/{guid:string}", sc.services)

	e := httptest.New(t, app)
	e.GET("/a/a").
		WithHeader("Authorization", "Bearer "+jwt).
		Expect().
		Status(httptest.StatusOK).
		JSON().Equal([]string{"slice"})
}

func TestDimensions(t *testing.T) {
	sigKeySet, jwt := genKeySetAndJwt()
	c := Config{SigKeySet: sigKeySet}
	app := newApp(&c)

	dims := []int32{2}
	sc := storeController{&mockStore{dims: dims}}
	app.Get("/{root:string}/{guid:string}/slice", sc.dimensions)

	e := httptest.New(t, app)
	e.GET("/a/a/slice").
		WithHeader("Authorization", "Bearer "+jwt).
		Expect().
		Status(httptest.StatusOK).
		JSON().Equal(dims)
}

func TestLines(t *testing.T) {
	sigKeySet, jwt := genKeySetAndJwt()
	c := Config{SigKeySet: sigKeySet}
	app := newApp(&c)

	lines := []int32{0}
	sc := storeController{&mockStore{linesMock: lines}}
	app.Get("/{root:string}/{guid:string}/slice/{dimension:int32}", sc.lines)

	e := httptest.New(t, app)
	e.GET("/a/a/slice/0").
		WithHeader("Authorization", "Bearer "+jwt).
		Expect().
		Status(httptest.StatusOK).
		JSON().Equal(lines)
}
