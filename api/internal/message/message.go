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
	Pid             string       `json:"pid"`
	Token           string       `json:"token"`
	Guid            string       `json:"guid"`
	Manifest        interface {} `json:"manifest"`
	StorageEndpoint string       `json:"storage_endpoint"`
	Function        string       `json:"function"`
	Args            interface {} `json:"args"`
	Opts            interface {} `json:"opts"`
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


type SliceParams struct {
	Dim    int `json:"dim"`
	Lineno int `json:"lineno"`
}

type CurtainParams struct {
	Dim0s []int `json:"dim0s"`
	Dim1s []int `json:"dim1s"`
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
	 * The pid for this process. It is usually redundant from the api/result
	 * perspective since the document itself will be stored (namespaced) with
	 * the pid as key, but having it easily available is quite useful for other
	 * parts of the system. Additionally, it can be used to sanity check data
	 * and request parameters.
	 */
	Pid string    `msgpack:"pid"`
	/*
	 * The number of separate parts this is broken into, where each part can be
	 * handled by a separate worker. This is the number of "bundles"
	 * (parts-of-results) the client will receive.
	 */
	Ntasks int    `msgpack:"nbundles"`
	/*
	 * The shape of the result *with padding*. It shall always hold that
	 * shape[n] >= len(index[n]) and len(shape) == len(index). This is an
	 * advice to clients that they can use to pre-allocate buffers - a buffer
	 * of size product(shape...) will hold the full response.
	 */
	Shape []int   `msgpack:"shape"`
	/*
	 * The index, i.e. the ordered set of keys for each dimension. This is only
	 * a (useful) suggestion for assembly, and data can be written in any
	 * order.
	 *
	 * While assembly must be aware that bundles may show up in any order,
	 * having a "map" (in the treasure map sense) of what shape and keys to
	 * expect is quite useful for pre-allocation, and stuff like building a
	 * language-specific index like in xarray in python.
	 */
	Index [][]int `msgpack:"index"`
	/*
	 * The attributes included in the request, such as cdpx, cdpy, cdpm etc.
	 * Getting attributes is just another task, but this is a parsing hint for
	 * the assembler.
	 */
	Attrs []string `msgpack:"attributes"`

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
