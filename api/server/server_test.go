package server

import (
	"crypto/rand"
	"crypto/rsa"
	"net/url"
	"testing"
	"log"

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
	zmqFailureAddr := "inproc://" + uuid.New().String()

	go coreMock(zmqReqAddr, zmqRepAddr, zmqFailureAddr)
	app, err := App(keys, issuer, *storageEndpoint, account, accountKey, zmqReqAddr, zmqRepAddr, zmqFailureAddr)
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

func coreMock(reqNdpt string, repNdpt string, failureAddr string) {
	in, _ := zmq4.NewSocket(zmq4.PULL)
	in.Connect(reqNdpt)

	out, _ := zmq4.NewSocket(zmq4.ROUTER)
	out.SetRouterMandatory(1)
	out.Connect(repNdpt)

	for {
		m, _ := in.RecvMessageBytes(0)
		proc := process{}
		err := proc.loadZMQ(m)
		if err != nil {
			msg := "Broken process (loadZMQ) in core emulation: %s"
			log.Fatalf(msg, err.Error())
		}
		fr := oneseismic.FetchResponse{Requestid: proc.pid}
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
		partial := routedPartialResult {
			address: proc.address,
			partial: partialResult {
				pid: proc.pid,
				n: 0,
				m: 1,
				payload: bytes,
			},
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
