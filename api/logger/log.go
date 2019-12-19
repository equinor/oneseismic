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
	"github.com/equinor/seismic-cloud-api/api/events"
	"github.com/kataras/golog"
	pq "github.com/lib/pq"
)

type logger interface {
	log(*events.Event) error
	isFlushed() bool
}

var innerLogger logger = &fileLogger{file: os.Stdout, verbosity: events.DebugLevel, wg: &sync.WaitGroup{}}
var logMut = &sync.Mutex{}

var Version string

type ConnString string

func pingDb(connStr string) error {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	defer db.Close()
	rows, err := db.Query("SELECT * FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = 'log'")
	if err != nil || rows == nil {
		return err
	}
	defer rows.Close()
	return nil
}

// Create logger to sink
func SetLogSink(sink interface{}, verbosity events.Severity) error {
	switch sink := sink.(type) {
	case *os.File:
		innerLogger = &fileLogger{file: sink, verbosity: verbosity, wg: &sync.WaitGroup{}}
	case ConnString:
		err := pingDb(string(sink))
		if err != nil {
			return events.E("Pinging Db", events.Op("logger.SetLogSink"), err)
		}
		innerLogger = &dbLogger{connStr: string(sink), verbosity: verbosity, eventBuffer: queue.NewFIFO()}
	default:
		return events.E("no logger defined for sink", events.Op("logger.factory"), events.CriticalLevel)
	}
	return nil
}

func errToLog(ev *events.Event) string {
	return fmt.Sprintf("%s [%s] %s\n",
		ev.When.Format(time.RFC3339),
		ev.Level,
		ev.Error())

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

			nErr := events.E(s, events.Op(loggerName), sev).(*events.Event)
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
			nErr := events.E(s, events.Op("iris.log"), sev).(*events.Event)

			Log(nErr)
		}
	}()

	output(pw)

}

type fileLogger struct {
	file      *os.File
	verbosity events.Severity
	wg        *sync.WaitGroup
}

func (fl *fileLogger) log(ev *events.Event) error {
	fl.wg.Add(1)
	defer fl.wg.Done()
	if ev.Level < fl.verbosity {
		return nil
	}
	s := errToLog(ev)
	_, wErr := fl.file.Write([]byte(s))
	if wErr != nil {
		return wErr
	}
	_ = fl.file.Sync()
	return nil
}

func (fl *fileLogger) isFlushed() bool {
	time.Sleep(100 * time.Millisecond)
	fl.wg.Wait()
	return true
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
							"Writing event to db sink",
							events.Op("log.bulkInsert"),
							err,
							events.ErrorLevel))
				}

			}
		}()

	})

	err := dl.eventBuffer.Enqueue(ev)
	return err
}

func (dl *dbLogger) isFlushed() bool {
	return true
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
			fmt.Println(events.E("Writing event to sink", op, err, events.ErrorLevel))
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
func LogD(msg string, opts ...LogEventOption) {
	e := events.E(msg, events.DebugLevel).(*events.Event)
	for _, opt := range opts {
		opt.apply(e)
	}
	logToSink(e)
}

// LogI Info
func LogI(msg string, opts ...LogEventOption) {
	e := events.E(msg, events.InfoLevel).(*events.Event)
	for _, opt := range opts {
		opt.apply(e)
	}
	logToSink(e)
}

// LogW Warning
func LogW(msg string, opts ...LogEventOption) {
	e := events.E(msg, events.WarnLevel).(*events.Event)
	for _, opt := range opts {
		opt.apply(e)
	}
	logToSink(e)
}

// LogE Error
func LogE(msg string, err error, opts ...LogEventOption) {
	e := events.E(msg, events.ErrorLevel, err).(*events.Event)
	for _, opt := range opts {
		opt.apply(e)
	}
	logToSink(e)
}

// LogC Critical
func LogC(msg string, err error, opts ...LogEventOption) {
	e := events.E(msg, events.CriticalLevel, err).(*events.Event)
	for _, opt := range opts {
		opt.apply(e)
	}
	logToSink(e)
}

// Log sends error to chosen sink
func Log(err *events.Event) {
	logToSink(err)
}

func Wait() {

	for !innerLogger.isFlushed() {
		time.Sleep(10 * time.Millisecond)
	}
	fmt.Println("Logger flushed")
}
