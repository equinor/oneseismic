package message

import (
	"encoding/json"

	"github.com/vmihailenco/msgpack/v5"
)

type Message interface {
	Pack()         ([]byte, error)
}

/*
 * Corresponds to basic_query and derivatives in oneseismic/messages.hpp. This
 * is the process prototype that groups request parameters, request metadata,
 * authorization etc. into a single message.
 */
type Query struct {
	Pid             string          `json:"pid"`
	UrlQuery        string          `json:"url-query"`
	Guid            string          `json:"guid"`
	Manifest        json.RawMessage `json:"manifest"`
	StorageEndpoint string          `json:"storage_endpoint"`
	Function        string          `json:"function"`
	Args            interface {}    `json:"args"`
	Opts            interface {}    `json:"opts"`
}

func (msg *Query) Pack() ([]byte, error) {
	return json.Marshal(msg)
}

func (msg *Query) Unpack(doc []byte) (*Query, error) {
	return msg, json.Unmarshal(doc, msg)
}

/*
 * Corresponds to basic_task and derivatives in oneseismic/messages.hpp, and
 * is read from the worker nodes when performing a task, which combined makes
 * up a process.

 * Only the useful (to go) fields are parsed - the actual document has more
 * fields.
 */
type Task struct {
	Pid             string       `json:"pid"`
	Token           string       `json:"token"`
	UrlQuery        string       `json:"url-query"`
	Guid            string       `json:"guid"`
	StorageEndpoint string       `json:"storage_endpoint"`
	Function        string       `json:"function"`
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

type DimensionDescription struct {
	Dimension int   `json:"dimension"`
	Size      int   `json:"size"`
	Keys      []int `json:"keys"`
}

func (m *DimensionDescription) Pack() ([]byte, error) {
	return json.Marshal(m)
}

/*
 * This document is written as a "process header" in redis when a process is
 * scheduled, and holds the required information to build the response header.
 * The response header are parameters the client can use to determine how to
 * parse the response and properly pre-allocate buffers.
 */
type ProcessHeader struct {
	/*
	 * The number of separate parts this is broken into, where each part can be
	 * handled by a separate worker. This is the number of "bundles"
	 * (parts-of-results) the client will receive.
	 */
	Ntasks int    `msgpack:"nbundles"`
	RawHeader []byte
}

func (m *ProcessHeader) Pack() ([]byte, error) {
	return msgpack.Marshal(m);
}

func (m *ProcessHeader) Unpack(doc []byte) (*ProcessHeader, error) {
	m.RawHeader = doc
	// Skip the first byte (the envelope), which should always be array-len = 2
	// but would make the msgpack object incomplete. We only care about the map
	// that follows immediately after
	return m, msgpack.Unmarshal(doc[1:], m)
}
