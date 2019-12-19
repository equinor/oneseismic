package store

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"time"

	"github.com/equinor/seismic-cloud-api/api/events"

	"github.com/Azure/azure-storage-blob-go/azblob"
	l "github.com/equinor/seismic-cloud-api/api/logger"
	"github.com/google/uuid"
)

type SurfaceStore interface {
	List(context.Context) ([]SurfaceMeta, error)
	Download(context.Context, string) (io.Reader, error)
	Upload(context.Context, string, string, io.Reader) (string, error)
}

type SurfaceMeta struct {
	SurfaceID    string    `json:"surfaceID"`
	Link         string    `json:"link"`
	LastModified time.Time `json:"lastModified"`
}

type (
	surfaceFileStore struct {
		localStore *LocalFileStore
	}

	surfaceBlobStore struct {
		blobStore *AzBlobStore
	}

	surfaceInMemoryStore struct {
		inMemoryStore *InMemoryStore
	}
)

func NewSurfaceStore(persistance interface{}) (SurfaceStore, error) {
	switch pers := persistance.(type) {
	case map[string][]byte:
		s, err := NewInMemoryStore(pers)
		if err != nil {
			return nil, events.E("new inmem store", err)
		}
		return &surfaceInMemoryStore{inMemoryStore: s}, nil
	case AzureBlobSettings:

		s, err := NewAzBlobStore(pers)
		if err != nil {
			return nil, events.E("new azure blob store", err)
		}
		return &surfaceBlobStore{blobStore: s}, nil
	case BasePath:
		s, err := NewLocalFileStore(pers)
		if err != nil {
			return nil, events.E("new local store", err)
		}
		return &surfaceFileStore{localStore: s}, nil
	default:
		return nil, events.E("No surface store selected")
	}
}

func (s *surfaceBlobStore) List(ctx context.Context) ([]SurfaceMeta, error) {
	az := s.blobStore
	var info []SurfaceMeta
	for marker := (azblob.Marker{}); marker.NotDone(); {
		listBlob, err := az.containerURL.ListBlobsFlatSegment(ctx, marker, azblob.ListBlobsSegmentOptions{})
		if err != nil {
			return nil, events.E("Could not list blobs", err)
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

func (s *surfaceFileStore) List(ctx context.Context) ([]SurfaceMeta, error) {
	local := s.localStore
	files, err := ioutil.ReadDir(string(local.basePath))
	if err != nil {
		return nil, events.E("Invalid local file store", err, events.NotFound)
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

func (s *surfaceInMemoryStore) List(ctx context.Context) ([]SurfaceMeta, error) {
	s.inMemoryStore.lock.Lock()
	defer s.inMemoryStore.lock.Unlock()
	var info []SurfaceMeta
	for k := range s.inMemoryStore.m {
		info = append(info, SurfaceMeta{
			SurfaceID: k,
			Link:      k})
	}
	sort.Slice(info, func(i, j int) bool { return info[i].SurfaceID < info[j].SurfaceID })

	return info, nil
}

func (s *surfaceBlobStore) Download(ctx context.Context, fileName string) (io.Reader, error) {
	az := s.blobStore
	blobURL := az.containerURL.NewBlockBlobURL(fileName)

	downloadResponse, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
	if err != nil {
		l.LogW("Surface download failed", l.Wrap(err), l.Kind(events.NotFound))
		return nil, err
	}
	l.LogI(fmt.Sprintf("Download: surfaceLength: %d bytes\n", downloadResponse.ContentLength()))
	retryReader := downloadResponse.Body(azblob.RetryReaderOptions{MaxRetryRequests: 3})

	return retryReader, nil
}

func (s *surfaceFileStore) Download(ctx context.Context, fileName string) (io.Reader, error) {
	local := s.localStore
	file, err := os.Open(path.Join(string(local.basePath), fileName))
	if err != nil {
		return nil, events.E("Could not open file from local store", err, events.NotFound)
	}
	return file, nil
}

func (s *surfaceBlobStore) Upload(ctx context.Context, fn string, userID string, r io.Reader) (string, error) {
	az := s.blobStore
	blobURL := az.containerURL.NewBlockBlobURL(blobNameGenerator(fn))

	_, err := azblob.UploadStreamToBlockBlob(ctx, r, blobURL, azblob.UploadStreamToBlockBlobOptions{BufferSize: az.bufferSize, MaxBuffers: az.maxBuffers})
	if err != nil {
		return "", events.E("Blob upload to block blob failed", err)
	}

	b := blobURL.URL()
	return b.String(), nil
}

func (s *surfaceFileStore) Upload(ctx context.Context, fn string, userID string, r io.Reader) (string, error) {

	local := s.localStore
	fo, err := os.Create(string(local.basePath) + blobNameGenerator(fn))
	if err != nil {
		return "", events.E("Could not create local file", err)
	}

	defer fo.Close()

	_, err = io.Copy(fo, r)
	if err != nil {
		return "", events.E("Could not copy to reader", err)
	}

	return fo.Name(), nil
}

func blobNameGenerator(fileName string) string {
	return fileName + "_" + uuid.New().String()
}

func (s *surfaceInMemoryStore) Download(ctx context.Context, fileName string) (io.Reader, error) {
	s.inMemoryStore.lock.RLock()
	defer s.inMemoryStore.lock.RUnlock()
	surface, ok := s.inMemoryStore.m[fileName]
	if !ok {
		return nil, events.E("Surface not found", events.NotFound)
	}
	buf := &bytes.Buffer{}
	_, err := buf.Write(surface)
	if err != nil {
		return nil, events.E("Surface write bytes error", err, events.Marshalling)
	}
	return buf, nil
}

func (s *surfaceInMemoryStore) Upload(ctx context.Context, fn string, userID string, r io.Reader) (string, error) {
	s.inMemoryStore.lock.Lock()
	defer s.inMemoryStore.lock.Unlock()
	buf := &bytes.Buffer{}

	_, err := io.Copy(buf, r)
	if err != nil {
		return "", events.E("Surface write bytes error", err, events.Marshalling)
	}
	s.inMemoryStore.m[fn] = buf.Bytes()
	return fn, nil
}
