package server

import (
	"github.com/google/uuid"
	"github.com/kataras/iris/v12"
)

func registerStoreController(app *iris.Application, storageURL, accountName, accountKey string) error {
	sURL, err := newServiceURL(storageURL, accountName, accountKey)
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
	reqNdpt string,
	repNdpt string,
	root string,
	mPlexName string,
) {
	sc := createSliceController(reqNdpt, repNdpt, root, mPlexName)

	app.Get("/{guid:string}/slice/{dim:int32}/{lineno:int32}", sc.get)
}

func Register(
	app *iris.Application,
	storageURL,
	accountName,
	accountKey,
	reqNdpt,
	repNdpt string,
) error {
	if err := registerStoreController(app, storageURL, accountName, accountKey); err != nil {
		return err
	}
	mPlexName := uuid.New().String()
	RegisterSlicer(app, reqNdpt, repNdpt, accountName, mPlexName)

	return nil
}
