package cmd

import (
	"testing"
	"time"

	"github.com/equinor/oneseismic/api/core"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/zeromq/goczmq"
)

func TestServeZMQError(t *testing.T) {
	addr := "inproc://TestServeZMQError"

	go serveZMQ(addr)
	time.Sleep(10 * time.Millisecond)

	dealer, err := goczmq.NewDealer(addr)
	assert.NoError(t, err)
	defer dealer.Destroy()

	buf, err := proto.Marshal(&core.ApiRequest{})
	err = dealer.SendFrame(buf, goczmq.FlagNone)
	assert.NoError(t, err)

	reply, err := dealer.RecvMessage()
	assert.NoError(t, err)

	assert.NotNil(t, reply)
	assert.True(t, len(reply[0]) > 0)

	//shutdown server
	err = dealer.SendFrame([]byte(""), goczmq.FlagNone)
}
