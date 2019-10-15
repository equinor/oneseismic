package store

import (
	"reflect"
	"testing"
)

func TestToJSON(t *testing.T) {

	manifest := &Manifest{
		Basename:   "mock",
		Cubexs:     1,
		Cubeys:     1,
		Cubezs:     1,
		Fragmentxs: 1,
		Fragmentys: 1,
		Fragmentzs: 1,
	}

	tests := []struct {
		got  Manifest
		want string
	}{
		{*manifest,
			`{"basename":"mock","cubexs":1,"cubeys":1,"cubezs":1,"fragmentxs":1,"fragmentys":1,"fragmentzs":1}`},
	}

	for _, tt := range tests {
		got, err := tt.got.ToJSON()

		if err != nil {
			t.Errorf("ManifestStore.ToJSON")
			return
		}

		if !reflect.DeepEqual(tt.want, got) {
			t.Errorf("ManifestStore.ToJSON : Got = %v, want %v", tt.want, got)
			return
		}
	}

}
