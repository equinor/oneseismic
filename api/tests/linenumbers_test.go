package tests

import (
	"strings"
	"testing"

	"github.com/equinor/oneseismic/api/api"
	"github.com/equinor/oneseismic/api/internal/datastorage"
	"github.com/stretchr/testify/assert"
)

func TestLinenumbersAuthorization(t *testing.T) {
	datastorage.Storage = datastorage.NewFileStorage("test", getDataPath())
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
	assert.NotEqual(t, nil, ln.String(), "Expected some data")
	// Verify (part of the) returned data
	firstLineNumbers := ln.Data().([]interface{})
	for i,v := range []int{0, 4000, 8000, 12000, 16000}  {
		assert.Equal(t, firstLineNumbers[i], v,
			       "Wrong linenumbers in manifest")
	}

	// Existing cube from existing user without autorization
	m = runQuery(t, opts, query,
				map[string]interface{}{"cubeId":"cube0"},
				"1")
	assert.Equal(t, "", m.Get("data").String(), "Should not return data")
	assert.NotEqual(t, "", m.Get("errors").String(), "Should return errors")
	assert.Equal(t, api.IllegalAccessErrorType,
				 m.Get("errors[0].extensions.type").Int(),
				"Wrong error-code")

	// Non-existing cube, existing user
	m = runQuery(t, opts, query,
				map[string]interface{}{"cubeId":"non-existing"},
				"0")
	assert.Equal(t, "", m.Get("data").String(), "Should not return data")
	assert.NotEqual(t, "", m.Get("errors").String(), "Should return errors")
	assert.Equal(t, api.NonExistingErrorType,
				 m.Get("errors[0].extensions.type").Int(),
				"Wrong error-code")

	// Finally, Unknown user
	m = runQuery(t, opts, query,
				map[string]interface{}{"cubeId":"non-existing"},
				"unknown-user")
	assert.Equal(t, "", m.Get("data").String(), "Should not return data")
	assert.NotEqual(t, "", m.Get("errors").String(), "Should return errors")
	assert.Equal(t, api.IllegalAccessErrorType,
				 m.Get("errors[0].extensions.type").Int(),
				"Wrong error-code")
}

func TestLinenumbersMalformedQuery(t *testing.T) {
	datastorage.Storage = datastorage.NewFileStorage("test", getDataPath())
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

	assert.Equal(t, "", m.Get("data").String(), "Should not return data")
	assert.NotEqual(t, "", m.Get("errors").String(), "Should return errors")
	assert.Contains(t, m.Get("errors[0].message").String(), "syntax error")
}

func TestLinenumbersIllegalProperty(t *testing.T) {
	datastorage.Storage = datastorage.NewFileStorage("test", getDataPath())
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
	assert.Equal(t, "", m.Get("data").String(), "Should not return data")
	assert.NotEqual(t, "", m.Get("errors").String(), "Should return errors")
	assert.Contains(t, m.Get("errors[0].message").String(), "illegalFieldCookie")
}

func TestLinenumbersManifestMissingDimensions(t *testing.T) {
// Test a malformed manifest (missing line-numbers) 
	datastorage.Storage = WrappedFileStorage{
		datastorage.NewFileStorage("test", getDataPath()),
		func(token string, guid string, orig []byte) ([]byte, error) {
			if strings.Contains(guid, "manifest") {
				tmp := string(orig)
				tmp  = strings.Replace(tmp, "line-numbers", "Line-Numbers", -1)
				return []byte(tmp), nil
			}
			return orig, nil
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
	assert.Equal(t, "", m.Get("data").String(), "Should not return data")
	assert.NotEqual(t, "", m.Get("errors").String(), "Should return errors")
	assert.Equal(t, api.InternalErrorType,
				 m.Get("errors[0].extensions.type").Int(),
				"Wrong error-code")
}

func TestLinenumbersManifestBadlyTyped(t *testing.T) {
// Test malformed manifest (linenumbers not convertible to floats)
	datastorage.Storage = WrappedFileStorage{
		datastorage.NewFileStorage("test", getDataPath()),
		func(token string, guid string, orig []byte) ([]byte, error) {
			if strings.Contains(guid, "manifest") {
				tmp := string(orig)
				tmp  = strings.Replace(tmp,"[ 1, 2, 3, 4, 5 ]",
							"[ \"one\", \"two\", 3, 4, 5 ]", -1)
				return []byte(tmp), nil
			}
			return orig, nil
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

	assert.Equal(t, "", m.Get("data").String(), "Should not return data")
	assert.NotEqual(t, "", m.Get("errors").String(), "Should return errors")
	assert.Equal(t, api.InternalErrorType,
				 m.Get("errors[0].extensions.type").Int(),
				"Wrong error-code")
}
