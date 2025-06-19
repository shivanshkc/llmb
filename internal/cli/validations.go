package cli

import (
	"net/url"
)

// validateBenchFlags validates the flags of the bench command.
func validateBenchFlags() string {
	// Base URL is required.
	if *benchBaseURL == "" {
		return "Base URL is required."
	}

	// Must be a valid URL.
	if _, err := url.Parse(*benchBaseURL); err != nil {
		return "Invalid Base URL: " + err.Error()
	}

	// Model is required.
	if *benchModel == "" {
		return "Model is required."
	}

	// Prompt is required.
	if *benchPrompt == "" {
		return "A prompt is required."
	}

	// At least 1 request required.
	if *benchRequestCount <= 0 {
		return "Request count must be greater than 0."
	}

	// At least 1 request should be executed concurrently.
	if *benchConcurrency <= 0 {
		return "Concurrency must be greater than 0."
	}

	return ""
}

// validateChatFlags validates the flags of the chat command.
func validateChatFlags() string {
	// Base URL is required.
	if *chatBaseURL == "" {
		return "Base URL is required."
	}

	// Must be a valid URL.
	if _, err := url.Parse(*chatBaseURL); err != nil {
		return "Invalid Base URL: " + err.Error()
	}

	// Model is required.
	if *chatModel == "" {
		return "Model is required."
	}

	return ""
}
