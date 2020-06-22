package auth

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testAuthServer *httptest.Server
var testAuthServerURL string

func mockGet(url string) (*http.Response, error) {
	jwksURI := `{"jwks_uri":"jwks"}`
	// no need to include e, n in test; they will become 0
	keys := `
	{
		"keys": [
		  {
			"kty": "RSA",
			"kid": "kid"
		  }
		]
	}`
	json := jwksURI
	if url == "jwks" {
		json = keys
	}

	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte(json))),
	}, nil
}

func TestGetRSAKeys(t *testing.T) {
	httpGet = mockGet

	key, err := GetRSAKeys("")
	assert.Nil(t, err)
	assert.Len(t, key, 1)
}
