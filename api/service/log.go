package service

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kataras/golog"
	_ "github.com/lib/pq"

	"github.com/equinor/seismic-cloud/api/config"
	"github.com/equinor/seismic-cloud/api/errors"
)

type logger interface {
	log(*errors.Event)
}

var innerLogger logger

func DbOpener() (*sql.DB, error) {
	db, err := sql.Open("postgres", config.LogDBConnStr())
	if err != nil {
		return nil, err
	}
	rows, err := db.Query("SELECT * FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = 'log'")
	if err != nil || rows == nil {
		return nil, err
	}
	defer rows.Close()
	return db, nil
}

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

func errToLog(err *errors.Event) string {
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
				fmt.Sscanf(s, "%s %d/%d/%d %d:%d:%d", &level, &year, &month, &day, &hour, &minute, &second)
				s = s[27:]
			}
			sev := errors.ParseLevel(level)

			nErr := errors.E(errors.Op(loggerName), sev, fmt.Errorf("%s", s))
			logtime := time.Date(
				year,
				time.Month(month),
				day, hour, minute, second, 0,
				time.FixedZone(time.Now().Zone()))
			nErr.When = logtime.UTC()

			Log(nErr)
		}
	}()

	output(pw)
}

func WrapIrisLogger(output func(io.Writer) *golog.Logger) {
	pr, pw := io.Pipe()
	br := bufio.NewReader(pr)

	go func() {
		for {

			s, err := br.ReadString('\n')
			if err != nil {
				panic(err)
			}
			s = strings.TrimRight(s, "\n")
			sev := errors.InfoLevel

			nErr := errors.E(errors.Op("iris.log"), sev, fmt.Errorf("%s", s))
			nErr.When = time.Now().UTC()

			Log(nErr)
		}
	}()

	output(pw)

}

type fileLogger struct {
	file      io.Writer
	verbosity errors.Severity
}

func (fl *fileLogger) log(err *errors.Event) {
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

func (dl *dbLogger) log(err *errors.Event) {
	if err.Level < dl.verbosity {
		return
	}

	_, wErr := dl.pool.Query(
		"INSERT INTO log (time, error, eventkind, severity, message) VALUES ($1,$2,$3,$4,$5)",
		err.When,
		err.Error(),
		err.Kind.String(),
		err.Level.String(),
		err.Message,
	)
	if wErr != nil {
		panic(fmt.Errorf("Error logging to db: %v, %v", wErr, err))
	}
}

// Log sends error to chosen sink
func Log(err *errors.Event) {
	innerLogger.log(err)
}
