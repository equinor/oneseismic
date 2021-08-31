package datastorage

import (
	"log"

	"github.com/equinor/oneseismic/api/api"
)

/*
* The factory-approach facilitates different backends but also greatly
* simplifies integration-testing. Performance-wise it makes no difference.
*
* Integration-test can set Storage and Create() will just return it
 */

// TODO: This is also a singleton - figure out if this is necessary/desired
var Storage api.AbstractStorage = nil
func CreateStorage(StorageKind string, url string) api.AbstractStorage {
	if Storage == nil {
		switch StorageKind {
		case "azure":
			Storage = NewAzureStorage(url)
		case "file":
			Storage = NewFileStorage(url)
		default:
			log.Fatalf("Unknown storage type %s", StorageKind)
		}
	}
	log.Printf("Storage type=%T", Storage)
	return Storage
}
