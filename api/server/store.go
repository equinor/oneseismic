package server

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Azure/azure-storage-blob-go/azblob"
)

type manifestStore interface {
	list(ctx context.Context) ([]string, error)
}

type ServiceURL struct {
	azblob.ServiceURL
}

type AzureBlobSettings struct {
	StorageURL  string
	AccountName string
	AccountKey  string
}

func (sURL *ServiceURL) list(ctx context.Context) ([]string, error) {
	names := make([]string, 0)

	for marker := (azblob.Marker{}); marker.NotDone(); {
		listContainer, err := sURL.ListContainersSegment(ctx, marker, azblob.ListContainersSegmentOptions{})
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

func NewServiceURL(az AzureBlobSettings) (*ServiceURL, error) {

	uri, err := url.Parse(
		fmt.Sprintf(az.StorageURL,
			az.AccountName))
	if err != nil {
		return nil, err
	}

	credential, err := azblob.NewSharedKeyCredential(az.AccountName, az.AccountKey)
	if err != nil {
		return nil, err
	}

	sURL := azblob.NewServiceURL(
		*uri,
		azblob.NewPipeline(credential, azblob.PipelineOptions{}),
	)

	return &ServiceURL{sURL}, err
}
