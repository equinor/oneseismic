package service

import (
	"fmt"
	"os"
	"syscall"

	"github.com/google/uuid"
)

func NewNamedPipe() (pr *os.File, pw *os.File, err error) {

	name := fmt.Sprintf("/tmp/sc-api-%s", uuid.New())

	//to create pipe: does not work in windows
	err = syscall.Mkfifo(name, 0666)
	if err != nil {
		return
	}

	pw, err = os.OpenFile(name, os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		return nil, nil, err
	}

	pr, err = os.OpenFile(name, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		pw.Close()
		return nil, nil, err
	}
	return
}
