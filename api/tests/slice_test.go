package tests

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/equinor/oneseismic/api/internal/datastorage"
	"github.com/equinor/oneseismic/api/internal/fetch"
	"github.com/stretchr/testify/assert"
)

func TestSliceByLinenumber(t *testing.T) {
	datastorage.Storage = datastorage.NewFileStorage("test",getDataPath())
	query := `
	query testSliceByLinenumbers($cubeId: ID!, $dim: Int!, $lineno: Int!)
	{
		cube(id: $cubeId) {
			sliceByLineno(dim: $dim, lineno: $lineno)
		}
	}
	`
	redisCtx, redisStore, redisOpts, redisReadArgs := setupRedis()

	// Request existing cube from authorized user
	m := runQuery(t, redisOpts, query,
					map[string]interface{}{
						"cubeId": "cube0",
						"dim": 2,
						"lineno" :4000},
				  "0")
	assert.NotEqual(t, nil, m.Get("data").Data(), "Expected data")
	assert.Equal(t, nil, m.Get("errors").Data(), "Unexpected errors")

	url := m.Get("data.cube.sliceByLineno.url").String()
	key := m.Get("data.cube.sliceByLineno.key").String()

	// Ensure that status for the query is "pending" or "working"
	statusQuery := fmt.Sprintf("/%s/status", url)
	m, _ = getResultAsObjx(t, redisOpts, statusQuery,
		                   fmt.Sprintf("Bearer %s", key))
	assert.Contains(t, []string{"pending", "working"},
	                   m.Get("status").String())

	// Run fetch-logic to move requested data from blob-storage to redis
	fetch.Run(redisCtx, redisStore, &redisReadArgs, 1, 1)

	// Request status until it changes...
	current := m.Get("status").String()
	for m.Get("status").String() == current {
		time.Sleep(50 * time.Millisecond) // yield cpu for a while...
		m, _ = getResultAsObjx(t, redisOpts, statusQuery,
			                   fmt.Sprintf("Bearer %s", key))
	}
	// ... then verify that query has finished
	assert.Equal(t, "finished", m.Get("status").String())

	// Now, request the data itself
	// TODO: How do we verify content??
	resultQuery := fmt.Sprintf("/%s/stream", url)
	res := getResult(t, redisOpts, resultQuery, fmt.Sprintf("Bearer %s", key))
	log.Printf("Received %v bytes", len(res))
}

func TestSliceByLinenumberFailingWithEOF(t *testing.T) {
	datastorage.Storage = WrappedFileStorage{
		datastorage.NewFileStorage("test", getDataPath()),
		func(token string, guid string, orig []byte) ([]byte, error) {
			time.Sleep(100 * time.Millisecond) // emulate network	
			// log.Printf("----> %v", guid)
			if strings.Contains(guid, "1-0-0.f32") {
				return nil, errors.New("Mocked EOF")
			}
			return orig, nil
		},
	}
	// Also ensure the fetch-module has this mocked storage
	fetch.StorageSingleton = datastorage.Storage

	query := `
	query testSliceByLinenumbers($cubeId: ID!, $dim: Int!, $lineno: Int!)
	{
		cube(id: $cubeId) {
			sliceByLineno(dim: $dim, lineno: $lineno)
		}
	}
	`
	redisCtx, redisStore, redisOpts, redisReadArgs := setupRedis()

	// Request existing cube from authorized user
	m := runQuery(t, redisOpts, query,
					map[string]interface{}{
						"cubeId": "cube0",
						"dim": 2,
						"lineno" :4000},
				  "0")
	assert.NotEqual(t, nil, m.Get("data").Data(), "Expected data")
	assert.Equal(t, nil, m.Get("errors").Data(), "Unexpected errors")

	url := m.Get("data.cube.sliceByLineno.url").String()
	key := m.Get("data.cube.sliceByLineno.key").String()

	// Ensure that status for the query is "pending" or "working"
	statusQuery := fmt.Sprintf("/%s/status", url)
	m, _ = getResultAsObjx(t, redisOpts, statusQuery,
		                   fmt.Sprintf("Bearer %s", key))
	assert.Contains(t, []string{"pending", "working"},
	                   m.Get("status").String())

	// Run fetch-logic to move requested data from blob-storage to redis
	fetch.Run(redisCtx, redisStore, &redisReadArgs, 1, 1)

	// Request status until it changes...
	current := m.Get("status").String()
	for m.Get("status").String() == current {
		// log.Printf("==> status=%s", current)
		time.Sleep(500 * time.Millisecond) // yield cpu for a while...
		m, _  = getResultAsObjx(t, redisOpts, statusQuery,
			                    fmt.Sprintf("Bearer %s", key))
	}
	// ... then verify that query failed
	assert.Equal(t, "failed", m.Get("status").String())

	// Now, request the data. We should receive a 500
	// (http internal-server-error)
	resultQuery := fmt.Sprintf("/%s/stream", url)
	res := getResponse(t, redisOpts, resultQuery, fmt.Sprintf("Bearer %s", key))
	// log.Printf("Received %v", res)
	assert.Equal(t, http.StatusInternalServerError, res.Result().StatusCode)
}
