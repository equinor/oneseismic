package errors

import (
	"bytes"
	"time"

	"github.com/google/uuid"
)

type Error struct {
	When   time.Time
	Err    error
	Op     Op
	Path   string
	UserID string
	CtxId  uuid.UUID
	Kind   ErrorKind
	Level  Severity
}

type Op string

type ErrorKind int

const (
	UnknownError ErrorKind = iota
	NotExists
)

type Severity int

const (
	DebugLevel Severity = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	PanicLevel
)

func (ek ErrorKind) String() string {
	switch ek {
	case NotExists:
		return "item doesn't exist"
	default:
		return "Unknown error kind"
	}
}

func pad(b *bytes.Buffer, str string) {
	if b.Len() == 0 {
		return
	}
	b.WriteString(str)
}

func (e *Error) isZero() bool {
	return e.Path == "" && e.UserID == "" && e.Op == "" && e.Kind == 0 && e.Err == nil
}

func (e *Error) Error() string {
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
		if prevErr, ok := e.Err.(*Error); ok {
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

func E(args ...interface{}) error {
	e := &Error{When: time.Now()}
	for _, arg := range args {
		switch arg := arg.(type) {
		case Op:
			e.Op = arg
		case error:
			e.Err = arg
		case ErrorKind:
			e.Kind = arg
		default:
			panic("bad call to E")
		}
	}
	return e
}
