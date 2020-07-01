package server

import (
	"net/url"

	"github.com/google/uuid"
	"github.com/kataras/iris/v12"
)

func registerStoreController(app *iris.Application, primaryURL url.URL, accountName, accountKey string) error {
	sURL, err := newServiceURL(primaryURL, accountName, accountKey)
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

func registerSlicer(
	app *iris.Application,
	primaryURL string,
	reqNdpt string,
	repNdpt string,
	root string,
	mPlexName string,
) {
	sc := createSliceController(reqNdpt, repNdpt, primaryURL, root, mPlexName)

	app.Get("/{guid:string}/slice/{dim:int32}/{lineno:int32}", sc.get)
}

func Register(
	app *iris.Application,
	primaryURL url.URL,
	accountName,
	accountKey,
	reqNdpt,
	repNdpt string,
) error {
	if err := registerStoreController(app, primaryURL, accountName, accountKey); err != nil {
		return err
	}
	mPlexName := uuid.New().String()
	registerSlicer(app, primaryURL.String(), reqNdpt, repNdpt, accountName, mPlexName)

	return nil
}
