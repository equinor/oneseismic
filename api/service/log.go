package service

import (
	"database/sql"
	"fmt"
	"io"
	"os"

	"github.com/equinor/seismic-cloud/api/errors"
)

type Logger interface {
	Log(errors.Error)
}

// Create logger to sink
func CreateLogger(sink interface{}, verbosity errors.Severity) (Logger, error) {
	switch sink.(type) {
	case *os.File:
		return &fileLogger{file: sink.(*os.File), verbosity: verbosity}, nil

	}
	return nil, errors.E(errors.Op("logger.factory"), fmt.Errorf("no logger defined for sink"))
}

type fileLogger struct {
	file      io.Writer
	verbosity errors.Severity
}

func (fl *fileLogger) Log(err errors.Error) {
	if err.Level < fl.verbosity {
		return
	}
	_, wErr := fl.file.Write([]byte(err.Error()))
	if wErr != nil {
		panic(fmt.Errorf("Error logging to file: %v, %v", wErr, err))
	}
}

type sqlLogger struct {
	Pool *sql.DB
}
