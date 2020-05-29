package server

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURL(t *testing.T) {
	u, err := parseStorageURL("account", "%s.url")
	assert.Nil(t, err)
	assert.Equal(t, u.Path, "account.url")
}

func TestU(t *testing.T) {
	u, _ := url.Parse("http://some.url")

	createServiceURL(*u, "some.token.cert")
}
