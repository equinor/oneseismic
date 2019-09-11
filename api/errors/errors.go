package errors

import (
	"bytes"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	When    time.Time
	Err     error
	Message string
	Op      Op
	Path    string
	UserID  string
	CtxId   uuid.UUID
	Kind    EventKind
	Level   Severity
}

type Op string

type EventKind int

const (
	UnknownError EventKind = iota
	NotExists
)

func (ek EventKind) String() string {
	switch ek {
	case NotExists:
		return "item doesn't exist"
	default:
		return "Unknown error kind"
	}
}

type Severity int

const (
	DebugLevel Severity = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	CriticalLevel
)

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
	return e.Path == "" && e.UserID == "" && e.Op == "" && e.Kind == 0 && e.Err == nil
}

func (e *Event) Error() string {
	b := new(bytes.Buffer)
	if e.Op != "" {
		pad(b, ": ")
		b.WriteString(string(e.Op))
	}
	if e.Path != "" {
		pad(b, ": ")
		b.WriteString(string(e.Path))
	}
	if e.UserID != "" {
		if e.Path == "" {
			pad(b, ": ")
		} else {
			pad(b, ", ")
		}
		b.WriteString("user ")
		b.WriteString(string(e.UserID))
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
	if b.Len() == 0 {
		return "no error"
	}
	return b.String()
}

func ParseLevel(s string) Severity {
	if len(s) == 0 {
		return DebugLevel
	}
	s = strings.ToUpper(s)
	f := s[0]
	switch f {
	case 'D':
	case 'T':
		return DebugLevel
	case 'I':
	case 'L':
		return InfoLevel
	case 'W':
		return WarnLevel
	case 'E':
		return ErrorLevel
	case 'C':
	case 'F':
		return CriticalLevel
	default:
		return DebugLevel
	}
	return DebugLevel
}

func Message() string {
	return ""
}

func E(args ...interface{}) *Event {
	e := &Event{When: time.Now().UTC()}
	for _, arg := range args {
		switch arg := arg.(type) {
		case Op:
			e.Op = arg
		case error:
			e.Err = arg
		case EventKind:
			e.Kind = arg
		case Severity:
			e.Level = arg
		case string:
			e.Message = arg
		default:
			panic("bad call to E")
		}
	}
	return e
}
