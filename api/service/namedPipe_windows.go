package service

import (
	"fmt"
	"os"
)

func NewNamedPipe() (*os.File, *os.File, error) {

	return nil, nil, fmt.Errorf("No support for named pipe as file on windows")
}
