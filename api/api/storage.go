package api

import (
	"context"
	"net/url"
	"github.com/equinor/oneseismic/api/internal/message"
)

/*
* This is the abstraction of the storage used by OneSeismic
*
* It is p.t. basically a collection of methods but should
* be extended to hold at least the actual location (e.g. URL)
* to the storage. Currently, all callers pass the URL as params
* to method-calls, which can be avioded
*
* All methods may return at least two different errors:
*
*  - InternalError if something unexpected happened internally
*  - IllegalAccess if the provided authentication fails or user
*     is not authorized to perform the operation
*
 */
type AbstractStorage interface {
	/*
	* Get the manifest for the specified cube from the blob store.
	*
	* It's important that this is a blocking read, since this is the first
	* authorization mechanism in oneseismic. If the user (through the
	* on-behalf-token) does not have permissions to read the manifest, it
	* shouldn't be able to read the cube either. If so, no more processing
	* should be done, and the request discarded.
	*
	*
	* In addition to the standard errors, this method may also return
	*
	* - NonExistingError if the requested manifest does not exist
	*
	*
	*
	 */
	FetchManifest(ctx context.Context, token string, guid string) ([]byte, error)

	/*
	* Create a container which can contain BLOBs. This container is p.t.
	* only a factory for objects used to retrieve BLOBs, but it can easily
	* be extended with methods for uploading and listing.
	 */
	CreateBlobContainer(message.Task) (AbstractBlobContainer, error)

	/*
	* Retrieves the URL to which this blobstorage is connected
	*/
	GetUrl() *url.URL
}

type AbstractBlobContainer interface {
	ConstructBlobURL(string) AbstractBlobURL
}

type AbstractBlobURL interface {
	Download(context.Context) ([]byte, error)
}
