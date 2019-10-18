package store

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/equinor/seismic-cloud/api/config"
	l "github.com/equinor/seismic-cloud/api/logger"
)

type (
	FileStore struct {
		basePath string
	}

	MongoDbStore struct {
		connString string
	}

	InMemoryStore struct {
		m    map[string]Manifest
		lock sync.RWMutex
	}

	AzBlobStorage struct {
		AccountName   string
		AccountKey    string
		ContainerName string
	}
)

type BlobStore struct {
	containerURL *azblob.ContainerURL
	bufferSize   int
	maxBuffers   int
}

// ConnStr is the connectionstring a db would use
type ConnStr string

func NewAzBlobStorage(accountName, accountKey, containerName string) (*BlobStore, error) {

	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		l.LogE("blobStore AzBlobStorage", "Invalid credentials", err)
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	u, err := url.Parse(
		fmt.Sprintf(config.AzStorageURL(),
			accountName,
			containerName))
	if err != nil {
		return nil, err
	}
	containerURL := azblob.NewContainerURL(*u, p)

	return &BlobStore{containerURL: &containerURL,
		bufferSize: 2 * 1024 * 1024,
		maxBuffers: 100}, nil
}
