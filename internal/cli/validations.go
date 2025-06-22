// Package cli contains all the command-line interface logic for the application,
// powered by the cobra library. It defines the root command, subcommands,
// and their respective flags.
package cli

import (
	"errors"
	"fmt"
	"net/url"
)

// These validation functions are designed to be used with Cobra's `PreRunE`
// lifecycle hook. Returning an error from `PreRunE` is the idiomatic way to handle
// validation, as it stops command execution and prints the error message automatically.

// validateRootFlags checks the validity of flags defined on the root command,
// which are shared across all subcommands.
func validateRootFlags() error {
	// Base URL is required.
	if rootBaseURL == "" {
		return errors.New("base URL is required")
	}

	// Ensure the URL is parsable before it's used in network requests.
	// This prevents panics or malformed requests in the API client layer.
	if _, err := url.Parse(rootBaseURL); err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	// Model is required.
	if rootModel == "" {
		return errors.New("model is required")
	}

	return nil
}

// validateBenchFlags checks the validity of all flags required by the `bench` command.
//
// This function composes validation by first calling `validateRootFlags`.
// This is a clean, DRY (Don't Repeat Yourself) pattern that ensures shared flags
// are always validated without duplicating logic.
func validateBenchFlags() error {
	// First, validate the shared root flags.
	if err := validateRootFlags(); err != nil {
		return err
	}

	// Then, validate flags specific to the `bench` command.
	if benchPrompt == "" {
		return errors.New("a prompt is required for benchmarking")
	}

	if benchRequestCount <= 0 {
		return errors.New("request count must be greater than 0")
	}

	if benchConcurrency <= 0 {
		return errors.New("concurrency must be greater than 0")
	}

	return nil
}

// validateChatFlags checks the validity of all flags required by the `chat` command.
func validateChatFlags() error {
	// The `chat` command only uses the shared root flags.
	return validateRootFlags()
}
