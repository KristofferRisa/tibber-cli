package main

import (
	"os"

	"github.com/kristofferrisa/powerctl-cli/internal/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
