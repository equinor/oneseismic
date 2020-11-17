package message

import (
	"encoding/json"
)

type Message interface {
	Pack()         ([]byte, error)
}

/*
 * Corresponds to common_task and derivatives in oneseismic/messages.hpp. This
 * is the process prototype that groups request parameters, request metadata,
 * authorization etc. into a single message.
 */
type Task struct {
	Pid             string       `json:"pid"`
	Token           string       `json:"token"`
	Guid            string       `json:"guid"`
	Manifest        string       `json:"manifest"`
	StorageEndpoint string       `json:"storage_endpoint"`
	Shape           []int32      `json:"shape"`
	Function        string       `json:"function"`
	Params          interface {} `json:"params"`
}

func (msg *Task) Pack() ([]byte, error) {
	return json.Marshal(msg)
}

type SliceParams struct {
	Dim    int `json:"dim"`
	Lineno int `json:"lineno"`
}
