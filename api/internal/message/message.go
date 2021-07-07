package message

import (
	"bytes"
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
	Pid string    `json:"pid"`
	/*
	 * The number of separate parts this is broken into, where each part can be
	 * handled by a separate worker. This is the number of "bundles"
	 * (parts-of-results) the client will receive.
	 */
	Ntasks int    `json:"ntasks"`
	/*
	 * The shape of the result *with padding*. It shall always hold that
	 * shape[n] >= len(index[n]) and len(shape) == len(index). This is an
	 * advice to clients that they can use to pre-allocate buffers - a buffer
	 * of size product(shape...) will hold the full response.
	 */
	Shape []int   `json:"shape"`
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
	Index [][]int `json:"index"`
}

func (m *ProcessHeader) Pack() ([]byte, error) {
	return json.Marshal(m)
}

func (m *ProcessHeader) Unpack(doc []byte) (*ProcessHeader, error) {
	return m, json.Unmarshal(doc, m)
}

/*
 * The header written as the first part of the end-user result, and meant to be
 * decoded by the clients. Since this is client-facing it has much higher
 * stability requirements than most messages in oneseismic.
 */
type ResultHeader struct {
	Bundles int
	Shape   []int
	Index   [][]int
}

/*
 * Pack the result header. Please note that the result of Pack() is *not* a
 * valid msgpack message - it assumes that the caller will add Bundles elements
 * after to complete the array.
 */
func (rh *ResultHeader) Pack() ([]byte, error) {
	var b bytes.Buffer
	enc := msgpack.NewEncoder(&b)

	if err := enc.EncodeArrayLen(2); err != nil {
		return nil, err
	}
	if err := enc.EncodeMapLen(3); err != nil {
		return nil, err
	}
	err := enc.EncodeMulti(
		"bundles", rh.Bundles,
		"shape",   rh.Shape,
		"index",   rh.Index,
	)
	if err != nil {
		return nil, err
	}
	if err := enc.EncodeArrayLen(rh.Bundles); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
