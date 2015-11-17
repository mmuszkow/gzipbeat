package main

import (
	"os"

	"github.com/elastic/libbeat/beat"
	"github.com/elastic/libbeat/logp"
)

var Version = "alpha1"
var Name = "gzipbeat"

func main() {
	gb := &Gzipbeat{}

	b := beat.NewBeat(Name, Version, gb)

	b.CommandLineSetup()

	b.LoadConfig()
	err := gb.Config(b)
	if err != nil {
		logp.Critical("Config error: %v", err)
		os.Exit(1)
	}

	b.Run()
}
