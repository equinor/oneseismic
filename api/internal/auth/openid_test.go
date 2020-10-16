package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

type singleResponseClient struct {
	status  int
	content string
}

func (c *singleResponseClient) Get(url string) (*http.Response, error) {
	return &http.Response {
		StatusCode: c.status,
		Body: ioutil.NopCloser(bytes.NewBuffer([]byte(c.content))),
	}, nil
}

func TestFailsOnHttpError(t *testing.T) {
	c := singleResponseClient {
		status: http.StatusNotFound,
	}
	_, err := GetOpenIDConfig(&c, "url")
	if err == nil {
		t.Errorf("GetOpenIDConfig err != nil when all HTTP requests give 404")
	}
}

/*
 * A response body implementation that keeps track of response.Body.Close() and
 * knows if it has been called.
 *
 * It implements the io.ReadCloser interface
 */
type closeRequestBody struct {
	closed int
	content *bytes.Buffer
}

func (b *closeRequestBody) Close() error {
	b.closed++;
	return nil
}

func (b *closeRequestBody) Read(p []byte) (n int, err error) {
	return b.content.Read(p)
}

type closeHttp struct {
	status int
	body closeRequestBody
}

func (h *closeHttp) Get(url string) (*http.Response, error) {
	return &http.Response {
		StatusCode: h.status,
		Body: &h.body,
	}, nil
}

func TestRequestBodyCloseIsCalled(t *testing.T) {
	/*
	 * This test feels super stupid, but any use of the HTTP client means
	 * explicitly closing the request body or leak (???) [1]. Thus this test is
	 * just an extra safety measure so that the close() is not removed by
	 * accident
	 *
	 * [1] https://stackoverflow.com/questions/33238518/what-could-happen-if-i-dont-close-response-body
	 */
	statuscodes := []int {
	    http.StatusOK,
	    http.StatusNotFound,
	}

	for _, status := range statuscodes {
		client := &closeHttp {
			status: status,
			body: closeRequestBody {
				content: bytes.NewBuffer([]byte("{}")),
				closed: 0,
			},
		}

		_ = getJSON(client, "url", struct {}{})
		if client.body.closed != 1 {
			msg := "Close() was not called on request body on status %d"
			t.Errorf(msg, status)
		}
	}
}

/*
 * It's only necessary to test for missing fields, since bad values will be
 * caught by recursive Unmarshal() calls
 */

func TestOpenIDConfigMissingField(t *testing.T) {
	docs := []string {
		"{                        \"issuer\": \"one\", \"token_endpoint\": \"to.ep\" }",
		"{ \"jwks_uri\": \"uri\",                      \"token_endpoint\": \"to.ep\" }",
		"{ \"jwks_uri\": \"uri\", \"issuer\": \"one\"                                }",
	}

	for _, doc := range docs {
		cfg := openIDConfig {}
		err := json.Unmarshal([]byte(doc), &cfg)

		if err == nil {
			t.Errorf("expected missing-field error, got nil; in %s", doc)
		}
	}
}

func TestJWKMissingField(t *testing.T) {
	docs := []string {
		"{                   \"kid\": \"id\", \"n\": \"N\", \"e\": \"E\" }",
		"{ \"kty\": \"key\",                  \"n\": \"N\", \"e\": \"E\" }",
		"{ \"kty\": \"key\", \"kid\": \"id\",               \"e\": \"E\" }",
		"{ \"kty\": \"key\", \"kid\": \"id\", \"n\": \"N\"               }",
	}

	for _, doc := range docs {
		j := jwk {}
		err := json.Unmarshal([]byte(doc), &j)

		if err == nil {
			t.Errorf("expected missing-field error, got nil; in %s", doc)
		}
	}
}

func TestJWKSMissingField(t *testing.T) {
	doc := "{}"
	keyset := jwks {}
	err := json.Unmarshal([]byte(doc), &keyset)
	if err == nil {
		t.Errorf("expected missing-field error, got nil")
	}
}

func TestJWKEmptyKeys(t *testing.T) {
	doc := "{ \"keys\": [] }"
	keyset := jwks {}
	err := json.Unmarshal([]byte(doc), &keyset)
	if err == nil {
		t.Errorf("expected missing-field error, got nil; in %s", doc)
	}
}

type openIDHttpClient struct {}
func (c *openIDHttpClient) Get(url string) (*http.Response, error) {
	if url == "auth.server" {
		response := "{ " +
			"\"jwks_uri\": \"jwk.uri\", " +
			"\"issuer\": \"test\", " +
			"\"token_endpoint\": \"tp.ep\" " +
		"}"
		return &http.Response {
			StatusCode: http.StatusOK,
			Body: ioutil.NopCloser(bytes.NewBuffer([]byte(response))),
		}, nil
	}

	if url == "jwk.uri" {
		response := "{ \"keys\": [" +
			"{ \"kty\": \"EC\", \"kid\": \"id\", \"e\": \"10\", \"n\": \"2\" }" +
		"] }"
		return &http.Response {
			StatusCode: http.StatusOK,
			Body: ioutil.NopCloser(bytes.NewBuffer([]byte(response))),
		}, nil
	}

	return nil, fmt.Errorf("Unknown url %s", url)
}

func TestOpenIDConfigWithoutKeys(t *testing.T) {
	c := openIDHttpClient {}
	_, err := GetOpenIDConfig(&c, "auth.server")
	if _, ok := err.(*noRSAKeys); !ok {
		t.Errorf("Expected err to be noRSAKeys; was %#v", err)
	}
}
