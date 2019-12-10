package store

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestManifestSerializable(t *testing.T) {

	manifest := &Manifest{
		Basename:   "mock",
		Cubexs:     1,
		Cubeys:     1,
		Cubezs:     1,
		Fragmentxs: 1,
		Fragmentys: 1,
		Fragmentzs: 1,
	}

	jsonManifest, err := json.Marshal(manifest)

	if err != nil {
		t.Errorf("json marshal manifest: %v", err)
		return
	}
	mani := new(Manifest)
	err = json.Unmarshal(jsonManifest, mani)

	if err != nil {
		t.Errorf("json unmarshal manifest: %v", err)
		return
	}

	if !reflect.DeepEqual(manifest, mani) {
		t.Errorf("ManifestStore.ToJSON : Got = %v, want %v", manifest, mani)
		return
	}

}
