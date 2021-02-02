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

func (msg *Task) Unpack(doc []byte) (*Task, error) {
	return msg, json.Unmarshal(doc, msg)
}

type Manifest struct {
	Dimensions [][]int `json:"dimensions"`
}

func (m *Manifest) Pack() ([]byte, error) {
	return json.Marshal(m)
}

func (m *Manifest) Unpack(doc []byte) (*Manifest, error) {
	return m, json.Unmarshal(doc, m)
}


type SliceParams struct {
	Dim    int `json:"dim"`
	Lineno int `json:"lineno"`
}

type DimensionDescription struct {
	Dimension int   `json:"dimension"`
	Size      int   `json:"size"`
	Keys      []int `json:"keys"`
}

func (m *DimensionDescription) Pack() ([]byte, error) {
	return json.Marshal(m)
}
