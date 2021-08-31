package tests

import (
	"log"
	"testing"

	"github.com/equinor/oneseismic/api/internal/datastorage"
)

func TestSliceByLinenumber(t *testing.T) {
	datastorage.Storage = datastorage.NewFileStorage(getDataPath())
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
	log.Printf("Object returned from query: %v", m)
	ln := m.Get("data") ; if ln.Data() == nil {
		t.Fatalf("Expected data")
	}
	ln = m.Get("errors")
	if ln.Data() != nil {
		t.Fatalf("Unexpected errors: %v", ln.Data())
	}

	// Verify that the request was queued/registered in Redis
	msgs, err := redisStore.XReadGroup(redisCtx, &redisReadArgs).Result()
	if err != nil {
		t.Fatalf("Unable to read from redis: %v", err)
	} else {
		log.Printf("Object in redis: %v", msgs)
	}

// TODO: verify Query-response corresponds with content in Redis
// The only reasonable x-check I see so far is that the map in Redis
// should contain a pid corresponding to the result-id of the response
// `
// {
// # Structure from redis
// jobs [
//    {1630011050954-0
//      map[
//         part:0/1
//         pid:468d88a1-4014-45c2-a36e-72ff01b3ded6
//         task: {"attribute":"data","dim":2,"ext":"f32","function":"slice","guid":"0","ids":[[0,0,0],[0,1,0],[1,0,0],[1,1,0]],"idx":1,"pid":"468d88a1-4014-45c2-a36e-72ff01b3ded6","prefix":"src","shape":[4,4,4],"shape-cube":[5,5,50],"storage_endpoint":"","token":"0"}
//      ]
//    }
// ]
// }
//
// # Result from the query
// {"data":{"cube":{"sliceByLineno":{"url":"result/468d88a1-4014-45c2-a36e-72ff01b3ded6","key":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MzAwMTEzNTAsInBpZCI6IjQ2OGQ4OGExLTQwMTQtNDVjMi1hMzZlLTcyZmYwMWIzZGVkNiJ9.G3L6DQ7Etz9EdKUUUVPUykLsKEGlLQ1kAHQNM5tzyiE"}}}}
// `
}
