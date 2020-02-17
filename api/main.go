package main

import (
	"fmt"
	"log"

	"github.com/equinor/seismic-cloud/api/cmd"
	l "github.com/equinor/seismic-cloud/api/logger"
	jww "github.com/spf13/jwalterweatherman"
)

var AppName = "sc-api"

func init() {
	jww.SetStdoutThreshold(jww.LevelFatal)
	log.SetPrefix("[INFO] ")
	l.AddLoggerSource("main.log", log.SetOutput)
	l.AddLoggerSource("setup.log", jww.SetLogOutput)
}

//@title Seismic Cloud Api
//@description The Seismic Cloud Api
//@license.name proprietary
//@contact.name Equinor
//@securityDefinitions.apikey ApiKeyAuth
//@in header
//@name Authorization

//@tag.name manifest
//@tag.description Operations for manifests
//@tag.name surface
//@tag.description Operations for surfaces
//@tag.name stitch
//@tag.description Stitch together cube data
func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in main", r)
		}
		l.Wait()

	}()

	cmd.Execute(AppName)

}
