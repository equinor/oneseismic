package events

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestError(t *testing.T) {
	err := E(Op("foo"), ErrorLevel, fmt.Errorf("bar"), "foobar", NotFound, UserID("ANON"), uuid.New())

	var ev *Event
	if !errors.As(err, &ev) {
		t.Errorf("err (%v) is not an error", err)
		return
	}

	if _, ok := interface{}(ev.When).(time.Time); !ok {
		return
	}

	if ev.Op != "foo" {
		return
	}

	if _, ok := ev.Err.(error); !ok {
		return
	}

	if ev.Error() != "bar" {
		return
	}

	if ev.Message != "foobar" {
		return
	}

	if ev.UserID != "" {
		return
	}

	if ev.ContextID() != "" {
		return
	}

	if ev.Kind.String() != "" {
		return
	}
}

func TestParseLevel(t *testing.T) {

	tests := []struct {
		name string
		have string
		want Severity
	}{
		{"Debug", "Debug", DebugLevel},
		{"Trace", "Trace", DebugLevel},
		{"Info", "Info", InfoLevel},
		{"Log", "Log", InfoLevel},
		{"Warn-1", "Warning", WarnLevel},
		{"Warn-2", "WARN", WarnLevel},
		{"Error", "Err", ErrorLevel},
		{"Critical", "CRIT", CriticalLevel},
		{"Critical", "[CRIT]", CriticalLevel},
		{"Fatal", "Fatal", CriticalLevel},
		{"Default", "asdf", DebugLevel},
		{"No string", "", DebugLevel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseLevel(tt.have); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s:ParseLevel() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestEventKind_String(t *testing.T) {
	tests := []struct {
		name string
		ek   EventKind
		want string
	}{
		{"unknown", UnknownError, "unknown error"},
		{"not found", NotFound, "not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ek.String(); got != tt.want {
				t.Errorf("EventKind.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		name string
		s    Severity
		want string
	}{
		{"Debug", DebugLevel, "DEBG"},
		{"Info", InfoLevel, "INFO"},
		{"Warning", WarnLevel, "WARN"},
		{"Error", ErrorLevel, "ERRO"},
		{"Critical", CriticalLevel, "CRIT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("Severity.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvent_Error(t *testing.T) {
	tests := []struct {
		name string
		e    *Event
		want string
	}{
		{"Single error",
			E(Op("foo"),
				ErrorLevel,
				fmt.Errorf("bar"),
				"foobar",
				NotFound,
				UserID("ANON"),
				uuid.New()).(*Event),
			"foo: foobar: not found: bar"},
		{"Double error",
			E(Op("foo"), ErrorLevel,
				E(Op("bar"), ErrorLevel,
					fmt.Errorf("baz"),
					"barbaz",
					NotFound,
					UserID("ANON"),
					uuid.New()),
				"foobar",
				NotFound,
				UserID("ANON"),
				uuid.New()).(*Event),
			"foo: foobar: not found-> bar: barbaz: not found: baz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.Error(); got != tt.want {
				t.Errorf("Event.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}
