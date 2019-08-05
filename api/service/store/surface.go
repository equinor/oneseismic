package store

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/equinor/seismic-cloud/api/config"
	"github.com/google/uuid"
)

type SurfaceStore interface {
	List(context.Context) ([]Surface, error)
	Download(context.Context, string) (io.Reader, error)
	Upload(context.Context, string, string, io.Reader) (string, error)
}

type (
	Surface struct {
		SurfaceID    string    `json:"surfaceID"`
		Link         string    `json:"link"`
		LastModified time.Time `json:"lastModified"`
	}
	SurfaceBlobStore struct {
		containerURL *azblob.ContainerURL
	}
	SurfaceLocalStore struct {
		localPath string
	}
)

func NewAzBlobStorage(accountName, accountKey, containerName string) (*SurfaceBlobStore, error) {

	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		log.Fatal("Invalid credentials with error: " + err.Error())
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

	return &SurfaceBlobStore{containerURL: &containerURL}, nil
}

func NewLocalStorage(localPath string) (*SurfaceLocalStore, error) {
	if len(localPath) == 0 {
		return &SurfaceLocalStore{localPath: "tmp"}, nil
	} else if _, err := os.Stat(localPath); os.IsNotExist(err) {
		os.MkdirAll(localPath, 0700)
	}
	return &SurfaceLocalStore{localPath: localPath}, nil
}

func (az *SurfaceBlobStore) List(ctx context.Context) ([]Surface, error) {

	var info []Surface
	for marker := (azblob.Marker{}); marker.NotDone(); {
		listBlob, err := az.containerURL.ListBlobsFlatSegment(ctx, marker, azblob.ListBlobsSegmentOptions{})
		if err != nil {
			return nil, err
		}

		marker = listBlob.NextMarker

		for _, blobInfo := range listBlob.Segment.BlobItems {
			info = append(info, Surface{
				SurfaceID:    blobInfo.Name,
				Link:         blobInfo.Name,
				LastModified: blobInfo.Properties.LastModified})
		}
	}
	return info, nil
}

func (local *SurfaceLocalStore) List(ctx context.Context) ([]Surface, error) {
	files, err := ioutil.ReadDir(local.localPath)
	if err != nil {
		return nil, err
	}
	var info []Surface
	for _, file := range files {
		info = append(info, Surface{
			SurfaceID:    file.Name(),
			Link:         file.Name(),
			LastModified: file.ModTime()})
	}

	return info, nil
}

func (az *SurfaceBlobStore) Download(ctx context.Context, fileName string) (io.Reader, error) {

	blobURL := az.containerURL.NewBlockBlobURL(fileName)

	downloadResponse, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return nil, err
	}
	retryReader := downloadResponse.Body(azblob.RetryReaderOptions{MaxRetryRequests: 3})

	return retryReader, nil
}

func (local *SurfaceLocalStore) Download(ctx context.Context, fileName string) (io.Reader, error) {

	file, err := os.Open(local.localPath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (az *SurfaceBlobStore) Upload(ctx context.Context, fn string, userID string, r io.Reader) (string, error) {
	blobURL := az.containerURL.NewBlockBlobURL(blobNameGenerator(fn))

	_, err := azblob.UploadStreamToBlockBlob(ctx, r, blobURL, azblob.UploadStreamToBlockBlobOptions{BufferSize: 20, MaxBuffers: 20})
	if err != nil {
		return "", err
	}

	b := blobURL.URL()
	return b.String(), nil
}

func (local *SurfaceLocalStore) Upload(ctx context.Context, fn string, userID string, r io.Reader) (string, error) {
	fo, err := os.Create(local.localPath + blobNameGenerator(fn))
	if err != nil {
		return "", nil
	}

	defer fo.Close()

	_, err = io.Copy(fo, r)
	if err != nil {
		return "", nil
	}

	return fo.Name(), nil
}

func blobNameGenerator(fileName string) string {
	return fileName + "_" + uuid.New().String()
}
