package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	"github.com/equinor/oneseismic/api/internal/message"
	"github.com/equinor/oneseismic/api/internal/util"
)

type MockStorage struct {}

func (s MockStorage) FetchManifest(
	ctx context.Context,
	token string,
	guid string,
) ([]byte, error) {
if token != "obo-0" || guid != "cube0" {
	return nil, api.NonExistingError{}
}
return []byte(`{}`), nil
}

// Wrapper to simplify patching a manifest in order to
// provoke errors
type WrappedFileStorage struct {
	*datastorage.FileStorage
	cb func(string)string
}
func (s WrappedFileStorage) FetchManifest(
	ctx context.Context,
	token string,
	guid string,
) ([]byte, error) {
	tmp, err := s.FileStorage.FetchManifest(ctx, token, guid) ; if err != nil {
		return nil, err
	}
	log.Printf("Initial==%s", string(tmp))
	return []byte(s.cb(string(tmp))), nil
}

// Path to where test-data is stored relative to THIS FILE
// TODO: can probably construct a way to get this relative to
// the CALLER of this method, but for now it is not necessary
func getDataPath() string {
	_, file, _, _ :=runtime.Caller(0)
	return path.Join(path.Dir(file),"testdata")
//	return path.Join(path.Dir(file),"..","..","tests","data")
}

func (s MockStorage) CreateBlobContainer(task message.Task) (api.AbstractBlobContainer, error) {
	return nil, nil
}
func (s MockStorage) GetUrl() *url.URL { return nil }

type MockTokens struct{ auth.Tokens }
func (t *MockTokens) GetOnbehalf(token string) (string, error) {
	if token == "0" { return "obo-0", nil }
	if token == "1" { return "obo-1", nil }
	return "", api.IllegalAccessError{}
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

//	doneChan := make(chan struct{})
//	go setupFetchProcess(context.Background(), opts, doneChan)

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
	log.Printf(
		"consumer %s in group %s connecting to stream %s",
		opts.consumerid,
		opts.group,
		opts.stream,
	)
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
							&MockTokens{},
							datastorage.CreateStorage("", ""))

	app.POST("/", gql.Post)
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, "/", &requestBody)
    request.Header.Set("Content-Type", "application/json")
	return app, request, response
}

/*
* Reduce boilerplate code in the actual tests...
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
