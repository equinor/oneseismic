package store

import (
	"fmt"
	"net/url"

	azb "github.com/Azure/azure-storage-blob-go/azblob"
)

type (
	MongoDbStore struct {
		connString ConnStr
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
