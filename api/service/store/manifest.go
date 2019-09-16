package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"time"

	l "github.com/equinor/seismic-cloud/api/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ManifestFileStore struct {
	BasePath string
}

type ManifestDbStore struct {
	ConnString string
}

type ManifestStore interface {
	Fetch(string) ([]byte, error)
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

func (m *ManifestFileStore) Fetch(id string) ([]byte, error) {
	fileName := path.Join(m.BasePath, id+".manifest")
	cont, err := ioutil.ReadFile(path.Clean(fileName))
	if err != nil {
		return nil, err
	}
	return cont, nil
}

func (m *ManifestDbStore) Fetch(id string) ([]byte, error) {
	db_ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(db_ctx, options.Client().ApplyURI(m.ConnString))
	if err != nil {
		return []byte{}, err
	}
	defer client.Disconnect(db_ctx)
	collection := client.Database("seismiccloud").Collection("manifests")
	var res Manifest
	err = collection.FindOne(db_ctx, bson.D{{"basename", id}}).Decode(&res)
	if err != nil {
		return nil, err
	}
	l.LogI("manifest fetch", fmt.Sprintf("Connected to manifest DB and fetched file %s", id))
	resBytes := new(bytes.Buffer)
	json.NewEncoder(resBytes).Encode(res)
	return resBytes.Bytes(), nil
}
