package service

import (
	"net/url"
	"reflect"
	"testing"
)

func TestGetKey(t *testing.T) {
	type args struct {
		authserver *url.URL
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetKeySet(tt.args.authserver)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetKeySet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetKeySet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getJSON(t *testing.T) {
	type args struct {
		url    *url.URL
		target interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := getJSON(tt.args.url, tt.args.target); (err != nil) != tt.wantErr {
				t.Errorf("getJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
