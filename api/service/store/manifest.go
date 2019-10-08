package store

import (
	"bytes"
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
	Fetch(string) ([]byte, error)
}

type (
	manifestFileStore struct {
		basePath string
	}

	manifestDbStore struct {
		connString string
	}

	manifestInMemoryStore struct {
		m    map[string][]byte
		lock sync.RWMutex
	}
)

// ConnStr is the connectionstring a db would use
type ConnStr string

// NewManifestStore provides a manifest store based on the persistance configuration given
func NewManifestStore(persistance interface{}) (ManifestStore, error) {
	switch persistance.(type) {
	case map[string][]byte:
		return &manifestInMemoryStore{m: persistance.(map[string][]byte)}, nil
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
	Cubexs     int32  `json:"cube-xs"`
	Cubeys     int32  `json:"cube-ys"`
	Cubezs     int32  `json:"cube-zs"`
	Fragmentxs int32  `json:"fragment-xs"`
	Fragmentys int32  `json:"fragment-ys"`
	Fragmentzs int32  `json:"fragment-zs"`
}

func (m *manifestFileStore) Fetch(id string) ([]byte, error) {
	fileName := path.Join(m.basePath, id+".manifest")
	cont, err := ioutil.ReadFile(path.Clean(fileName))
	if err != nil {
		return nil, err
	}
	return cont, nil
}

func (m *manifestDbStore) Fetch(id string) ([]byte, error) {
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(dbCtx, options.Client().ApplyURI(m.connString))
	if err != nil {
		return []byte{}, err
	}
	defer client.Disconnect(dbCtx)
	collection := client.Database("seismiccloud").Collection("manifests")
	var res Manifest
	err = collection.FindOne(dbCtx, bson.D{{"basename", id}}).Decode(&res)
	if err != nil {
		return nil, err
	}
	l.LogI("manifest fetch", fmt.Sprintf("Connected to manifest DB and fetched file %s", id))
	resBytes := new(bytes.Buffer)
	json.NewEncoder(resBytes).Encode(res)
	return resBytes.Bytes(), nil
}

func (inMem *manifestInMemoryStore) Fetch(id string) ([]byte, error) {
	inMem.lock.RLock()
	defer inMem.lock.RUnlock()
	manifest, ok := inMem.m[id]
	if !ok {
		return nil, events.E("No manifest for id")
	}
	return manifest, nil
}
