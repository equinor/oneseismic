package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"sync"
	"time"

	"github.com/equinor/seismic-cloud/api/events"
	l "github.com/equinor/seismic-cloud/api/logger"
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
		basePath string
	}

	manifestDbStore struct {
		connString string
	}

	manifestInMemoryStore struct {
		m    map[string]Manifest
		lock sync.RWMutex
	}
)

// ConnStr is the connectionstring a db would use
type ConnStr string

// NewManifestStore provides a manifest store based on the persistance configuration given
func NewManifestStore(persistance interface{}) (ManifestStore, error) {
	switch persistance.(type) {
	case map[string]Manifest:
		return &manifestInMemoryStore{m: persistance.(map[string]Manifest)}, nil
	case ConnStr:
		return &manifestDbStore{string(persistance.(ConnStr))}, nil
	case string:
		return &manifestFileStore{persistance.(string)}, nil
	default:
		return nil, events.E("No manifest store persistance selected")
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

func (m *manifestFileStore) Fetch(ctx context.Context, id string) (Manifest, error) {
	var mani Manifest
	fileName := path.Join(m.basePath, id)
	cont, err := ioutil.ReadFile(path.Clean(fileName))
	if err != nil {
		return mani, err
	}

	err = json.Unmarshal(cont, &mani)
	if err != nil {
		return mani, err
	}
	return mani, nil
}

func (m *manifestDbStore) Fetch(ctx context.Context, id string) (Manifest, error) {
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var res Manifest
	client, err := mongo.Connect(dbCtx, options.Client().ApplyURI(m.connString))
	if err != nil {
		return res, err
	}
	defer client.Disconnect(dbCtx)
	collection := client.Database("seismiccloud").Collection("manifests")
	err = collection.FindOne(dbCtx, bson.D{{"basename", id}}).Decode(&res)
	if err != nil {
		return res, err
	}
	l.LogI("manifest fetch", fmt.Sprintf("Connected to manifest DB and fetched file %s", id))
	return res, nil
}

func (inMem *manifestInMemoryStore) Fetch(ctx context.Context, id string) (Manifest, error) {
	inMem.lock.RLock()
	defer inMem.lock.RUnlock()
	manifest, ok := inMem.m[id]
	if !ok {
		return Manifest{}, events.E("No manifest for id")
	}
	return manifest, nil
}

func (m *manifestFileStore) List(ctx context.Context) ([]Manifest, error) {
	op := "manifestFileStore list"
	var res []Manifest
	files, err := ioutil.ReadDir(m.basePath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		var mani Manifest
		b, _ := ioutil.ReadFile(path.Join(m.basePath, file.Name()))
		err = json.Unmarshal(b, &mani)
		if err == nil {
			res = append(res, mani)
		}
	}
	l.LogI(op, fmt.Sprintf("Fetched %d files from local store", len(res)))
	return res, nil
}

func (m *manifestDbStore) List(ctx context.Context) ([]Manifest, error) {
	op := "manifestDbStore list"
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var res []Manifest
	client, err := mongo.Connect(dbCtx, options.Client().ApplyURI(m.connString))
	if err != nil {
		return res, err
	}
	defer client.Disconnect(dbCtx)
	collection := client.Database("seismiccloud").Collection("manifests")
	find, err := collection.Find(dbCtx, bson.D{})
	if err != nil {
		return res, err
	}

	for find.Next(dbCtx) {
		var mani Manifest
		err := find.Decode(&mani)
		if err == nil {
			res = append(res, mani)
		}
	}
	l.LogI(op, fmt.Sprintf("Connected to manifest DB and fetched %d files", len(res)))
	return res, nil

}

func (inMem *manifestInMemoryStore) List(ctx context.Context) ([]Manifest, error) {
	op := "manifestInMemoryStore list"
	inMem.lock.RLock()
	defer inMem.lock.RUnlock()
	var res []Manifest
	for _, v := range inMem.m {
		res = append(res, v)
	}
	l.LogI(op, fmt.Sprintf("Fetched %d files from memory store", len(res)))
	return res, nil
}

func (m Manifest) ToJSON() (string, error) {

	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
