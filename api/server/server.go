package server

import (
	"net/url"

	"github.com/kataras/iris/v12"
)

// Register endpoints for oneseismic
func Register(
	app *iris.Application,
	storageEndpoint url.URL,
	accountName string,
	accountKey string,
	zmqReqAddr,
	zmqRepAddr string,
	zmqFailureAddr string,
) error {
	sURL, err := newServiceURL(storageEndpoint, accountName, accountKey)
	if err != nil {
		return err
	}

	sessions := newSessions()
	go sessions.Run(zmqReqAddr, zmqRepAddr, zmqFailureAddr)

	sc := storeController{sURL}
	app.Get("/", sc.list)
	app.Get("/{guid:string}", sc.services)
	app.Get("/{guid:string}/slice", sc.dimensions)
	app.Get("/{guid:string}/slice/{dimension:int32}", sc.lines)

	slice := sliceController {
		slicer: &slicer {
			root: accountName,
			endpoint: storageEndpoint.String(),
			sessions: sessions,
		},
	}
	app.Get("/{guid:string}/slice/{dim:int32}/{lineno:int32}", slice.get)

	return nil
}
