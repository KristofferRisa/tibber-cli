package main

import (
	"os"

	"github.com/kristofferrisa/tibber-cli/internal/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
