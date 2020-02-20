package tests

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/equinor/seismic-cloud/api/service"
	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/kataras/iris/v12"
	irisCtx "github.com/kataras/iris/v12/context"
	"github.com/stretchr/testify/mock"
)

func (m *MockStitch) Stitch(ctx context.Context, out io.Writer, sp service.StitchParams) (string, error) {

	args := m.Called(ctx, out, sp)

	return args.String(0), args.Error(1)
}

type MockStitch struct {
	mock.Mock
}

type MockManifestStore struct {
	mock.Mock
}

func GenerateManifest(id string) *store.Manifest {
	return &store.Manifest{Guid: id}
}

func (ms *MockManifestStore) Download(ctx context.Context, id string) (*store.Manifest, error) {

	args := ms.Called(ctx, id)
	arg0 := args.Get(0)
	m, ok := arg0.(*store.Manifest)
	if !ok {
		return nil, fmt.Errorf("Manifest Download is not a manifest")
	}
	return m, args.Error(1)
}

func (ms *MockManifestStore) Upload(ctx context.Context, id string, man store.Manifest) error {

	args := ms.Called(ctx, id, man)

	return args.Error(0)
}

type ServiceSetup struct {
	ManifestStore store.ManifestStore
	Stitch        *MockStitch
	Ctx           irisCtx.Context
	Recorder      *httptest.ResponseRecorder
}

func NewTestServiceSetup() *ServiceSetup {
	stitch := &MockStitch{}
	ctx := irisCtx.NewContext(iris.Default())
	ms := &MockManifestStore{}
	recorder := httptest.NewRecorder()

	mani := GenerateManifest("exists")
	ms.On("Download", mock.Anything, "exists").Return(mani, nil)
	ms.On("Download", mock.Anything, "not-exists").Return(nil, fmt.Errorf("Not exist"))
	stitch.On("Stitch", mock.Anything, service.StitchParams{Dim: 0, CubeManifest: mani}).Return()

	return &ServiceSetup{ms, stitch, ctx, recorder}
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
