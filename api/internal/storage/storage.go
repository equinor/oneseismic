package storage

import (
	"context"
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
