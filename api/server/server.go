package server

import (
	"crypto/rsa"
	"net/url"

	"github.com/equinor/oneseismic/api/auth"
	"github.com/google/uuid"
	"github.com/kataras/iris/v12"
)

// App for oneseismic
func App(
	rsaKeys map[string]rsa.PublicKey,
	issuer string,
	storageEndpoint url.URL,
	accountName string,
	accountKey string,
	zmqReqAddr,
	zmqRepAddr string,
) (*iris.Application, error) {
	app := iris.Default()

	app.Use(auth.CheckJWT(rsaKeys))
	app.Use(auth.ValidateIssuer(issuer))
	app.Use(iris.Gzip)

	sURL, err := newServiceURL(storageEndpoint, accountName, accountKey)
	if err != nil {
		return nil, err
	}

	sc := storeController{sURL}
	app.Get("/", sc.list)
	app.Get("/{guid:string}", sc.services)
	app.Get("/{guid:string}/slice", sc.dimensions)
	app.Get("/{guid:string}/slice/{dimension:int32}", sc.lines)

	slicer := createSliceController(zmqReqAddr, zmqRepAddr, storageEndpoint.String(), accountName, uuid.New().String())
	app.Get("/{guid:string}/slice/{dim:int32}/{lineno:int32}", slicer.get)

	return app, nil
}
