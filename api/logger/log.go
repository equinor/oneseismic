package logger

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	queue "github.com/enriquebris/goconcurrentqueue"
	"github.com/equinor/seismic-cloud/api/events"
	"github.com/kataras/golog"
	pq "github.com/lib/pq"
)

type logger interface {
	log(*events.Event) error
}

var innerLogger logger = &fileLogger{file: os.Stdout, verbosity: events.DebugLevel}
var logMut = &sync.Mutex{}

type ConnString string

func DbOpener(connStr string) (ConnString, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return "", err
	}
	rows, err := db.Query("SELECT * FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = 'log'")
	if err != nil || rows == nil {
		return "", err
	}
	defer rows.Close()
	return ConnString(connStr), nil
}

// Create logger to sink
func SetLogSink(sink interface{}, verbosity events.Severity) error {
	switch sink := sink.(type) {
	case *os.File:
		innerLogger = &fileLogger{file: sink, verbosity: verbosity}
	case ConnString:
		innerLogger = &dbLogger{connStr: string(sink), verbosity: verbosity, eventBuffer: queue.NewFIFO()}
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
	file      *os.File
	verbosity events.Severity
}

func (fl *fileLogger) log(ev *events.Event) error {
	if ev.Level < fl.verbosity {
		return nil
	}

	_, wErr := fl.file.Write([]byte(errToLog(ev)))
	if wErr != nil {
		return wErr
	}
	_ = fl.file.Sync()
	return nil
}

type dbLogger struct {
	connStr     string
	verbosity   events.Severity
	eventBuffer *queue.FIFO
}

var doOnce sync.Once

func (dl *dbLogger) log(ev *events.Event) error {
	if ev.Level < dl.verbosity {
		return nil
	}
	doOnce.Do(func() {

		go func() {
			ticker := time.NewTicker(2000 * time.Millisecond)
			for range ticker.C {

				err := dl.bulkInsert()
				if err != nil {
					fmt.Println(
						events.E(
							events.Op("log.bulkInsert"),
							"Writing event to db sink",
							err,
							events.ErrorLevel))
				}

			}
		}()

	})

	err := dl.eventBuffer.Enqueue(ev)
	return err
}

func (dl *dbLogger) bulkInsert() error {

	evs := make([]*events.Event, 0)
	for {
		it, qErr := dl.eventBuffer.Dequeue()
		if qErr != nil {
			break
		}
		evs = append(evs, it.(*events.Event))
	}
	if len(evs) == 0 {
		return nil
	}

	db, err := sql.Open("postgres", dl.connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	txn, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := txn.Prepare(
		pq.CopyIn("log",
			"time",
			"operation",
			"error",
			"message",
			"severity",
			"userid",
			"ctxid",
			"eventkind"))
	if err != nil {
		return err
	}

	for _, ev := range evs {
		_, err = stmt.Exec(
			ev.When,
			ev.Op,
			ev.Error(),
			ev.Message,
			ev.Level.String(),
			ev.UserID,
			ev.ContextID(),
			ev.Kind.String(),
		)
		if err != nil {
			return err
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	err = stmt.Close()
	if err != nil {
		return err
	}

	err = txn.Commit()
	if err != nil {
		return err
	}
	return nil
}

func logToSink(ev *events.Event) {
	op := events.Op("logging.log")
	go func(ev *events.Event) {
		logMut.Lock()
		err := innerLogger.log(ev)
		if err != nil {
			fmt.Println(events.E(op, "Writing event to sink", err, events.ErrorLevel))
		}
		logMut.Unlock()
	}(ev)
}

func Kind(kind events.EventKind) LogEventOption {
	return newFuncOption(func(ev *events.Event) {
		ev.Kind = kind
	})
}

func Wrap(err error) LogEventOption {
	return newFuncOption(func(ev *events.Event) {
		ev.Err = err
	})
}

// LogD Debug
func LogD(op, msg string, opts ...LogEventOption) {
	e := events.E(events.Op(op), events.DebugLevel, msg).(*events.Event)
	for _, opt := range opts {
		opt.apply(e)
	}
	logToSink(e)
}

// LogI Info
func LogI(op, msg string, opts ...LogEventOption) {
	e := events.E(events.Op(op), events.InfoLevel, msg).(*events.Event)
	for _, opt := range opts {
		opt.apply(e)
	}
	logToSink(e)
}

// LogW Warning
func LogW(op, msg string, opts ...LogEventOption) {
	e := events.E(events.Op(op), events.WarnLevel, msg).(*events.Event)
	for _, opt := range opts {
		opt.apply(e)
	}
	logToSink(e)
}

// LogE Error
func LogE(op, msg string, err error, opts ...LogEventOption) {
	e := events.E(events.Op(op), events.ErrorLevel, err, msg).(*events.Event)
	for _, opt := range opts {
		opt.apply(e)
	}
	logToSink(e)
}

// LogC Critical
func LogC(op, msg string, err error, opts ...LogEventOption) {
	e := events.E(events.Op(op), events.CriticalLevel, err, msg).(*events.Event)
	for _, opt := range opts {
		opt.apply(e)
	}
	logToSink(e)
}

// Log sends error to chosen sink
func Log(err *events.Event) {
	logToSink(err)
}
