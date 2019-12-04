package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getCaller(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"caller is test", "runtime"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCaller(func(s string) bool { return false })
			assert.Contains(t, got, tt.want)
		})
	}
}

func Test_isInternal(t *testing.T) {
	type args struct {
		pkg string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Not internal", args{"cmd"}, false},
		{"Is internal", args{"runtime"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isInternal(tt.args.pkg); got != tt.want {
				t.Errorf("isInternal() = %v, want %v", got, tt.want)
			}
		})
	}
}
