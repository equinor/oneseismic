package service

import (
	"io/ioutil"
	"path"
)

type ManifestFileStore struct {
	BasePath string
}

type ManifestStore interface {
	Fetch(string) ([]byte, error)
}

func (m *ManifestFileStore) Fetch(id string) ([]byte, error) {
	fileName := path.Join(m.BasePath, id+".manifest")
	cont, err := ioutil.ReadFile(path.Clean(fileName))
	if err != nil {
		return nil, err
	}
	return cont, nil
}
