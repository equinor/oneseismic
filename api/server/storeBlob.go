package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/prometheus/common/log"
)

type store interface {
	list(ctx context.Context) ([]string, error)
	manifest(ctx context.Context, guid string) (*Manifest, error)
	dimensions(ctx context.Context, guid string) ([]int32, error)
	lines(ctx context.Context, guid string, dimension int32) ([]int32, error)
}

type serviceURL struct {
	azblob.ServiceURL
}

type Manifest struct {
	Dimensions [][]int32 `json:"dimensions"`
	Samples    int32     `json:"samples"`
}

func (sURL *serviceURL) manifest(ctx context.Context, guid string) (*Manifest, error) {
	cURL := sURL.NewContainerURL(guid)

	blobURL := cURL.NewBlockBlobURL("manifest.json")
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

func (sURL *serviceURL) dimensions(ctx context.Context, guid string) ([]int32, error) {
	manifest, err := sURL.manifest(ctx, guid)
	if err != nil {
		log.Errorf("cube: %s", err)
		return nil, err
	}
	dims := make([]int32, len(manifest.Dimensions))
	for i := 0; i < len(manifest.Dimensions); i++ {
		dims[i] = int32(i)
	}

	return dims, nil
}

func (sURL *serviceURL) lines(ctx context.Context, guid string, dimension int32) ([]int32, error) {
	manifest, err := sURL.manifest(ctx, guid)
	if err != nil {
		log.Errorf("cube: %s", err)
		return nil, err
	}

	if dimension >= int32(len(manifest.Dimensions)) {
		return nil, fmt.Errorf("index out of bounds")
	}
	return manifest.Dimensions[dimension], nil
}

func (sURL *serviceURL) list(ctx context.Context) ([]string, error) {
	names := make([]string, 0)

	for marker := (azblob.Marker{}); marker.NotDone(); {
		listContainer, err := sURL.ListContainersSegment(
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

func newServiceURL(storageURL, accountName, accountKey string) (*serviceURL, error) {

	uri, err := url.Parse(
		fmt.Sprintf(storageURL,
			accountName))
	if err != nil {
		return nil, err
	}
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, err
	}

	sURL := azblob.NewServiceURL(
		*uri,
		azblob.NewPipeline(credential, azblob.PipelineOptions{}),
	)

	return &serviceURL{sURL}, nil
}
