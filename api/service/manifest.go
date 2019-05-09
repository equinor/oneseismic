package service

import (
	"io/ioutil"
	"path"
)

type Manifest struct {
	basePath string
}

func (m *Manifest) Fetch(id string) (string, error) {
	fileName := path.Join(m.basePath, id+".manifest")
	cont, err := ioutil.ReadFile(path.Clean(fileName))
	if err != nil {
		return "", err
	}
	return string(cont), nil
}
