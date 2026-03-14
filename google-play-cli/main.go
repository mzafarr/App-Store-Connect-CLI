package main

import (
	"os"

	"github.com/rudrankriyam/google-play-cli/cmd"
)

var version = "dev"

func main() {
	os.Exit(cmd.Run(os.Args[1:], version))
}
