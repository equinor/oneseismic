package events

import (
	"bytes"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	When     time.Time
	Op       Op
	Err      error
	Message  string
	Level    Severity
	UserID   UserID
	CtxID    uuid.UUID
	Kind     EventKind
	Unknowns []interface{}
}

type Op string

type UserID string

type EventKind int

const (
	UnknownError EventKind = iota
	NotFound
	Marshalling
)

func (ek EventKind) String() string {
	switch ek {
	case NotFound:
		return "not found"
	default:
		return "unknown error"
	}
}

type Severity int

const (
	NoLevel Severity = iota
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
	CriticalLevel
)

func (e *Event) ContextID() string {
	if e.CtxID == uuid.Nil {
		return ""
	}
	return e.CtxID.String()
}

func (s Severity) String() string {
	switch s {
	case DebugLevel:
		return "DEBG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERRO"
	case CriticalLevel:
		return "CRIT"
	default:
		return "UNKN"
	}
}

func pad(b *bytes.Buffer, str string) {
	if b.Len() == 0 {
		return
	}
	b.WriteString(str)
}

func (e *Event) isZero() bool {
	return e.UserID == "" && e.Op == "" && e.Kind == 0 && e.Err == nil
}

func (e *Event) Error() string {
	b := &bytes.Buffer{}
	if e.Op != "" {
		pad(b, ": ")
		b.WriteString(string(e.Op))
	}
	if len(e.Message) > 0 {
		pad(b, ": ")
		b.WriteString(e.Message)
	}
	if e.Kind != 0 {
		pad(b, ": ")
		b.WriteString(e.Kind.String())
	}
	if e.Err != nil {
		if prevErr, ok := e.Err.(*Event); ok {
			if !prevErr.isZero() {
				pad(b, "-> ")
				b.WriteString(e.Err.Error())
			}
		} else {
			pad(b, ": ")
			b.WriteString(e.Err.Error())
		}
	}
	if len(e.Unknowns) > 0 {
		b.WriteString("Unknown error")
	}
	if b.Len() == 0 {
		return "no error"
	}
	return b.String()
}

func (e *Event) Unwrap() error { return e.Err }

func ParseLevel(s string) Severity {
	if len(s) == 0 {
		return DebugLevel
	}
	s = strings.ToUpper(s)
	f := s[0]
	if f == '[' {
		f = s[1]
	}
	switch f {
	case 'D', 'T':
		return DebugLevel
	case 'I', 'L':
		return InfoLevel
	case 'W':
		return WarnLevel
	case 'E':
		return ErrorLevel
	case 'C', 'F':
		return CriticalLevel
	default:
		return NoLevel
	}

}

func E(msg string, args ...interface{}) error {
	e := &Event{When: time.Now().UTC()}
	e.Message = msg
	for _, arg := range args {
		switch arg := arg.(type) {
		case error:
			e.Err = arg
		case Op:
			e.Op = arg
		case Severity:
			e.Level = arg
		case UserID:
			e.UserID = arg
		case uuid.UUID:
			e.CtxID = arg
		case EventKind:
			e.Kind = arg
		default:
			e.Unknowns = append(e.Unknowns, arg)
		}
	}
	if e.Op == "" {
		e.Op = Op(getCaller(isInternal))
	}
	if e.Err != nil && e.Level == NoLevel {
		e.Level = ErrorLevel
	}

	return e
}
