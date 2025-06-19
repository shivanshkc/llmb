package main

import (
	"os"

	"github.com/shivanshkc/llmb/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
