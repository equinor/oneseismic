package server

import (
	"net/url"

	"github.com/google/uuid"
	"github.com/kataras/iris/v12"
)

func registerStoreController(app *iris.Application, storageEndpoint url.URL, accountName, accountKey string) error {
	sURL, err := newServiceURL(storageEndpoint, accountName, accountKey)
	if err != nil {
		return err
	}

	sc := storeController{sURL}
	app.Get("/", sc.list)
	app.Get("/{guid:string}", sc.services)
	app.Get("/{guid:string}/slice", sc.dimensions)
	app.Get("/{guid:string}/slice/{dimension:int32}", sc.lines)

	return nil
}

func RegisterSlicer(
	app *iris.Application,
	storageEndpoint string,
	reqNdpt string,
	repNdpt string,
	root string,
	mPlexName string,
) {
	sc := createSliceController(reqNdpt, repNdpt, storageEndpoint, root, mPlexName)

	app.Get("/{guid:string}/slice/{dim:int32}/{lineno:int32}", sc.get)
}

func Register(
	app *iris.Application,
	storageEndpoint url.URL,
	accountName,
	accountKey,
	reqNdpt,
	repNdpt string,
) error {
	if err := registerStoreController(app, storageEndpoint, accountName, accountKey); err != nil {
		return err
	}
	mPlexName := uuid.New().String()
	RegisterSlicer(app, storageEndpoint.String(), reqNdpt, repNdpt, accountName, mPlexName)

	return nil
}
