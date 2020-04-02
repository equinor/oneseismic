package server

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/kataras/iris/v12"
)

func registerManifestController(app *iris.Application, a AzureBlobSettings) error {
	mc, err := createManifestController(a)
	if err != nil {
		return fmt.Errorf("creating manifestController: %w", err)
	}
	app.Get("/", mc.list)

	return nil
}

func RegisterSlicer(app *iris.Application, reqNdpt string, repNdpt string, root string, mPlexName string) {
	sc := createSliceController(reqNdpt, repNdpt, root, mPlexName)

	app.Get("/{guid:string}/slice/{dim:int32}/{ordinal:int32}", sc.get)
}

func Register(app *iris.Application, a AzureBlobSettings, reqNdpt string, repNdpt string) error {
	err := registerManifestController(app, a)
	if err != nil {
		return fmt.Errorf("register manifestController: %w", err)
	}
	mPlexName := uuid.New().String()
	RegisterSlicer(app, reqNdpt, repNdpt, a.AccountName, mPlexName)

	return nil
}
