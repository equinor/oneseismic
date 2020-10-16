package util

import (
	"github.com/google/uuid"
)

func MakePID() string {
	return uuid.New().String()
}
