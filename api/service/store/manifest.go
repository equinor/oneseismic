package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/equinor/seismic-cloud-api/api/events"
	l "github.com/equinor/seismic-cloud-api/api/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ManifestStore interface {
	Fetch(context.Context, string) (Manifest, error)
	List(context.Context) ([]Manifest, error)
}

type (
	manifestFileStore struct {
		localStore *LocalFileStore
	}

	manifestInMemoryStore struct {
		inMemoryStore *InMemoryStore
	}

	manifestDbStore struct {
		dbStore *MongoDbStore
	}

	manifestBlobStore struct {
		blobStore *AzBlobStore
	}
)

func NewManifestStore(persistance interface{}) (ManifestStore, error) {
	const op = events.Op("store.NewManifestStore")
	switch persistance := persistance.(type) {
	case map[string][]byte:
		s, err := NewInMemoryStore(persistance)
		if err != nil {
			return nil, events.E(op, "new inmem store", err)
		}
		return &manifestInMemoryStore{inMemoryStore: s}, nil
	case AzureBlobSettings:

		s, err := NewAzBlobStore(persistance.AccountName, persistance.AccountKey, persistance.ContainerName)
		if err != nil {
			return nil, events.E(op, "new azure blob store", err)
		}
		return &manifestBlobStore{blobStore: s}, nil
	case ConnStr:
		s, err := NewMongoDbStore(persistance)
		if err != nil {
			return nil, events.E(op, "new mongo db store", err)
		}
		return &manifestDbStore{dbStore: s}, nil
	case BasePath:
		s, err := NewLocalFileStore(persistance)
		if err != nil {
			return nil, events.E(op, "new local store", err)
		}
		return &manifestFileStore{localStore: s}, nil
	default:
		return nil, events.E(op, "No manifest store persistance selected")
	}
}

type Manifest struct {
	Basename   string `json:"basename"`
	Cubexs     uint32 `json:"cubexs"`
	Cubeys     uint32 `json:"cubeys"`
	Cubezs     uint32 `json:"cubezs"`
	Fragmentxs uint32 `json:"fragmentxs"`
	Fragmentys uint32 `json:"fragmentys"`
	Fragmentzs uint32 `json:"fragmentzs"`
}

func (s *manifestFileStore) Fetch(ctx context.Context, id string) (Manifest, error) {
	const op = events.Op("store.manifestFileStore.Fetch")
	m := s.localStore
	var mani Manifest
	fileName := path.Join(string(m.basePath), id)
	cont, err := ioutil.ReadFile(path.Clean(fileName))
	if err != nil {
		return mani, events.E(op, "Could not read manifest from local store", err, events.NotFound)
	}
	err = json.Unmarshal(cont, &mani)
	if err != nil {
		return mani, events.E(op, "Unmarshaling to Manifest", err, events.Marshalling)
	}
	return mani, nil
}

func (s *manifestBlobStore) Fetch(ctx context.Context, id string) (Manifest, error) {
	const op = events.Op("store.manifestFileStore.Fetch")
	az := s.blobStore
	var mani Manifest
	blobURL := az.containerURL.NewBlockBlobURL(id)
	downloadResponse, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return mani, events.E(op, "Manifest download ", err)
	}
	l.LogI(string(op), fmt.Sprintf("Download: manifestLength: %d bytes\n", downloadResponse.ContentLength()))
	retryReader := downloadResponse.Body(azblob.RetryReaderOptions{MaxRetryRequests: 3})
	buf := &bytes.Buffer{}
	_, err = buf.ReadFrom(retryReader)
	if err != nil {
		return mani, events.E(op, "Buffer read", err)
	}
	err = json.Unmarshal(buf.Bytes(), &mani)
	if err != nil {
		return mani, events.E(op, "Manifest unmarshal", err)
	}
	return mani, nil
}

func (s *manifestDbStore) Fetch(ctx context.Context, id string) (Manifest, error) {
	const op = events.Op("store.manifestDbStore.Fetch")
	m := s.dbStore
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var res Manifest
	client, err := mongo.Connect(dbCtx, options.Client().ApplyURI(string(m.connString)))
	if err != nil {
		return res, err
	}
	defer func() {
		err := client.Disconnect(dbCtx)
		if err != nil {
			l.LogE(string(op), "Disconnect manifest store", err)
		}
	}()
	collection := client.Database("seismiccloud").Collection("manifests")
	err = collection.FindOne(dbCtx, bson.D{bson.E{Key: "basename", Value: id}}).Decode(&res)
	if err != nil {
		return res, events.E(op, "Finding and unmarshaling to Manifest", err, events.Marshalling)
	}
	l.LogI(string(op), fmt.Sprintf("Connected to manifest DB and fetched file %s", id))
	return res, nil
}

