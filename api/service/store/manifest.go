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
	Fetch(string) (Manifest, error)
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
	Cubexs     uint32 `json:"cube-xs"`
	Cubeys     uint32 `json:"cube-ys"`
	Cubezs     uint32 `json:"cube-zs"`
	Fragmentxs uint32 `json:"fragment-xs"`
	Fragmentys uint32 `json:"fragment-ys"`
	Fragmentzs uint32 `json:"fragment-zs"`
}

func (m *manifestFileStore) Fetch(id string) (Manifest, error) {
	var mani Manifest
	fileName := path.Join(m.basePath, id+".manifest")
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

func (m *manifestDbStore) Fetch(id string) (Manifest, error) {
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

func (inMem *manifestInMemoryStore) Fetch(id string) (Manifest, error) {
	inMem.lock.RLock()
	defer inMem.lock.RUnlock()
	manifest, ok := inMem.m[id]
	if !ok {
		return Manifest{}, events.E("No manifest for id")
	}
	return manifest, nil
}
