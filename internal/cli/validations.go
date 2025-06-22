package cli

import (
	"net/url"
)

// validateRootFlags validates the flags of the root command.
func validateRootFlags() string {
	// Base URL is required.
	if rootBaseURL == "" {
		return "Base URL is required."
	}

	// Must be a valid URL.
	if _, err := url.Parse(rootBaseURL); err != nil {
		return "Invalid Base URL: " + err.Error()
	}

	// Model is required.
	if rootModel == "" {
		return "Model is required."
	}

	return ""
}

// validateBenchFlags validates the flags of the bench command.
func validateBenchFlags() string {
	// Root command flags are used by the bench command too.
	if message := validateRootFlags(); message != "" {
		return message
	}

	// Prompt is required.
	if benchPrompt == "" {
		return "A prompt is required."
	}

	// At least 1 request required.
	if benchRequestCount <= 0 {
		return "Request count must be greater than 0."
	}

	// At least 1 request should be executed concurrently.
	if benchConcurrency <= 0 {
		return "Concurrency must be greater than 0."
	}

	return ""
}

// validateChatFlags validates the flags of the chat command.
func validateChatFlags() string {
	// Root command flags are used by the chat command too.
	return validateRootFlags()
}
