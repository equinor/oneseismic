package logger

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/equinor/oneseismic/api/events"
	"github.com/stretchr/testify/assert"
)

func TestLogSourceAndSink(t *testing.T) {
	err := SetLogSink("invalid log sink", events.DebugLevel)
	assert.Contains(t, err.Error(), "no logger defined for sink")

	AddLoggerSource("test.log", log.SetOutput)
	r, w, _ := os.Pipe()
	err = SetLogSink(w, events.NoLevel)
	assert.NoError(t, err, "Setting log sink failed")
	log.Print("dummy__using builtin logger")
	time.Sleep(1000 * time.Millisecond)
	w.Close()
	out, err := ioutil.ReadAll(r)
	assert.NoError(t, err, "Reading from log sink failed")
	expect := "[UNKN] test.log: using builtin logger"
	assert.Contains(t, string(out), expect)
}

func TestErrToLog(t *testing.T) {
	tt, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
	str := errToLog(&events.Event{
		When: tt,
		Op:   events.Op("TestErrToLog.errToLog"),
		Err:  errors.New("Testing")})
	expected := "2006-01-02T15:04:05Z [UNKN] TestErrToLog.errToLog: Testing\n"
	assert.Equal(t, expected, str, fmt.Sprintf("errToLog failed, expected %v, got %v", expected, str))
}

func TestParseIrisSeverity(t *testing.T) {
	tests := []struct {
		name         string
		irisSeverity string
		want         events.Severity
	}{
		{"Debug", "[DBUG]", events.DebugLevel},
		{"Info", "[INFO]", events.InfoLevel},
		{"Warning", "[WARN]", events.WarnLevel},
		{"Error", "[ERRO]", events.ErrorLevel},
		{"Critical", "[FTAL]", events.CriticalLevel},
		{"Default", "[WE'RE ALL GONNA DIE]", events.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sev := parseIrisSeverity(tt.irisSeverity)
			assert.Equal(t, tt.want, sev)
		})
	}
}

func TestLog(t *testing.T) {
	tests := []struct {
		name     string
		f        interface{}
		expects  []string
		callArgs []interface{}
	}{
		{"Debug",
			LogD,
			[]string{"DEBG", "TestingD", "errorD"},
			[]interface{}{"TestingD", Wrap(errors.New("errorD"))}},
		{"Info",
			LogI,
			[]string{"INFO", "TestingI", "not found"},
			[]interface{}{"TestingI", Kind(events.NotFound)}},
		{"Warning",
			LogW,
			[]string{"WARN", "TestingW"},
			[]interface{}{"TestingW", EmptyOption{}}},
		{"Error",
			LogE,
			[]string{"ERRO", "TestingE", "errorE", "not found"},
			[]interface{}{"TestingE", errors.New("errorE"), Kind(events.NotFound)}},
		{"Critical",
			LogC,
			[]string{"CRIT", "TestingC", "errorC", "not found"},
			[]interface{}{
				"TestingC",
				errors.New("errorC"),
				Kind(events.NotFound),
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, w, _ := os.Pipe()
			err := SetLogSink(w, events.DebugLevel)
			assert.NoError(t, err, "Setting log sink failed")
			v := reflect.ValueOf(tt.f)
			ty := v.Type()
			argv := make([]reflect.Value, ty.NumIn())
			for idx, arg := range tt.callArgs {
				argv[idx] = reflect.ValueOf(arg)
			}
			_ = v.Call(argv)
			time.Sleep(100 * time.Millisecond)
			w.Close()

			out, err := ioutil.ReadAll(reader)
			assert.NoError(t, err, "Reading from log sink failed")
			for _, expect := range tt.expects {
				assert.Contains(t, string(out), expect)
			}

		})
	}
}
