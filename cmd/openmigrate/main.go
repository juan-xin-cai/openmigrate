package main

import (
	"os"

	"github.com/openmigrate/openmigrate/internal/cli"
)

func main() {
	os.Exit(cli.Execute(cli.DefaultStreams()))
}
