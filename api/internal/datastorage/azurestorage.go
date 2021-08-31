package datastorage

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"

	"github.com/Azure/azure-pipeline-go/pipeline"
	"github.com/Azure/azure-storage-blob-go/azblob"

	"github.com/equinor/oneseismic/api/api"
	"github.com/equinor/oneseismic/api/internal/message"
)

func NewAzureStorage(baseurl string) *AzureStorage {
	retval := AzureStorage{}
	myUrl, err := url.Parse(baseurl) ; if err != nil {
		panic(fmt.Sprintf("Malformed URL %v", baseurl))
	}
	retval.baseurl = myUrl
	return &retval
}
type AzureStorage struct {
	baseurl *url.URL
}
func (s AzureStorage) GetUrl() *url.URL { return s.baseurl }

func (s AzureStorage) FetchManifest(
	ctx context.Context,
	token string,
	guid string,
) ([]byte, error) {
	keys := ctx.Value("keys").(map[string]string)
	pid := keys["pid"]

	container, err := url.Parse(fmt.Sprintf("%s/%s", s.baseurl, guid))
	if err != nil {
		log.Printf("pid=%s %v", pid, err)
		return nil, err
	}

	ret, err := s.fetchManifestImpl(ctx, container, token)
	if err != nil {
		log.Printf("pid=%s %v", pid, err)
		return nil, err
	}
	return ret, nil
}

/*
 * Azure-implementation. Original comment left below - should be tidied up
 *
 *
 * Get the manifest for the cube from the blob store.
 *
 * It's important that this is a blocking read, since this is the first
 * authorization mechanism in oneseismic. If the user (through the
 * on-behalf-token) does not have permissions to read the manifest, it
 * shouldn't be able to read the cube either. If so, no more processing should
 * be done, and the request discarded.
 */
func (s AzureStorage) fetchManifestImpl(
	ctx context.Context,
	containerURL *url.URL,
	token string,
) ([]byte, error) {
	credentials := azblob.NewTokenCredential(token, nil)
	pipeline := azblob.NewPipeline(credentials, azblob.PipelineOptions{})
	container := azblob.NewContainerURL(*containerURL, pipeline)
	blob := container.NewBlobURL("manifest.json")

	dl, err := blob.Download(
		ctx,
		0, /* offset */
		azblob.CountToEnd,
		azblob.BlobAccessConditions{},
		false, /* content-get-md5 */
		azblob.ClientProvidedKeyOptions{},
	)
	if err != nil {
		return nil, err
	}

	body := dl.Body(azblob.RetryReaderOptions{})
	defer body.Close()
	return ioutil.ReadAll(body)
}

func (s AzureStorage) CreateBlobContainer(task message.Task) (api.AbstractBlobContainer, error) {
	endpoint := task.StorageEndpoint
	guid     := task.Guid
	container, err := url.Parse(fmt.Sprintf("%s/%s", endpoint, guid))
	if err != nil {
		err = fmt.Errorf("Container URL would be malformed: %w", err)
		return AzureBlobContainer{}, err
	}

	credentials := azblob.NewTokenCredential(task.Token, nil)
	pipeline    := azblob.NewPipeline(credentials, azblob.PipelineOptions{})
	return AzureBlobContainer{
		container: azblob.NewContainerURL(*container, pipeline),
		url: *container,
		pipeline: pipeline}, nil
}

type AzureBlobContainer struct {
	container azblob.ContainerURL
	url url.URL
	pipeline pipeline.Pipeline
}

func (c AzureBlobContainer) ConstructBlobURL(id string) api.AbstractBlobURL {
	return AzureBlobUrl{
		url: c.container.NewBlobURL(id),
	}
}

type AzureBlobUrl struct {
	url azblob.BlobURL
}

func (blob AzureBlobUrl) Download(ctx context.Context) ([]byte, error) {
	dl, err := blob.url.Download(
		ctx,
		0,
		azblob.CountToEnd,
		azblob.BlobAccessConditions{},
		false,
		azblob.ClientProvidedKeyOptions {},
	)	
	if err != nil {
		return nil, err
	}

	body := dl.Body(azblob.RetryReaderOptions{MaxRetryRequests: 1}) // TODO: introduce maxRetries
	defer body.Close()
	return ioutil.ReadAll(body)
}
