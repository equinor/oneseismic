package logger

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
	"github.com/equinor/seismic-cloud/api/events"
)

type logger interface {
	log(*events.Event)
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
func SetLogSink(sink interface{}, verbosity events.Severity) error {
	switch sink.(type) {
	case *os.File:
		innerLogger = &fileLogger{file: sink.(*os.File), verbosity: verbosity}
	case *sql.DB:
		innerLogger = &dbLogger{pool: sink.(*sql.DB), verbosity: verbosity}
	default:
		return events.E(events.Op("logger.factory"), fmt.Errorf("no logger defined for sink"))
	}
	return nil
}

func errToLog(ev *events.Event) string {
	return fmt.Sprintln(ev.When.Format(time.RFC3339), ev.Level, ev.Error(), ev.Message)
}

// adds log source to logger
func AddLoggerSource(loggerName string, output func(io.Writer)) {
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
			sev := events.ParseLevel(level)

			nErr := events.E(events.Op(loggerName), sev, fmt.Errorf("%s", s)).(*events.Event)
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

func parseIrisSeverity(s string) events.Severity {
	if strings.Contains(s, "[DBUG]") {
		return events.DebugLevel
	} else if strings.Contains(s, "[INFO]") {
		return events.InfoLevel
	} else if strings.Contains(s, "[WARN]") {
		return events.WarnLevel
	} else if strings.Contains(s, "[ERRO]") {
		return events.ErrorLevel
	} else if strings.Contains(s, "[FTAL]") {
		return events.CriticalLevel
	} else {
		return events.InfoLevel
	}
}

func AddGoLogSource(output func(io.Writer) *golog.Logger) {
	pr, pw := io.Pipe()
	br := bufio.NewReader(pr)

	go func() {
		for {

			s, err := br.ReadString('\n')
			if err != nil {
				panic(err)
			}
			sev := parseIrisSeverity(s)
			s = strings.TrimRight(s, "\n")
			if strings.Contains(s, "iris:") {
				// Remove severity and timestamp
				s = s[29:]
			}
			nErr := events.E(events.Op("iris.log"), sev, fmt.Errorf("%s", s)).(*events.Event)

			Log(nErr)
		}
	}()

	output(pw)

}

type fileLogger struct {
	file      io.Writer
	verbosity events.Severity
}

func (fl *fileLogger) log(err *events.Event) {
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
	verbosity events.Severity
}

func (dl *dbLogger) log(ev *events.Event) {
	if ev.Level < dl.verbosity {
		return
	}

	_, wErr := dl.pool.Query(
		"INSERT INTO log (time, operation, error, message, severity, userid, ctxid, eventkind) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
		ev.When,
		ev.Op,
		ev.Error(),
		ev.Message,
		ev.Level.String(),
		ev.UserID,
		ev.ContextId(),
		ev.Kind.String(),
	)
	if wErr != nil {
		panic(fmt.Errorf("Error logging to db: %v, %v", wErr, ev))
	}
}

func Kind(kind events.EventKind) LogEventOption {
	return newFuncOption(func(ev *events.Event) {
		ev.Kind = kind
		return
	})
}

func Wrap(err error) LogEventOption {
	return newFuncOption(func(ev *events.Event) {
		ev.Err = err
		return
	})
}

// LogD Debug
func LogD(op, msg string, opts ...LogEventOption) {
	e := events.E(events.Op(op), events.DebugLevel, msg).(*events.Event)
	for _, opt := range opts {
		opt.apply(e)
	}
	innerLogger.log(e)
}

// LogI Info
func LogI(op, msg string, opts ...LogEventOption) {
	e := events.E(events.Op(op), events.InfoLevel, msg).(*events.Event)
	for _, opt := range opts {
		opt.apply(e)
	}
	innerLogger.log(e)
}

// LogW Warning
func LogW(op, msg string, opts ...LogEventOption) {
	e := events.E(events.Op(op), events.WarnLevel, msg).(*events.Event)
	for _, opt := range opts {
		opt.apply(e)
	}
	innerLogger.log(e)
}

// LogE Error
func LogE(op, msg string, err error, opts ...LogEventOption) {
	e := events.E(events.Op(op), events.ErrorLevel, err, msg).(*events.Event)
	for _, opt := range opts {
		opt.apply(e)
	}
	innerLogger.log(e)
}

// LogC Critical
func LogC(op, msg string, err error, opts ...LogEventOption) {
	e := events.E(events.Op(op), events.CriticalLevel, err, msg).(*events.Event)
	for _, opt := range opts {
		opt.apply(e)
	}
	innerLogger.log(e)
}

// Log sends error to chosen sink
func Log(err *events.Event) {
	innerLogger.log(err)
}
