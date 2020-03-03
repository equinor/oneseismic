package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"

	azb "github.com/Azure/azure-storage-blob-go/azblob"

	"github.com/equinor/seismic-cloud/api/events"
	seismic_core "github.com/equinor/seismic-cloud/api/proto"
)

type ManifestStore interface {
	Download(context.Context, string) (*Manifest, error)
}

type Manifest seismic_core.Geometry

type manifestBlobStore struct {
	blobStore *AzBlobStore
}

type AzBlobStore struct {
	containerURL *azb.ContainerURL
	bufferSize   int
	maxBuffers   int
}

type AzureBlobSettings struct {
	StorageURL    string
	AccountName   string
	AccountKey    string
	ContainerName string
}

func NewManifestStore(persistance interface{}) (ManifestStore, error) {

	switch persistance := persistance.(type) {
	case AzureBlobSettings:

		s, err := NewAzBlobStore(persistance)
		if err != nil {
			return nil, events.E("new azure blob store", err)
		}
		return &manifestBlobStore{blobStore: s}, nil
	default:
		return nil, events.E("No manifest store persistance selected", events.ErrorLevel)
	}
}

func (mbs *manifestBlobStore) Download(ctx context.Context, manifestID string) (*Manifest, error) {
	mani := &Manifest{}

	blobURL := mbs.blobStore.containerURL.NewBlockBlobURL(manifestID)

	resp, err := blobURL.Download(
		ctx,
		0,
		azb.CountToEnd,
		azb.BlobAccessConditions{},
		false,
	)
	if err != nil {
		return mani, events.E("Download from blobstore", err, events.Marshalling, events.ErrorLevel)
	}
	b, err := ioutil.ReadAll(resp.Body(azb.RetryReaderOptions{}))
	if err != nil {
		return mani, events.E("Could not read manifest from blob store", err)
	}
	err = json.Unmarshal(b, mani)
	if err != nil {
		return mani, events.E("Unmarshaling to Manifest", err, events.Marshalling, events.ErrorLevel)
	}
	return mani, nil
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