func (s *manifestInMemoryStore) Fetch(ctx context.Context, id string) (Manifest, error) {
	const op = events.Op("store.manifestInMemoryStore.Fetch")
	s.inMemoryStore.lock.RLock()
	defer s.inMemoryStore.lock.RUnlock()
	var mani Manifest
	b, ok := s.inMemoryStore.m[id]
	if !ok {
		return mani, events.E(op, "No manifest for id", events.NotFound)
	}
	err := json.Unmarshal(b, &mani)
	if err != nil {
		return mani, events.E(op, "Unmarshaling to Manifest", err, events.Marshalling)
	}
	return mani, nil
}

func (s *manifestFileStore) List(ctx context.Context) ([]Manifest, error) {
	const op = events.Op("store.manifestFileStore.List")
	m := s.localStore
	var res []Manifest
	files, err := ioutil.ReadDir(string(m.basePath))
	if err != nil {
		return nil, events.E(op, "Invalid local file store", err, events.NotFound)
	}
	for _, file := range files {
		var mani Manifest
		b, _ := ioutil.ReadFile(path.Join(string(m.basePath), file.Name()))
		err = json.Unmarshal(b, &mani)
		if err == nil {
			res = append(res, mani)
		}
	}
	l.LogI(string(op), fmt.Sprintf("Fetched %d files from local store", len(res)))
	return res, nil
}

func (s *manifestBlobStore) List(ctx context.Context) ([]Manifest, error) {
	const op = events.Op("store.manifestBlobStore.List")
	az := s.blobStore
	var manis []Manifest
	for marker := (azblob.Marker{}); marker.NotDone(); {
		listBlob, err := az.containerURL.ListBlobsFlatSegment(ctx, marker, azblob.ListBlobsSegmentOptions{})
		if err != nil {
			return nil, events.E(op, "Could not list blobs", err)
		}
		marker = listBlob.NextMarker
		for _, blobInfo := range listBlob.Segment.BlobItems {
			mani, err := s.Fetch(ctx, blobInfo.Name)
			if err == nil {
				manis = append(manis, mani)
			}
		}
	}
	return manis, nil
}

func (s *manifestDbStore) List(ctx context.Context) ([]Manifest, error) {
	const op = events.Op("store.manifestDbStore.List")
	m := s.dbStore
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var res []Manifest
	client, err := mongo.Connect(dbCtx, options.Client().ApplyURI(string(m.connString)))
	if err != nil {
		return res, err
	}
	defer func() {
		err := client.Disconnect(dbCtx)
		if err != nil {
			l.LogE(string(op), "Disconnect manifest store", err)
		}
	}()
	collection := client.Database("seismiccloud").Collection("manifests")
	find, err := collection.Find(dbCtx, bson.D{})
	if err != nil {
		return res, events.E(op, "Find in DB error", err, events.NotFound)
	}

	for find.Next(dbCtx) {
		var mani Manifest
		err := find.Decode(&mani)
		if err == nil {
			res = append(res, mani)
		}
	}
	l.LogI(string(op), fmt.Sprintf("Connected to manifest DB and fetched %d files", len(res)))
	return res, nil
}

func (s *manifestInMemoryStore) List(ctx context.Context) ([]Manifest, error) {
	const op = events.Op("store.manifestInMemoryStore.List")
	s.inMemoryStore.lock.RLock()
	defer s.inMemoryStore.lock.RUnlock()
	var res []Manifest
	for _, v := range s.inMemoryStore.m {
		var mani Manifest
		err := json.Unmarshal(v, &mani)
		if err == nil {
			res = append(res, mani)
		}
	}
	l.LogI(string(op), fmt.Sprintf("Fetched %d files from memory store", len(res)))
	return res, nil
}

func (m Manifest) ToJSON() (string, error) {
	const op = events.Op("store.ToJSON")
	b, err := json.Marshal(m)
	if err != nil {
		return "", events.E(op, "Manifest to JSON", err, events.Marshalling)
	}
	return string(b), nil
}
