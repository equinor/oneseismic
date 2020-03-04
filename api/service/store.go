package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/Azure/azure-storage-blob-go/azblob"

	"github.com/equinor/seismic-cloud/api/events"
)

type ManifestStore interface {
	Download(context.Context, string) (*Manifest, error)
}

type Manifest struct {
	Guid string
}

type ContainerURL struct {
	azblob.ContainerURL
}

type AzureBlobSettings struct {
	StorageURL    string
	AccountName   string
	AccountKey    string
	ContainerName string
}

func (mbs *ContainerURL) Download(ctx context.Context, manifestID string) (*Manifest, error) {
	mani := &Manifest{}

	blobURL := mbs.NewBlockBlobURL(manifestID)

	resp, err := blobURL.Download(
		ctx,
		0,
		azblob.CountToEnd,
		azblob.BlobAccessConditions{},
		false,
	)
	if err != nil {
		return mani, events.E("Download from blobstore", err, events.Marshalling, events.ErrorLevel)
	}
	b, err := ioutil.ReadAll(resp.Body(azblob.RetryReaderOptions{}))
	if err != nil {
		return mani, events.E("Could not read manifest from blob store", err)
	}
	err = json.Unmarshal(b, mani)
	if err != nil {
		return mani, events.E("Unmarshaling to Manifest", err, events.Marshalling, events.ErrorLevel)
	}
	return mani, nil
}

func NewContainerURL(az AzureBlobSettings) (*ContainerURL, error) {

	credential, err := azblob.NewSharedKeyCredential(az.AccountName, az.AccountKey)
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
	containerURL := azblob.NewContainerURL(
		*url,
		azblob.NewPipeline(credential, azblob.PipelineOptions{}),
	)

	return &ContainerURL{containerURL}, err
}
