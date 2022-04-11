package storage

import (
	"context"
	"net/url"
	"io/ioutil"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"github.com/equinor/oneseismic/api/internal"
)


type entity struct {
	Data []byte
	Tag  *string
}

/*
 *  Minimal interface for fetching blobs/files from storage. This hides a lot of
 *  feature and details about the underlying storage from the rest of the
 *  system, making it easy to swap out the storage provider. This means testing
 *  becomes easier through custom storage implementations.
 */
type StorageClient interface {
	/*
	 * Get a blob or file from storage
	 */
	Get(ctx context.Context, blob string) (*entity, error)
}

/*
 * Azure Blob Store implementation of a oneseismic StorageClient
 */
type AzStorage struct {
}

func (c *AzStorage) Get(ctx context.Context, blob string) (*entity, error) {
	_, err := url.Parse(blob)
	if err != nil {
		log.Printf("Invalid URL: %s", blob)
		return nil, internal.InternalError(err.Error())
	}

	client, err := azblob.NewBlobClientWithNoCredential(blob, nil)
	if err != nil {
		return nil, internal.InternalError(err.Error())
	}

	dl, err := client.Download(ctx, nil)
	if err != nil {
		return nil, err
	}

	body := dl.Body(&azblob.RetryReaderOptions{})
	defer body.Close()
	data, err := ioutil.ReadAll(body)
	return &entity{ Data: data, Tag: dl.ETag }, err
}

func NewAzStorage() *AzStorage {
	return &AzStorage{}
}
