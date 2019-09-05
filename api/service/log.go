package service

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/equinor/seismic-cloud/api/errors"
)

type logger interface {
	log(*errors.Error)
}

var innerLogger logger

// Create logger to sink
func SetLogSink(sink interface{}, verbosity errors.Severity) error {
	switch sink.(type) {
	case *os.File:
		innerLogger = &fileLogger{file: sink.(*os.File), verbosity: verbosity}
	case *sql.DB:
		innerLogger = &dbLogger{pool: sink.(*sql.DB), verbosity: verbosity}
	default:
		return errors.E(errors.Op("logger.factory"), fmt.Errorf("no logger defined for sink"))
	}
	return nil
}

func errToLog(err *errors.Error) string {
	return fmt.Sprintln(err.When.Format(time.RFC3339), err.Level, err.Error())
}

// adds log source to logger
func WrapLogger(loggerName string, output func(io.Writer)) {
	pr, pw := io.Pipe()
	br := bufio.NewReader(pr)

	go func() {
		for {

			s, err := br.ReadString('\n')
			if err != nil {
				panic(err)
			}
			s = strings.TrimRight(s, "\n")
			var level string
			var year, month, day, hour, minute, second int
			if loggerName == "setup.log" {
				fmt.Sscanf(s, "%s %d/%d/%d %d:%d:%d", &level, &year, &month, &day, &hour, &minute, &second)
				s = s[len(level)+21:]
			} else {
				level = "INFO"
				fmt.Sscanf(s, "%d/%d/%d %d:%d:%d", &year, &month, &day, &hour, &minute, &second)
				s = s[20:]
			}
			sev := errors.ParseLevel(level)

			nErr := errors.E(errors.Op(loggerName), sev, fmt.Errorf("%s", s))
			nErr.When = time.Date(
				year,
				time.Month(month),
				day, hour, minute, second, 0, time.UTC)

			Log(nErr)
		}
	}()

	output(pw)
}

type fileLogger struct {
	file      io.Writer
	verbosity errors.Severity
}

func (fl *fileLogger) log(err *errors.Error) {
	if err.Level < fl.verbosity {
		return
	}

	_, wErr := fl.file.Write([]byte(errToLog(err)))
	if wErr != nil {
		panic(fmt.Errorf("Error logging to file: %v, %v", wErr, err))
	}
}

type dbLogger struct {
	pool      *sql.DB
	verbosity errors.Severity
}

func (dl *dbLogger) log(err *errors.Error) {
	if err.Level < dl.verbosity {
		return
	}

	_, wErr := dl.pool.Exec(
		"INSERT INTO log (date,log) VALUES ($1,$2)",
		err.When,
		err.Error(),
	)
	if wErr != nil {
		panic(fmt.Errorf("Error logging to db: %v, %v", wErr, err))
	}
}

// Log sends error to chosen sink
func Log(err *errors.Error) {
	innerLogger.log(err)
}
