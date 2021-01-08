package main

import (
	"os"

	"github.com/calvinmclean/automated-garden/garden-app/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
