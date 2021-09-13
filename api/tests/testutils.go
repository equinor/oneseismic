package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"path"
	"runtime"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/objx"

	"github.com/equinor/oneseismic/api/api"
	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/datastorage"
	"github.com/equinor/oneseismic/api/internal/util"
)

type MockStorage struct {}

func (s MockStorage) Get(
	ctx context.Context,
	token string,
	guid string,
) ([]byte, error) {
if token != "obo-0" || guid != "cube0" {
	return nil, api.NonExistingError{}
}
return []byte(`{}`), nil
}

// Wrapper to simplify patching a result in order to eg. provoke errors
// The cb is the callback-func doing the actual patching
type WrappedFileStorage struct {
	*datastorage.FileStorage
	cb func([]byte)[]byte
}
func (s WrappedFileStorage) Get(
	ctx context.Context,
	token string,
	guid string,
) ([]byte, error) {
	tmp, err := s.FileStorage.Get(ctx, token, guid) ; if err != nil {
		return nil, err
	}
	return s.cb(tmp), nil
}

// Path to where test-data is stored relative to THIS FILE
// TODO: can probably construct a way to get this relative to
// the CALLER of this method, but for now it is not necessary
func getDataPath() string {
	_, file, _, _ :=runtime.Caller(0)
	return path.Join(path.Dir(file),"testdata")
//	return path.Join(path.Dir(file),"..","..","tests","data")
}
func (s MockStorage) GetEndpoint() string { return "" }
func (s MockStorage) GetKind() string { return "mock" }

type MockTokens struct{}
func (t *MockTokens) GetOnbehalf(token string) (string, error) {
	if token == "0" { return "obo-0", nil }
	if token == "1" { return "obo-1", nil }
	return "", api.NewIllegalAccessError(
		fmt.Sprintf("Unknown token %s", token))
}
func (t *MockTokens) Invalidate(auth string) {}


func xxTestMockStorage(t *testing.T) {
	datastorage.Storage = MockStorage{}
}

type RedisOpts struct {
	stream string
	group string
	consumerid string
	url string
}

func setupRedis() (context.Context, *redis.Client, RedisOpts, redis.XReadGroupArgs) {
	// A simple in-process redis
	redisServer, err := miniredis.Run()
	if err != nil { panic(err) }

	opts := RedisOpts {
		stream: "jobs",
		group:  "fetch",
		consumerid: "id",
		url: redisServer.Addr(),
	} // See fetch/main.go for default values

	ctx := context.Background()
	storage := redis.NewClient(&redis.Options {
		Addr: opts.url,
		DB: 0,
	})
	err = storage.XGroupCreateMkStream(ctx, opts.stream, opts.group, "0").Err()
	if err != nil {
		_, busygroup := err.(interface{RedisError()});
		if !busygroup {
			log.Fatalf(
				"Unable to create group %s for stream %s: %v",
				opts.group,
				opts.stream,
				err,
			)
		}
	}
	args := redis.XReadGroupArgs {
		Group:    opts.group,
		Consumer: opts.consumerid,
		Streams:  []string { opts.stream, ">", },
		Count:    1,
		NoAck:    true,
	}
	return ctx, storage, opts, args
}

func setupGraphqlEndpoint(opts RedisOpts, query string, vars map[string]interface{},) (*gin.Engine, *http.Request, *httptest.ResponseRecorder) {
	var requestBody bytes.Buffer
	requestBodyObj := struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}{
		Query:     query,
		Variables: vars,
	}
	if err := json.NewEncoder(&requestBody).Encode(requestBodyObj); err != nil {
		log.Fatal(err)
	}

	response := httptest.NewRecorder()
	ctx, app := gin.CreateTestContext(response)
	app.Use(util.GeneratePID) // inject pid - nothing works without it...

	// TODO: this should probably be joined with Tokens in an AuthProvider or similar
	kr := auth.MakeKeyring([]byte("test"))

	client := redis.NewClient(&redis.Options{
		Addr: opts.url,
	})

	gql := api.MakeGraphQL(&kr, client,
						datastorage.CreateStorage("", ""),
						&MockTokens{},
						)

	app.POST("/", gql.Post)
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, "/", &requestBody)
    request.Header.Set("Content-Type", "application/json")
	return app, request, response
}

/*
* Helper to reduce boilerplate code in the actual tests
*/
func runQuery(t *testing.T, opts RedisOpts, query string, vars map[string]interface{}, authToken string) objx.Map {
	app, request, response := setupGraphqlEndpoint(opts, query, vars)
	request.Header.Add("Authorization", authToken)
	app.ServeHTTP(response, request)
	responseString := response.Body.String()
	m, err := objx.FromJSON(responseString) ; if err != nil {
		t.Error(err)
	}
	return m
}
