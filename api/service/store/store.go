package store

import (
	"fmt"
	"net/url"
	"os"
	"sync"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/equinor/seismic-cloud/api/config"
	"github.com/equinor/seismic-cloud/api/events"
	l "github.com/equinor/seismic-cloud/api/logger"
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
		containerURL *azblob.ContainerURL
		bufferSize   int
		maxBuffers   int
	}
)
type BasePath string
type ConnStr string
type AzureBlobSettings struct {
	AccountName   string
	AccountKey    string
	ContainerName string
}

// func New(kind interface{}, backingStore interface{}) (*Store, error){
// 	if kint.type == Manifest
// 		return manifeststore(backingStore)

// }

func NewAzBlobStore(accountName, accountKey, containerName string) (*AzBlobStore, error) {

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

	return &AzBlobStore{
		containerURL: &containerURL,
		bufferSize:   2 * 1024 * 1024,
		maxBuffers:   100}, nil
}

func NewMongoDbStore(connStr ConnStr) (*MongoDbStore, error) {
	return &MongoDbStore{connStr}, nil
}

func NewLocalFileStore(basePath BasePath) (*LocalFileStore, error) {
	op := events.Op("store.NewLocalFileStore")
	basePathStr := string(basePath)
	if len(basePath) == 0 {
		return nil, events.E(op, "basePath cannot be empty")
	}

	if _, err := os.Stat(basePathStr); os.IsNotExist(err) {
		os.MkdirAll(basePathStr, 0700)
	} else if err != nil {
		return nil, events.E(op, "accessing basePath failed", err)
	}

	return &LocalFileStore{basePath}, nil
}

func NewInMemoryStore(m map[string][]byte) (*InMemoryStore, error) {
	return &InMemoryStore{m: m}, nil
}
