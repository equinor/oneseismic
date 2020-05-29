package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/Azure/azure-storage-blob-go/azblob"
	l "github.com/equinor/oneseismic/api/logger"
)

type store interface {
	list(ctx context.Context, accountName, token string) ([]string, error)
	manifest(ctx context.Context, accountName, guid, token string) (*Manifest, error)
	dimensions(ctx context.Context, accountName, guid, token string) ([]int32, error)
	lines(ctx context.Context, accountName, guid string, dimension int32, token string) ([]int32, error)
}

type storageURL struct {
	string
}

type Manifest struct {
	Dimensions [][]int32 `json:"dimensions"`
	Samples    int32     `json:"samples"`
}

func (sURL *storageURL) manifest(ctx context.Context, accountName, guid, token string) (*Manifest, error) {
	uri, err := parseStorageURL(accountName, sURL.string)
	if err != nil {
		return nil, err
	}
	blobURL := createServiceURL(*uri, token).NewContainerURL(guid).NewBlockBlobURL("manifest.json")
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

func (sURL *storageURL) dimensions(ctx context.Context, accountName, guid, token string) ([]int32, error) {
	manifest, err := sURL.manifest(ctx, accountName, guid, token)
	if err != nil {
		return nil, err
	}
	dims := make([]int32, len(manifest.Dimensions))
	for i := 0; i < len(manifest.Dimensions); i++ {
		dims[i] = int32(i)
	}

	return dims, nil
}

func (sURL *storageURL) lines(ctx context.Context, accountName, guid string, dimension int32, token string) ([]int32, error) {
	manifest, err := sURL.manifest(ctx, accountName, guid, token)
	if err != nil {
		return nil, err
	}

	if dimension >= int32(len(manifest.Dimensions)) {
		return nil, fmt.Errorf("index out of bounds")
	}
	return manifest.Dimensions[dimension], nil
}

func (sURL *storageURL) list(ctx context.Context, accountName, token string) ([]string, error) {
	uri, err := parseStorageURL(accountName, sURL.string)
	if err != nil {
		return nil, err
	}
	su := createServiceURL(*uri, token)
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

func parseStorageURL(accountName, storageURL string) (*url.URL, error) {
	uri, err := url.Parse(
		fmt.Sprintf(storageURL,
			accountName))
	if err != nil {
		l.LogE("creating storage url", err)
		return nil, err
	}
	return uri, nil
}

func createServiceURL(uri url.URL, token string) azblob.ServiceURL {
	credential := azblob.NewTokenCredential(token, nil)

	return azblob.NewServiceURL(
		uri,
		azblob.NewPipeline(credential, azblob.PipelineOptions{}),
	)

}
