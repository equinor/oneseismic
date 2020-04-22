package server

import (
	"fmt"

	"github.com/kataras/iris/v12"
)

func registerManifestController(path string, app *iris.Application, a AzureBlobSettings) error {
	sURL, err := newServiceURL(a)
	if err != nil {
		return fmt.Errorf("creating ServiceURL: %w", err)
	}

	mc := &manifestController{ms: sURL}
	app.Get("/", mc.list)

	return nil
}

func Register(app *iris.Application, a AzureBlobSettings) error {
	err := registerManifestController("/", app, a)
	if err != nil {
		return fmt.Errorf("register manifestController: %w", err)
	}

	return nil
}
