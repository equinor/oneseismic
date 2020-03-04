package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/equinor/seismic-cloud/api/service"
	"github.com/kataras/iris/v12"
	irisCtx "github.com/kataras/iris/v12/context"
	"github.com/stretchr/testify/mock"
)

type MockManifestStore struct {
	mock.Mock
}

func GenerateManifest(id string) *service.Manifest {
	return &service.Manifest{Guid: id}
}

func (ms *MockManifestStore) Download(ctx context.Context, id string) (*service.Manifest, error) {

	args := ms.Called(ctx, id)
	arg0 := args.Get(0)
	m, ok := arg0.(*service.Manifest)
	if !ok {
		return nil, fmt.Errorf("Manifest Download is not a manifest")
	}
	return m, args.Error(1)
}

type ServiceSetup struct {
	ManifestStore service.ManifestStore
	Ctx           irisCtx.Context
	Recorder      *httptest.ResponseRecorder
}

func NewTestServiceSetup() *ServiceSetup {
	ctx := irisCtx.NewContext(iris.Default())
	ms := &MockManifestStore{}
	recorder := httptest.NewRecorder()

	mani := GenerateManifest("exists")
	ms.On("Download", mock.Anything, "exists").Return(mani, nil)
	ms.On("Download", mock.Anything, "not-exists").Return(nil, fmt.Errorf("Not exist"))

	return &ServiceSetup{ms, ctx, recorder}
}

func (ts *ServiceSetup) Result() *http.Response {
	return ts.Recorder.Result()
}
func (ts *ServiceSetup) BeginRequest(r *http.Request) {
	_ = r.ParseForm()
	ts.Ctx.BeginRequest(ts.Recorder, r)
}

func (ts *ServiceSetup) EndRequest() {
	ts.Ctx.EndRequest()
}

func (ts *ServiceSetup) SetParam(id string, v string) {
	ts.Ctx.Params().Set(id, v)
}
