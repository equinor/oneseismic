// +build !prod

package main

import (
	"io/ioutil"
	"log"
	"os/exec"

	"github.com/equinor/seismic-cloud/api/config"
)

func init() {

	cmd := exec.Command("git", "describe", "--always", "--long", "--dirty")
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
	config.SetDevVersion("DEV", "BUILD", string(gVer))
}
