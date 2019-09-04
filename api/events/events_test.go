package events

import (
	"fmt"
	"testing"
	"time"
)

func TestError(t *testing.T) {
	ev := E(Op("foo"), ErrorLevel, fmt.Errorf("bar"), "foobar")
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

	if ev.ContextId() != "" {
		return
	}

	if ev.Kind.String() != "" {
		return
	}
}
