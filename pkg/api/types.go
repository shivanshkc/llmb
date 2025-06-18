package api

import (
	"time"
)

// ChatCompletionEvent represents a single event from the Chat-Completion API response stream.
type ChatCompletionEvent struct {
	Choices []ChatCompletionChoice `json:"choices"`

	Created           int    `json:"created"`
	Id                string `json:"id"`
	Model             string `json:"model"`
	SystemFingerprint string `json:"system_fingerprint"`
	Object            string `json:"object"`

	// index can be used to process events in the correct order.
	index int
	// timestamp is the local timestamp of event reception.
	// It is not received from the API.
	timestamp time.Time
	// Error in processing the event.
	err error
}

func (cce ChatCompletionEvent) Index() int           { return cce.index }
func (cce ChatCompletionEvent) Timestamp() time.Time { return cce.timestamp }

type ChatCompletionChoice struct {
	Delta ChatCompletionDelta `json:"delta"`

	FinishReason any `json:"finish_reason"`
	Index        int `json:"index"`
}

type ChatCompletionDelta struct {
	Content string `json:"content"`
}
