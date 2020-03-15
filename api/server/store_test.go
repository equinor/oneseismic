package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceURL(t *testing.T) {
	_, err := NewServiceURL(
		AzureBlobSettings{
			StorageURL:  "http://localhost:10000/%s",
			AccountName: "devstoreaccount1",
			AccountKey:  "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==",
		},
	)
	assert.NoError(t, err)
}
