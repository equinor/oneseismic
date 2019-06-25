package service

import (
	"os"
	"syscall"
)

type NamedPipeProvider struct {
	BasePath string
}

func (npp *NamedPipeProvider) New(uniqueName string) (*os.File, *os.File, error) {

	name := "/tmp/" + uniqueName

	if _, err := os.Stat(name); os.IsNotExist(err) {
		// path/to/whatever does not exist
	} else {
		os.Remove(name)
	}
	//to create pipe: does not work in windows
	err := syscall.Mkfifo(name, 0666)
	if err != nil {
		return nil, nil, err
	}
	// to open pipe to write
	pw, err := os.OpenFile(name, os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		return nil, nil, err
	}
	//to open pipe to read
	pr, err := os.OpenFile(name, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		pw.Close()
		return nil, nil, err
	}
	return pr, pw, nil
}
