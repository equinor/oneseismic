package internal

import (
	"net/http"
)

type InternalE struct {
	msg string
}

func NewInternalError() *InternalE {
	return &InternalE{ msg: "Internal error" }
}

func InternalError(msg string) *InternalE {
	return &InternalE{ msg: msg }
}

func (ie *InternalE) Error() string {
	return ie.msg
}

type PermissionDeniedE struct {
	msg string
}

func PermissionDenied(msg string) *PermissionDeniedE {
	return &PermissionDeniedE{ msg: msg }
}

func PermissionDeniedFromStatus(status int) *PermissionDeniedE {
	return &PermissionDeniedE{ http.StatusText(status) }
}

func (pd *PermissionDeniedE) Error() string {
	return pd.msg
}

type QueryE struct {
	msg string
}

func QueryError(msg string) *QueryE {
	return &QueryE{ msg: msg }
}

func (qe *QueryE) Error() string {
	return qe.msg
}

type NotFoundE struct {
	msg string
}

func NotFound(msg string) *NotFoundE {
	return &NotFoundE{ msg: msg }
}

func NewNotFoundError() *NotFoundE {
	return &NotFoundE{}
}

func (nf *NotFoundE) Error() string {
	return nf.msg
}
