package store

import (
	"fmt"
	"net/url"
	"os"
	"sync"

	azb "github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/equinor/seismic-cloud/api/events"
)

type (
	LocalFileStore struct {
		basePath BasePath
	}

	MongoDbStore struct {
		connString ConnStr
	}

	InMemoryStore struct {
		m    map[string][]byte
		lock sync.RWMutex
	}

	AzBlobStore struct {
		containerURL *azb.ContainerURL
		bufferSize   int
		maxBuffers   int
	}
	AzSecBlobStore struct {
		ServiceURL *azb.ServiceURL
	}
)
type BasePath string
type ConnStr string
type AzureBlobSettings struct {
	StorageURL    string
	AccountName   string
	AccountKey    string
	ContainerName string
}

func NewAzBlobStore(az AzureBlobSettings) (*AzBlobStore, error) {

	credential, err := azb.NewSharedKeyCredential(az.AccountName, az.AccountKey)
	if err != nil {
		return nil, err
	}

	url, err := url.Parse(
		fmt.Sprintf(az.StorageURL,
			az.AccountName,
			az.ContainerName))
	if err != nil {
		return nil, err
	}
	containerURL := azb.NewContainerURL(
		*url,
		azb.NewPipeline(credential, azb.PipelineOptions{}),
	)

	return &AzBlobStore{
		containerURL: &containerURL,
		bufferSize:   2 * 1024 * 1024,
		maxBuffers:   100}, nil
}

func NewMongoDbStore(connStr ConnStr) (*MongoDbStore, error) {
	return &MongoDbStore{connStr}, nil
}

func NewLocalFileStore(basePath BasePath) (*LocalFileStore, error) {
	basePathStr := string(basePath)
	if len(basePath) == 0 {
		return nil, events.E("basePath cannot be empty", events.ErrorLevel)
	}

	if _, err := os.Stat(basePathStr); os.IsNotExist(err) {
		err = os.MkdirAll(basePathStr, 0700)
		if err != nil {
			return nil, events.E("Make basePath", err)
		}
	} else if err != nil {
		return nil, events.E("accessing basePath", err)
	}

	return &LocalFileStore{basePath}, nil
}

func NewInMemoryStore(m map[string][]byte) (*InMemoryStore, error) {
	return &InMemoryStore{m: m}, nil
}
