package tests

import (
	"log"
	"strings"
	"testing"

	"github.com/equinor/oneseismic/api/api"
	"github.com/equinor/oneseismic/api/internal/datastorage"
)

func TestLinenumbersAuthorization(t *testing.T) {
	datastorage.Storage = datastorage.NewFileStorage(getDataPath())
	query := `
	query testLinenumbers($cubeId: ID!)
	{
		cube(id: $cubeId) { linenumbers }
	}
	`
	_,_, opts, _ := setupRedis()

	// Request existing cube from authorized user
	m := runQuery(t, opts, query,
				  map[string]interface{}{"cubeId":"cube0"},
				  "0")
	ln := m.Get("data.cube.linenumbers[2]")
	if ln.Data() == nil {
		t.Fatalf("Bad data returned")
	}
	// Verify (part of the) returned data
	firstLineNumbers := ln.Data().([]interface{})
	for i,v := range []int{0, 4000, 8000, 12000, 16000}  {
		if firstLineNumbers[i] != v {
			t.Fatalf("Wrong linenumbers in manifest")
		}
	}

	// Existing cube from existing user without autorization
	m = runQuery(t, opts, query,
				map[string]interface{}{"cubeId":"cube0"},
				"1")
	ln = m.Get("data") ; if ln.Data() != nil {
		t.Fatalf("Should not return data")
	}
	ln = m.Get("errors")
	// log.Printf("Errors==%s", ln.Data())
	if ln.Data() == nil {
		t.Fatalf("Should return errors")
	}
	ln = m.Get("errors[0].extensions.type")
	if ln.Int() != api.IllegalAccessErrorType {
		t.Fatalf("Expected error-code %v", api.IllegalAccessErrorType)
	}

	// Non-existing cube, existing user
	m = runQuery(t, opts, query,
				map[string]interface{}{"cubeId":"non-existing"},
				"0")
	ln = m.Get("data") ; if ln.Data() != nil {
		t.Fatalf("Should not return data")
	}
	ln = m.Get("errors")
	log.Printf("Errors==%s", ln.Data())
	if ln.Data() == nil {
		t.Fatalf("Should return errors")
	}
	ln = m.Get("errors[0].extensions.type")
	if ln.Int() != api.NonExistingErrorType {
		t.Fatalf("Expected error-code %v", api.NonExistingErrorType)
	}

	// Finally, Unknown user
	m = runQuery(t, opts, query,
				map[string]interface{}{"cubeId":"non-existing"},
				"unknown-user")
	ln = m.Get("data") ; if ln.Data() != nil {
		t.Fatalf("Should not return data")
	}
	ln = m.Get("errors")
	log.Printf("Errors==%s", ln.Data())
	if ln.Data() == nil {
		t.Fatalf("Should return errors")
	}
	ln = m.Get("errors[0].extensions.type")
	if ln.Int() != api.IllegalAccessErrorType {
		t.Fatalf("Expected error-code %v", api.IllegalAccessErrorType)
	}
}

func TestLinenumbersMalformedQuery(t *testing.T) {
	datastorage.Storage = datastorage.NewFileStorage(getDataPath())
	// This is a malformed query, note hyphen...
	query := `
	query testLinenumbers($cubeId: ID!)
	{
		cube(id: $cubeId) { malformed-linenumbers }
	}
	`
	_,_, opts, _ := setupRedis()
	m := runQuery(t, opts, query,
				  map[string]interface{}{"cubeId":"cube0"},
				  "0")

	ln := m.Get("data") ; if ln.Data() != nil {
		t.Fatalf("Should not return data")
	}
	ln = m.Get("errors[0].message") ; if ln.Data() == nil {
		t.Fatalf("Should return some errors")
	}
	if !strings.Contains(ln.Data().(string), "syntax error") {
		t.Fatalf("First error should be a variant of syntax-error")
	}
}

func TestLinenumbersIllegalProperty(t *testing.T) {
	datastorage.Storage = datastorage.NewFileStorage(getDataPath())
	query := `
	query testLinenumbers($cubeId: ID!)
	{
		cube(id: $cubeId) { illegalFieldCookie }
	}
	`
	_,_, opts, _ := setupRedis()
	m := runQuery(t, opts, query,
				  map[string]interface{}{"cubeId":"cube0"},
				  "0")
	log.Printf("Response==%s", m)

	ln := m.Get("data") ; if ln.Data() != nil {
		t.Fatalf("Should not return data")
	}
	ln = m.Get("errors[0].message") ; if ln.Data() == nil {
		t.Fatalf("Should return some errors")
	}
	if !strings.Contains(ln.Data().(string), "illegalFieldCookie") {
		t.Fatalf("First error should complain about illegalFieldCookie")
	}
}

func TestLinenumbersManifestMissingDimensions(t *testing.T) {
// Test a malformed manifest (missing line-numbers) 
	datastorage.Storage = WrappedFileStorage{
		datastorage.NewFileStorage(getDataPath()),
		func(orig string) string {
			return strings.Replace(string(orig), "line-numbers", "Line-Numbers", -1)
		},
	}
	query := `
	query testLinenumbers($cubeId: ID!)
	{
		cube(id: $cubeId) { linenumbers }
	}
	`
	_,_, opts, _ := setupRedis()
	m := runQuery(t, opts, query,
				  map[string]interface{}{"cubeId":"cube0"},
				  "0")
	ln := m.Get("data") ; if ln.Data() != nil {
		t.Fatalf("Should not return data")
	}
	ln = m.Get("errors[0].extensions.type")
	if ln.Int() != api.InternalErrorType {
		t.Fatalf("Expected error-code %v", api.InternalErrorType)
	}
}

func TestLinenumbersManifestBadlyTyped(t *testing.T) {
// Test malformed manifest (linenumbers not convertible to floats)
	datastorage.Storage = WrappedFileStorage{
		datastorage.NewFileStorage(getDataPath()),
		func(orig string) string {
			return strings.Replace(string(orig),"[ 1, 2, 3, 4, 5 ]",
												"[ \"one\", \"two\", 3, 4, 5 ]", -1)
		},
	}
	query := `
	query testLinenumbers($cubeId: ID!)
	{
		cube(id: $cubeId) { linenumbers }
	}
	`
	_,_, opts, _ := setupRedis()
	m := runQuery(t, opts, query,
				  map[string]interface{}{"cubeId":"cube0"},
				  "0")
	ln := m.Get("data") ; if ln.Data() != nil {
		t.Fatalf("Should not return data")
	}
	ln = m.Get("errors[0].extensions.type")
	if ln.Int() != api.InternalErrorType {
		t.Fatalf("Expected error-code %v", api.InternalErrorType)
	}
}
