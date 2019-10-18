package store

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"sort"
	"sync"
	"time"

	"github.com/equinor/seismic-cloud/api/events"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/equinor/seismic-cloud/api/config"
	l "github.com/equinor/seismic-cloud/api/logger"
	"github.com/google/uuid"
)

type SurfaceStore interface {
	List(context.Context) ([]SurfaceMeta, error)
	Download(context.Context, string) (io.Reader, error)
	Upload(context.Context, string, string, io.Reader) (string, error)
}

type (
	SurfaceMeta struct {
		SurfaceID    string    `json:"surfaceID"`
		Link         string    `json:"link"`
		LastModified time.Time `json:"lastModified"`
	}
	surfaceBlobStore struct {
		containerURL *azblob.ContainerURL
		bufferSize   int
		maxBuffers   int
	}
	surfaceLocalStore struct {
		localPath string
	}

	surfaceInMemoryStore struct {
		m    map[string][]byte
		lock sync.RWMutex
	}
)

type AzureBlobSettings struct {
	AccountName   string
	AccountKey    string
	ContainerName string
}

func NewSurfaceStore(persistance interface{}) (SurfaceStore, error) {
	switch persistance.(type) {
	case map[string][]byte:
		return &surfaceInMemoryStore{m: persistance.(map[string][]byte)}, nil
	case AzureBlobSettings:
		azbs := persistance.(AzureBlobSettings)
		return azBlobStorage(azbs.AccountName, azbs.AccountKey, azbs.ContainerName)
	case string:
		return localStorage(persistance.(string))
	default:
		return nil, events.E("No surface store selected")
	}

}

func azBlobStorage(accountName, accountKey, containerName string) (*surfaceBlobStore, error) {

	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		l.LogE("surfaceStore download", "Invalid credentials", err)
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

	return &surfaceBlobStore{containerURL: &containerURL,
		bufferSize: 2 * 1024 * 1024,
		maxBuffers: 100}, nil
}

func localStorage(localPath string) (*surfaceLocalStore, error) {
	op := events.Op("service.localStorage")
	if len(localPath) == 0 {
		return nil, events.E(op, "localPath cannot be empty")
	}

	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		os.MkdirAll(localPath, 0700)
	} else if err != nil {
		return nil, events.E(op, "accessing localPath failed", err)
	}

	return &surfaceLocalStore{localPath: localPath}, nil
}

func (az *surfaceBlobStore) List(ctx context.Context) ([]SurfaceMeta, error) {

	var info []SurfaceMeta
	for marker := (azblob.Marker{}); marker.NotDone(); {
		listBlob, err := az.containerURL.ListBlobsFlatSegment(ctx, marker, azblob.ListBlobsSegmentOptions{})
		if err != nil {
			return nil, err
		}

		marker = listBlob.NextMarker

		for _, blobInfo := range listBlob.Segment.BlobItems {
			info = append(info, SurfaceMeta{
				SurfaceID:    blobInfo.Name,
				Link:         blobInfo.Name,
				LastModified: blobInfo.Properties.LastModified})
		}
	}
	return info, nil
}

func (local *surfaceLocalStore) List(ctx context.Context) ([]SurfaceMeta, error) {
	files, err := ioutil.ReadDir(local.localPath)
	if err != nil {
		return nil, err
	}
	var info []SurfaceMeta
	for _, file := range files {
		info = append(info, SurfaceMeta{
			SurfaceID:    file.Name(),
			Link:         file.Name(),
			LastModified: file.ModTime()})
	}

	return info, nil
}

func (inMem *surfaceInMemoryStore) List(ctx context.Context) ([]SurfaceMeta, error) {
	inMem.lock.Lock()
	defer inMem.lock.Unlock()
	var info []SurfaceMeta
	for k := range inMem.m {
		info = append(info, SurfaceMeta{
			SurfaceID: k,
			Link:      k})
	}
	sort.Slice(info, func(i, j int) bool { return info[i].SurfaceID < info[j].SurfaceID })

	return info, nil
}

func (az *surfaceBlobStore) Download(ctx context.Context, fileName string) (io.Reader, error) {

	blobURL := az.containerURL.NewBlockBlobURL(fileName)

	downloadResponse, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
	if err != nil {
		l.LogW("surface store download", "Surface download failed:", l.Wrap(err))
		return nil, err
	}
	l.LogI("surfaceStore download", fmt.Sprintf("Download: surfaceLength: %d bytes\n", downloadResponse.ContentLength()))
	retryReader := downloadResponse.Body(azblob.RetryReaderOptions{MaxRetryRequests: 3})

	return retryReader, nil
}

func (local *surfaceLocalStore) Download(ctx context.Context, fileName string) (io.Reader, error) {

	file, err := os.Open(path.Join(local.localPath, fileName))
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (az *surfaceBlobStore) Upload(ctx context.Context, fn string, userID string, r io.Reader) (string, error) {
	blobURL := az.containerURL.NewBlockBlobURL(blobNameGenerator(fn))

	_, err := azblob.UploadStreamToBlockBlob(ctx, r, blobURL, azblob.UploadStreamToBlockBlobOptions{BufferSize: az.bufferSize, MaxBuffers: az.maxBuffers})
	if err != nil {
		return "", err
	}

	b := blobURL.URL()
	return b.String(), nil
}

func (local *surfaceLocalStore) Upload(ctx context.Context, fn string, userID string, r io.Reader) (string, error) {
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

func (inMem *surfaceInMemoryStore) Download(ctx context.Context, fileName string) (io.Reader, error) {
	inMem.lock.RLock()
	defer inMem.lock.RUnlock()
	surface, ok := inMem.m[fileName]
	if !ok {
		return nil, events.E("Surface not found")
	}
	buf := &bytes.Buffer{}
	_, err := buf.Write(surface)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func (inMem *surfaceInMemoryStore) Upload(ctx context.Context, fn string, userID string, r io.Reader) (string, error) {
	inMem.lock.Lock()
	defer inMem.lock.Unlock()
	buf := &bytes.Buffer{}

	_, err := io.Copy(buf, r)
	if err != nil {
		return "", err
	}
	inMem.m[fn] = buf.Bytes()
	return fn, nil
}
