package storage

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"github.com/equinor/oneseismic/api/internal"
	"github.com/equinor/oneseismic/api/internal/util"
)


type entity struct {
	Data []byte
	Tag  *string
}

/*
 *  Minimal interface for fetching blobs/files from storage. This hides a lot of
 *  feature and details about the underlying storage from the rest of the
 *  system, making it easy to swap out the storage provider. This means testing
 *  becomes easier through custom storage implementations.
 */
type StorageClient interface {
	/*
	 * Get a blob or file from storage
	 */
	Get(ctx context.Context, blob string) (*entity, error)
}

/*
 * Azure Blob Store implementation of a oneseismic StorageClient
 */
type AzStorage struct {
	cache blobCache
}

func (c *AzStorage) Get(ctx context.Context, blob string) (*entity, error) {
	bloburl, err := url.Parse(blob)
	if err != nil {
		log.Printf("Invalid URL: %s", blob)
		return nil, internal.InternalError(err.Error())
	}

	key := newCacheKey(bloburl)
	cached, hit := c.cache.get(key)

	cold, err := c.download(ctx, bloburl, cached.Tag)
	if err == nil {
		/* nil means the azblob.Download succeeded *and* was not etag match */
		if hit {
			/* This probably means expired ETag, which again means a fragment
			* has been updated since cached. This should not happen in a
			* healthy system and must be investigated immediately.
			 */
			log.Printf(
				"ETag (= %s) expired for %v; investigate immediately",
				*cached.Tag,
				bloburl,
			)
			return nil, internal.NewInternalError()
		} else {
			// This is good; not in cache, so clean fetch was expected.
			go c.cache.set(key, *cold)
			return cold, nil
		}
	}

	switch e := err.(type) {
	case azblob.StorageError:
		status := e.Response().StatusCode
		switch status {
		case http.StatusNotModified:
			return &cached, nil
		case http.StatusNotFound:
			msg := fmt.Sprintf("Not found: %s/%s", bloburl.Host, bloburl.Path)
			return nil, internal.NotFound(msg)
		case http.StatusForbidden:
			return nil, internal.PermissionDeniedFromStatus(status)
		case http.StatusUnauthorized:
			return nil, internal.PermissionDeniedFromStatus(status)
		default:
			log.Printf("Unhandled azblob.StorageError: %v", err)
			return nil, internal.InternalError(err.Error())
		}
	default:
		log.Printf("Unhandled error type %T (= %v)", e, e)
		return nil, internal.InternalError(err.Error())
	}

	return nil, err
}

func (c *AzStorage) download(
	ctx     context.Context,
	bloburl *url.URL,
	etag    *string,
) (*entity, error) {
	client, err := azblob.NewBlobClientWithNoCredential(bloburl.String(), nil)
	if err != nil {
		return nil, err
	}

	options := &azblob.DownloadBlobOptions{
		BlobAccessConditions: &azblob.BlobAccessConditions{
			ModifiedAccessConditions : &azblob.ModifiedAccessConditions{
				IfNoneMatch: etag,
			},
		},
	}

	dl, err := client.Download(ctx, options)
	if err != nil {
		return nil, util.UnpackAzStorageError(err)
	}
	body := dl.Body(&azblob.RetryReaderOptions{})
	defer body.Close()
	data, err := ioutil.ReadAll(body)
	return &entity{ Data: data, Tag: dl.ETag }, err
}

func NewAzStorage(cache blobCache) *AzStorage {
	return &AzStorage{cache: cache}
}

/*
 * Creates a cache key from the host + path
 */
func newCacheKey(bloburl *url.URL) string {
	return fmt.Sprintf("%s/%s", bloburl.Host, bloburl.Path)
}
