package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	azb "github.com/Azure/azure-storage-blob-go/azblob"

	"github.com/equinor/seismic-cloud/api/events"
	l "github.com/equinor/seismic-cloud/api/logger"
	seismic_core "github.com/equinor/seismic-cloud/api/proto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ManifestStore interface {
	Download(context.Context, string) (*Manifest, error)
	Upload(context.Context, string, Manifest) error
}

type Manifest seismic_core.Geometry

type (
	manifestDbStore struct {
		dbStore *MongoDbStore
	}

	manifestBlobStore struct {
		blobStore *AzBlobStore
	}
)

func NewManifestStore(persistance interface{}) (ManifestStore, error) {

	switch persistance := persistance.(type) {
	case AzureBlobSettings:

		s, err := NewAzBlobStore(persistance)
		if err != nil {
			return nil, events.E("new azure blob store", err)
		}
		return &manifestBlobStore{blobStore: s}, nil
	case ConnStr:
		s, err := NewMongoDbStore(persistance)
		if err != nil {
			return nil, events.E("new mongo db store", err)
		}
		return &manifestDbStore{dbStore: s}, nil
	default:
		return nil, events.E("No manifest store persistance selected", events.ErrorLevel)
	}
}

func (mbs *manifestBlobStore) Download(ctx context.Context, manifestID string) (*Manifest, error) {
	mani := &Manifest{}

	blobURL := mbs.blobStore.containerURL.NewBlockBlobURL(manifestID)

	resp, err := blobURL.Download(
		ctx,
		0,
		azb.CountToEnd,
		azb.BlobAccessConditions{},
		false,
	)
	if err != nil {
		return mani, events.E("Download from blobstore", err, events.Marshalling, events.ErrorLevel)
	}
	b, err := ioutil.ReadAll(resp.Body(azb.RetryReaderOptions{}))
	if err != nil {
		return mani, events.E("Could not read manifest from blob store", err)
	}
	err = json.Unmarshal(b, mani)
	if err != nil {
		return mani, events.E("Unmarshaling to Manifest", err, events.Marshalling, events.ErrorLevel)
	}
	return mani, nil
}

func (mbs *manifestBlobStore) Upload(ctx context.Context, manifestID string, manifest Manifest) error {

	j, err := json.Marshal(manifest)

	if err != nil {
		return events.E("Marshaling to Manifest", err, events.Marshalling)
	}

	blobURL := mbs.blobStore.containerURL.NewBlockBlobURL(manifestID)

	// resp, err := blobURL.UploadUpload(ctx, body io.ReadSeeker, h BlobHTTPHeaders, metadata Metadata, ac BlobAccessConditions)
	_, err = azb.UploadStreamToBlockBlob(
		ctx,
		bytes.NewBuffer(j),
		blobURL,
		azb.UploadStreamToBlockBlobOptions{})

	if err != nil {
		return events.E("Upload to blobstore", err)
	}

	return nil
}

func (s *manifestDbStore) Download(ctx context.Context, id string) (*Manifest, error) {
	m := s.dbStore
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mani := &Manifest{}
	client, err := mongo.Connect(dbCtx, options.Client().ApplyURI(string(m.connString)))
	if err != nil {
		return mani, err
	}
	defer func() {
		err := client.Disconnect(dbCtx)
		if err != nil {
			l.LogE("Disconnect manifest store", err)
		}
	}()
	collection := client.Database("seismiccloud").Collection("manifests")
	err = collection.FindOne(dbCtx, bson.D{bson.E{Key: "basename", Value: id}}).Decode(mani)
	if err != nil {
		return mani, events.E("Finding and unmarshaling to Manifest", err, events.Marshalling)
	}
	l.LogI(fmt.Sprintf("Connected to manifest DB and fetched file %s", id))
	return mani, nil
}

func (s *manifestDbStore) Upload(ctx context.Context, id string, manifest Manifest) error {

	return events.E("Not implemented", events.CriticalLevel)
}
