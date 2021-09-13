package api

import (
	"context"
)

/*
* This is the abstraction of the storage used by OneSeismic
 */
type AbstractStorage interface {
	/*
	* Requests are for fragment of some resource. Thus, the provided
	* string ("request") identifies both the resource itself (in our case
	* a cube-uuid) and the fragment (eg. "manifest.json" or a data-fragment
	* identified like "src/64-64-64/0-0-0.f32").
	*
	* This means that requests must follow the URI-specification at
	*
	*     https://en.wikipedia.org/wiki/URI_fragment
	*
	* in particular the fragment is separated from the resource-identifier
	* with a #-sign
	*
	* The method returns the requested fragment or one of the following errors
	*
	*  - IllegalInputError if the request or provided credenials are malformed
	*  - IllegalAccess if the provided credentials fail to authenticate or user
	*     is not authorized to perform the operation
	*  - NonExistingError if the requested resource or fragment does not exist
	*  - InternalError if something unexpected happened
	*/
	Get(ctx context.Context, credentials string, request string) ([]byte, error)

	/*
	* Returns endpoint for this storage-instance.
	*
	* TODO:
	* The method exists because the query-server needs to (transparently)
	* retrieve this in order to pass it to the fetch-server. If we decide
	* to give the fetch-server full responsibility for its backing-storage
	* this method could be removed.
	*/
	GetEndpoint() string

	/*
	* Returns which kind of storage this instance is.
	*
	* TODO:
	* The method exists because the query-server needs to (transparently)
	* retrieve this in order to pass it to the fetch-server. If we decide
	* to give the fetch-server full responsibility for its backing-storage
	* this method could be removed.
	*/
	GetKind() string
}
