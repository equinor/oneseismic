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
	case *sql.DB:
		return &dbLogger{pool: sink.(*sql.DB), verbosity: verbosity}, nil
	default:
		return nil, errors.E(errors.Op("logger.factory"), fmt.Errorf("no logger defined for sink"))
	}

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

type dbLogger struct {
	pool      *sql.DB
	verbosity errors.Severity
}

func (dl *dbLogger) Log(err errors.Error) {
	if err.Level < dl.verbosity {
		return
	}

	_, wErr := dl.pool.Exec(
		"INSERT INTO log (log) VALUES ($1)",
		err.Error(),
	)
	if wErr != nil {
		panic(fmt.Errorf("Error logging to db: %v, %v", wErr, err))
	}
}
