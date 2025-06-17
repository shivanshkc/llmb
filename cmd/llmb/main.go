package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/shivanshkc/llmb/pkg/api"
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
		fmt.Println("ERROR:", err.Error())
		flag.Usage()
		os.Exit(1)
	}

	// Do something with the flags.
	client := api.NewClient("http://localhost:8080")
	stream, err := client.ChatCompletionStream(context.Background(), *prompt)
	if err != nil {
		fmt.Println("ERROR:", err.Error())
		os.Exit(1)
	}

	for event := range stream {
		if event.Error != nil {
			fmt.Printf("<error>%s</error>", event.Error.Error())
			continue
		}
		fmt.Print(event.Choices[0].Delta.Content)
	}
}

// defineAndParseFlags defines all flags required by the tool,
// calls flag.Parse and returns the flag variable pointers.
func defineAndParseFlags() (*int, *int, *string) {
	// Flag definitions.
	total := flag.Int("t", 1, "Total number of requests (must be > 0)")
	concurrency := flag.Int("c", 1, "Number of concurrent requests (must be > 0)")
	prompt := flag.String("p", "", "The prompt string (must not be empty)")

	flag.Parse()
	return total, concurrency, prompt
}

// validateFlags ...
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
