package store

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"sync"

	"github.com/Azure/azure-storage-blob-go/azblob"
)

type (
	SurfaceBlobStore struct {
		containerURL *azblob.ContainerURL
		bufferSize   int
		maxBuffers   int
	}

	SurfaceInMemoryStore struct {
		m    map[string][]byte
		lock sync.RWMutex
	}

	LocalFileStore struct {
		basePath string
	}
)

type AzureBlobSettings struct {
	AccountName   string
	AccountKey    string
	ContainerName string
}

type SurfaceStore interface {
	Download(surfaceID string) (Surface, error)
}

type Surface struct {
	Points []*Point
}

type Point struct {
	X uint64
	Y uint64
	Z uint64
}

func NewSurfaceStore(persistance interface{}) (SurfaceStore, error) {
	switch persistance.(type) {
	case map[string][]byte:
		s, err := NewInMemoryStore(persistance.(map[string][]byte))
		if err != nil {
			return nil, err
		}
		return s, nil
	case AzureBlobSettings:
		azbs := persistance.(AzureBlobSettings)
		s, err := NewAzBlobStore(azbs.AccountName, azbs.AccountKey, azbs.ContainerName)
		if err != nil {
			return nil, err
		}
		return s, nil
	case string:
		s, err := NewLocalFileStore(persistance.(string))
		if err != nil {
			return nil, err
		}
		return s, nil
	default:
		return nil, fmt.Errorf("No surface store selected")
	}
}

func decodeSurface(in io.Reader) (Surface, error) {
	var surface Surface
	for {
		var p struct {
			X, Y, Z uint64
		}
		err := binary.Read(in, binary.LittleEndian, &p)
		if err == io.EOF {
			break
		}
		if err != nil {
			return surface, err
		}
		surface.Points = append(surface.Points, &Point{X: p.X, Y: p.Y, Z: p.Z})
	}
	return surface, nil
}

func (az *SurfaceBlobStore) Download(surfaceID string) (Surface, error) {
	var surf Surface
	blobURL := az.containerURL.NewBlockBlobURL(surfaceID)
	ctx := context.Background()
	downloadResponse, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return surf, err
	}
	fmt.Printf("Download: surfaceLength: %d bytes\n", downloadResponse.ContentLength())
	retryReader := downloadResponse.Body(azblob.RetryReaderOptions{MaxRetryRequests: 3})
	surf, err = decodeSurface(retryReader)
	if err != nil {
		fmt.Println("decode fail")
	}
	return surf, nil
}

func (s *SurfaceInMemoryStore) Download(surfaceID string) (Surface, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var surf Surface
	surface, ok := s.m[surfaceID]
	if !ok {
		return surf, fmt.Errorf("Surface not found")
	}
	buf := &bytes.Buffer{}
	_, err := buf.Write(surface)
	if err != nil {
		return surf, err
	}
	surf, err = decodeSurface(buf)
	if err != nil {
		fmt.Println("decode fail")
	}
	return surf, nil
}

func (s *LocalFileStore) Download(surfaceID string) (Surface, error) {
	var surf Surface
	file, err := os.Open(path.Join(string(s.basePath), surfaceID))
	if err != nil {
		return surf, err
	}
	surf, err = decodeSurface(file)
	if err != nil {
		fmt.Println("decode fail")
	}
	return surf, nil
}

func NewAzBlobStore(accountName, accountKey, containerName string) (*SurfaceBlobStore, error) {

	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, err
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	azStorageURL := "https://%s.blob.core.windows.net/%s"
	u, err := url.Parse(
		fmt.Sprintf(azStorageURL,
			accountName,
			containerName))
	if err != nil {
		return nil, err
	}
	containerURL := azblob.NewContainerURL(*u, p)

	return &SurfaceBlobStore{
		containerURL: &containerURL,
		bufferSize:   2 * 1024 * 1024,
		maxBuffers:   100}, nil
}

func NewInMemoryStore(m map[string][]byte) (*SurfaceInMemoryStore, error) {
	return &SurfaceInMemoryStore{m: m}, nil
}

func NewLocalFileStore(basePath string) (*LocalFileStore, error) {
	if len(basePath) == 0 {
		return nil, fmt.Errorf("invalid file path")
	}

	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		os.MkdirAll(basePath, 0700)
	} else if err != nil {
		return nil, err
	}

	return &LocalFileStore{basePath}, nil
}
