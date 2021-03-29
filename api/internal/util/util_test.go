package util

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

/*
 * Implement the azblob.StorageError interface to emulate failures without
 * having to connect anywhere, have correct URLs or accounts set up.
 */
type fakeAzStorageError struct {
	serviceCode azblob.ServiceCodeType
	azblob.ResponseError
}

func (se *fakeAzStorageError) ServiceCode() azblob.ServiceCodeType {
	return se.serviceCode
}

/*
 * Emulate a trivial error response from azure blobs.
 *
 * The payload looks nothing like what azure would return, so if azblob tries
 * to actually parse the data to make something meaningful of it, it will have
 * a bad time. Plain http status codes should go a long way though to ensure
 * that the correct code paths are taken & tested.
 *
 * The arguments are deliberately minimal. This gives no flexibility, but more
 * arguments can be added later should there be a need. The goal is to make the
 * tests building on it simple and clear, and hand-building requests is tedious
 * and noisy.
 */
func fakeStorageError(status int) azblob.ResponseError {
	text := http.StatusText(status)
	response := &http.Response {
		Status:     text,
		StatusCode: status,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Body: ioutil.NopCloser(strings.NewReader(text)),
		ContentLength: int64(len(text)),
	}
	return azblob.NewResponseError(
		fmt.Errorf("fake-failure"),
		response,
		"fake-description",
	).(azblob.ResponseError)
}

/*
 * This is pretty much just a named constructor, and a type conversion to the
 * azblob.StorageError interface.
 */
func makeStorageError(
	serviceCode azblob.ServiceCodeType,
	responseErr azblob.ResponseError,
) (azblob.StorageError) {
	return &fakeAzStorageError {
		serviceCode: azblob.ServiceCodeContainerNotFound,
		ResponseError: responseErr,
	}
}

/*
 * Reading all of the response body should be a fairly common thing to do, but
 * the code and error-check to do so is *super* tedious and uninteresting. This
 * helper drains and closes the stream, and aborts the test should it fail.
 */
func readBody(response *http.Response, t *testing.T) string {
	body := response.Body
	defer body.Close()
	text, err := ioutil.ReadAll(body)
	assert.Nil(t, err)
	return string(text)
}

/*
 * Right now this only verifies that 404 from azblob's backend (i.e. the live
 * azure) results in a 404 and aborted request. Whenever the
 * AbortOnManifestError() interrogates the StorageError to write a more
 * structured response, this test, and possible siblings, should be extended to
 * cover the error conditions and code paths.
 */
func TestAbortOnManifest404(t *testing.T) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	status := http.StatusNotFound
	storageError := makeStorageError(
		azblob.ServiceCodeContainerNotFound,
		fakeStorageError(status),
	)
	AbortOnManifestError(ctx, storageError)

	assert.Equal(t, status, w.Result().StatusCode, "Wrong status code")
	want := "Not Found"
	text := readBody(w.Result(), t)
	assert.Equal(t, want, text, "Wrong response body")
	assert.True(t, ctx.IsAborted(), "gin.Context was not aborted as it should")
}

type TokensIdentity struct {}

func (*TokensIdentity) GetOnbehalf(s string) (string, error) {
	return s, nil
}

func (*TokensIdentity) Invalidate(s string) {
}
