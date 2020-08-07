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
	zmqReqAddr,
	zmqRepAddr string,
	zmqFailureAddr string,
) {
	sc := storeController{&storageURL{storageEndpoint}}

	sessions := newSessions()
	go sessions.Run(zmqReqAddr, zmqRepAddr, zmqFailureAddr)

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

}
