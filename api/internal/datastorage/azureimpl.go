package datastorage

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strings"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/equinor/oneseismic/api/api"
)

type AzureStorage struct {
	kind string  // not strictly necessary, but lets the factory decide
	baseurl string
}
func NewAzureStorage(kind string, baseurl string) *AzureStorage {
	return &AzureStorage{kind, baseurl}
}
func (s AzureStorage) GetEndpoint() string { return s.baseurl }
func (s AzureStorage) GetKind() string { return s.kind }

func (s AzureStorage) getFullUri(ctx context.Context, resource string) (*url.URL, error) {
	ret, err := url.Parse(fmt.Sprintf("%s/%s", s.baseurl, resource)); if err != nil {
		log.Printf("%v", err)
		return nil, api.NewInternalError(fmt.Sprintf("URL would be malformed for %v", resource))
	}
	return ret, nil
}

func (s AzureStorage) Get(
	ctx context.Context,
	creds string,
	request string,
) ([]byte, error) {
	pid := api.GetPid(ctx)

	blob, err := s.createBlobUrl(ctx, creds, request); if err != nil {
		log.Printf("pid=%s %v", pid, err)
		return nil, err
	}

	dl, err := blob.Download(
		ctx,
		0, /* offset */
		azblob.CountToEnd,
		azblob.BlobAccessConditions{},
		false, /* content-get-md5 */
		azblob.ClientProvidedKeyOptions{},
	)
	if err != nil {
		log.Printf("pid=%s %v", pid, err)
		return nil, api.NewNonExistingError(
			fmt.Sprintf("%s does not exist", request))
	}

	body := dl.Body(azblob.RetryReaderOptions{})
	defer body.Close()
	retval, err := ioutil.ReadAll(body); if err != nil {
		log.Printf("pid=%s %v", pid, err)
		return nil, api.NewInternalError(
				fmt.Sprintf("Failed reading %v", request))
	}
	return retval, nil
}

/*
* Internal helper to construct the azblob.BlobURL object.
* Could probably be inlined. Errors can just be forwarded
* by caller unless stricter control is required.
*/
func (s AzureStorage)createBlobUrl(
	ctx context.Context,
	credentials string,
	request string,
) (*azblob.BlobURL, error) {
	// In Azure we must construct a container representing the resource,
	// then request the fragment from the container. This allows to control 
	// access on the resource (cube) level, which is what we want.
	//
	// Hence we parse and verify the full URI to resource and fragment,
	// then strip off the fragment-part in the URI to address the container.
	containerUri, err := url.Parse(fmt.Sprintf("%s/%s", s.baseurl, request))
	if err != nil {
		log.Printf("%v", err)
		return nil, api.NewIllegalInputError(
			fmt.Sprintf("URL would be malformed for %v", request))
	}

	// Keep and verify fragment, then clear it to only address the resource
	fragment := containerUri.Fragment
	if fragment == "" {
		log.Printf("Missing fragment in %v", containerUri)
		return nil, api.NewIllegalInputError(
			fmt.Sprintf("No fragment specified in %v", request))
	}
	containerUri.Fragment = "" // See e.g. https://stackoverflow.com/a/55299809

	// Figure out what kind of credentials we have and construct the
	// corresponding azblob.Credential-object
	//log.Printf("Credentials string: %v", creds)
	creds := azblob.NewAnonymousCredential()
	kind, values, err := api.DecodeCredentials(credentials)
	if err != nil { return nil, err }
	switch(strings.ToLower(kind)) {
		case "obo":
			// OBO-based credentials is a token
			token, ok := values["token"]; if !ok {
				return nil, api.NewInternalError(
					   fmt.Sprintf("Missing OBO-token for %v", request))
			}
			creds = azblob.NewTokenCredential(token, nil)
		case "saas":
			// SaaS-based credentials is a token/cookie to be appended to the
			// query. Since the URI is already constructed and theoretically
			// can contain query-params, append the token/cookie
			token, ok := values["token"]; if !ok {
				return nil, api.NewInternalError(
					   fmt.Sprintf("Missing SaaS-token for %v", request))
			}
			creds = azblob.NewAnonymousCredential()
			containerUri.RawQuery += token
		default:
			log.Printf("Failed to parse credentials encoded in %v", credentials)
			return nil, api.NewIllegalInputError("Illegal credentials")
	}

	pipeline := azblob.NewPipeline(creds, azblob.PipelineOptions{})
	container := azblob.NewContainerURL(*containerUri, pipeline)
	blob := container.NewBlobURL(fragment)
	return &blob, nil
}
