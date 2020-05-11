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

	mc := storeController{sURL}
	app.Get("/", mc.list)

	return nil
}

func RegisterSlicer(app *iris.Application, reqNdpt string, repNdpt string, root string, mPlexName string) {
	sc := createSliceController(reqNdpt, repNdpt, root, mPlexName)

	app.Get("/{guid:string}/slice/{dim:int32}/{ordinal:int32}", sc.get)
}

func Register(app *iris.Application, a AzureBlobSettings, reqNdpt string, repNdpt string) error {
	err := registerStoreController(app, a)
	if err != nil {
		return err
	}
	mPlexName := uuid.New().String()
	RegisterSlicer(app, reqNdpt, repNdpt, a.AccountName, mPlexName)

	return nil
}
