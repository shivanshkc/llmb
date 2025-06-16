package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

// Input errors.
var (
	errTotal       = errors.New("total number of requests (-t) must be greater than zero")
	errConcurrency = errors.New("number of concurrent requests (-c) must be greater than zero")
	errPrompt      = errors.New("prompt (-p) must not be empty")
)

func main() {
	// Inputs.
	total, concurrency, prompt := defineAndParseFlags()
	if err := validateFlags(total, concurrency, prompt); err != nil {
		exitWithError(err)
	}

	// Do something with the flags.
}

func defineAndParseFlags() (*int, *int, *string) {
	// Flag definitions.
	total := flag.Int("t", 1, "Total number of requests (must be > 0)")
	concurrency := flag.Int("c", 1, "Number of concurrent requests (must be > 0)")
	prompt := flag.String("p", "", "The prompt string (must not be empty)")

	flag.Parse()
	return total, concurrency, prompt
}

func validateFlags(total, concurrency *int, prompt *string) error {
	// At least 1 request required.
	if *total <= 0 {
		return errTotal
	}
	// At least 1 request should be executed concurrently.
	if *concurrency <= 0 {
		return errConcurrency
	}
	// Check if -p is a non-empty string.
	if *prompt == "" {
		return errPrompt
	}

	return nil
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, "Error: "+err.Error())
	flag.Usage()
	os.Exit(1)
}
