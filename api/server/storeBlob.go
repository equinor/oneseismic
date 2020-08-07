package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/Azure/azure-storage-blob-go/azblob"
)

type store interface {
	list(ctx context.Context, token string) ([]string, error)
	manifest(ctx context.Context, guid, token string) (*Manifest, error)
	dimensions(ctx context.Context, guid, token string) ([]int32, error)
	lines(ctx context.Context, guid string, dimension int32, token string) ([]int32, error)
}

type storageURL struct {
	url.URL
}

type Manifest struct {
	Dimensions [][]int32 `json:"dimensions"`
	Samples    int32     `json:"samples"`
}

func (storage *storageURL) manifest(ctx context.Context, guid, token string) (*Manifest, error) {
	blobURL := createServiceURL(storage.URL, token).NewContainerURL(guid).NewBlockBlobURL("manifest.json")
	downloadResponse, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return nil, err
	}

	bodyStream := downloadResponse.Body(azblob.RetryReaderOptions{MaxRetryRequests: 20})
	downloadedData := bytes.Buffer{}
	_, err = downloadedData.ReadFrom(bodyStream)
	if err != nil {
		return nil, err
	}
	b := downloadedData.Bytes()
	manifest := Manifest{}
	_ = json.Unmarshal(b, &manifest)
	return &manifest, nil
}

func (storage *storageURL) dimensions(ctx context.Context, guid, token string) ([]int32, error) {
	manifest, err := storage.manifest(ctx, guid, token)
	if err != nil {
		return nil, err
	}
	dims := make([]int32, len(manifest.Dimensions))
	for i := 0; i < len(manifest.Dimensions); i++ {
		dims[i] = int32(i)
	}

	return dims, nil
}

func (storage *storageURL) lines(ctx context.Context, guid string, dimension int32, token string) ([]int32, error) {
	manifest, err := storage.manifest(ctx, guid, token)
	if err != nil {
		return nil, err
	}

	if dimension >= int32(len(manifest.Dimensions)) {
		return nil, fmt.Errorf("index out of bounds")
	}
	return manifest.Dimensions[dimension], nil
}

func (storage *storageURL) list(ctx context.Context, token string) ([]string, error) {
	su := createServiceURL(storage.URL, token)
	names := make([]string, 0)

	for marker := (azblob.Marker{}); marker.NotDone(); {
		listContainer, err := su.ListContainersSegment(
			ctx,
			marker,
			azblob.ListContainersSegmentOptions{},
		)
		if err != nil {
			return nil, err
		}

		for _, val := range listContainer.ContainerItems {
			names = append(names, val.Name)
		}

		marker = listContainer.NextMarker
	}
	return names, nil
}

func createServiceURL(storageEndpoint url.URL, token string) azblob.ServiceURL {
	return azblob.NewServiceURL(
		storageEndpoint,
		azblob.NewPipeline(
			azblob.NewTokenCredential(token, nil),
			azblob.PipelineOptions{},
		),
	)
}
