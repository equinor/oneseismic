package server

import (
	"github.com/google/uuid"
	"github.com/kataras/iris/v12"
)

func registerStoreController(app *iris.Application, az AzureBlobSettings) error {
	sURL, err := newServiceURL(az)
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
	a AzureBlobSettings,
	reqNdpt string,
	repNdpt string,
) error {
	if err := registerStoreController(app, a); err != nil {
		return err
	}
	mPlexName := uuid.New().String()
	RegisterSlicer(app, reqNdpt, repNdpt, a.AccountName, mPlexName)

	return nil
}
