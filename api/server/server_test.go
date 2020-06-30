package server

import (
	"crypto/rand"
	"crypto/rsa"
	"net/url"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/equinor/oneseismic/api/oneseismic"
	"github.com/google/uuid"
	"github.com/kataras/iris/v12/httptest"
	"github.com/pebbe/zmq4"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestSlicer(t *testing.T) {
	keys, jwt := mockRSAKeysJwt()
	issuer := ""
	storageEndpoint, _ := url.Parse("http://some.url")
	account := ""
	accountKey := ""
	zmqReqAddr := "inproc://" + uuid.New().String()
	zmqRepAddr := "inproc://" + uuid.New().String()

	go coreMock(zmqReqAddr, zmqRepAddr)
	app, err := App(keys, issuer, *storageEndpoint, account, accountKey, zmqReqAddr, zmqRepAddr)
	assert.Nil(t, err)

	e := httptest.New(t, app)
	jsonResponse := e.GET("/some_guid/slice/0/0").
		WithHeader("Authorization", "Bearer "+jwt).
		Expect().
		Status(httptest.StatusOK).
		JSON()
	jsonResponse.Path("$.tiles[0].layout.chunk_size").Number().Equal(1)
	jsonResponse.Path("$.tiles[0].v").Array().Elements(0.1)
}

func coreMock(reqNdpt string, repNdpt string) {
	in, _ := zmq4.NewSocket(zmq4.PULL)
	in.Connect(reqNdpt)

	out, _ := zmq4.NewSocket(zmq4.ROUTER)
	out.SetRouterMandatory(1)
	out.Connect(repNdpt)

	for {
		m, _ := in.RecvMessageBytes(0)
		pr := partitionRequest{}
		pr.loadZMQ(m)
		fr := oneseismic.FetchResponse{Requestid: pr.jobID}
		fr.Function = &oneseismic.FetchResponse_Slice{
			Slice: &oneseismic.SliceResponse{
				Tiles: []*oneseismic.SliceTile{
					{
						Layout: &oneseismic.SliceLayout{
							ChunkSize:  1,
							Iterations: 0,
						},
						V: []float32{0.1},
					},
				},
			},
		}

		bytes, _ := proto.Marshal(&fr)
		partial := partialResult {
			address: pr.address,
			jobID: pr.jobID,
			payload: bytes,
		}

		_, err = partial.sendZMQ(out)

		for err == zmq4.EHOSTUNREACH {
			_, err = out.SendMessage(m)
		}
	}
}

func mockRSAKeysJwt() (map[string]rsa.PublicKey, string) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	kid := "a"

	keys := make(map[string]rsa.PublicKey)
	keys[kid] = *privateKey.Public().(*rsa.PublicKey)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{})
	token.Header["kid"] = kid
	jwt, err := token.SignedString(privateKey)
	if err != nil {
		panic(err)
	}

	return keys, jwt
}
