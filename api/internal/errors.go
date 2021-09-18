package internal

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
