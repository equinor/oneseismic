// +build prod

package main

import (
	"io/ioutil"
	"log"
	"os/exec"
	"strings"

	"github.com/equinor/seismic-cloud/api/config"
)

var MajVer, MinVer, PatchVer string

func init() {

	if len(MajVer) > 0 {
		config.SetDevVersion(MajVer, MinVer, PatchVer)
		return
	}
	cmd := exec.Command("git", "describe", "--tags", "--long")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	gVer, err := ioutil.ReadAll(stdout)
	if err != nil {
		log.Fatal(err)
	}

	spl := strings.Split(string(gVer), ".")
	if len(spl) >= 3 {
		config.SetDevVersion(spl[0], spl[1], strings.Join(spl[2:], "."))
	} else {
		config.SetDevVersion("PROD", "BUILD", string(gVer))
	}

}
