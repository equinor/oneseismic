package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"runtime"

	"github.com/equinor/seismic-cloud-api/api/cmd"
	"github.com/equinor/seismic-cloud-api/api/config"
	l "github.com/equinor/seismic-cloud-api/api/logger"
	jww "github.com/spf13/jwalterweatherman"
)

var AppName string = "sc-api"
var Version string = "v0.0.0"

func getVersionFromGit() string {

	versionCmd := exec.Command("git", "describe", "--always", "--long", "--dirty")
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	run := func(cmd *exec.Cmd) string {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal(err)
		}

		if err := cmd.Start(); err != nil {
			log.Fatal(err)
		}

		ret, err := ioutil.ReadAll(stdout)
		if err != nil {
			log.Fatal(err)
		}
		return string(ret[:len(ret)-1])
	}
	branch := run(branchCmd)
	version := run(versionCmd)
	v := fmt.Sprintf("%s-%s", branch, version)
	return v
}

func init() {
	if Version == "v0.0.0" {
		Version = getVersionFromGit()
	}
	Version += " " + runtime.Version()
	config.SetVersion(Version)

	jww.SetStdoutThreshold(jww.LevelFatal)
	log.SetPrefix("[INFO] ")
	l.Version = Version
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

	cmd.Execute(AppName, Version)

}
