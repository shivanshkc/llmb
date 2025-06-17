package api

import (
	"time"
)

type ChatCompletionEvent struct {
	Choices []ChatCompletionChoice `json:"choices"`

	Created           int    `json:"created"`
	Id                string `json:"id"`
	Model             string `json:"model"`
	SystemFingerprint string `json:"system_fingerprint"`
	Object            string `json:"object"`

	// Received is the local timestamp of event reception.
	// It is not received from the API.
	Received time.Time `json:"-"`
	// Error in processing the event.
	Error error `json:"-"`
}

type ChatCompletionChoice struct {
	Delta ChatCompletionDelta `json:"delta"`

	FinishReason any `json:"finish_reason"`
	Index        int `json:"index"`
}

type ChatCompletionDelta struct {
	Content string `json:"content"`
}
