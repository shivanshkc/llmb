package httpx

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/shivanshkc/llmb/pkg/streams"
)

// ServerSentEvent represents a single event sent by the server.
type ServerSentEvent struct {
	Index     int
	Value     string
	Error     error
	Timestamp time.Time
}

// ReadServerSentEvents reads the given response body assuming it is a stream of Server-Sent events
// and returns a channel for the caller to consume the events.
//
// It takes ownership of the response body and guarantees it will be closed.
func ReadServerSentEvents(ctx context.Context, body io.ReadCloser) streams.Stream[ServerSentEvent] {
	eventChan := make(chan ServerSentEvent, 100)

	// producerCtx is a local context for managing the producer's lifecycle.
	// When the producer goroutine finishes (for any reason), it calls cancel(),
	// which signals the context watcher goroutine to exit.
	producerCtx, cancel := context.WithCancel(ctx)

	// This goroutine listens for the parent context's cancellation
	// and closes the body to unblock the reader.
	go func() {
		// Producer finished or parent context was canceled.
		<-producerCtx.Done()
		// Force the reader to unblock.
		_ = body.Close()
	}()

	// The producer goroutine.
	// It starts a loop to read events and produces them to the returned channel.
	go func() {
		defer close(eventChan) // Close the returned channel once producer is done.
		defer cancel()         // Signal all related goroutines to clean up.

		// For reading events from the body stream.
		reader := bufio.NewReader(body)

		for index := 0; ; index++ {
			line, err := reader.ReadString('\n')
			timestamp := time.Now() // Capture timestamp immediately after read.

			if err != nil {
				// If the error is due to context cancellation, report the context error.
				if ctx.Err() != nil {
					err = ctx.Err()
				}
				// Send the final error and exit.
				if !errors.Is(err, io.EOF) { // Don't send EOF as a discrete error event.
					eventChan <- ServerSentEvent{Index: index, Error: err, Timestamp: timestamp}
				}
				return
			}

			switch value := sanitizeSSE(line); value {
			case "":
				// SSE spec says to ignore empty lines.
				continue
			case "[DONE]":
				// Stream signaled completion.
				return
			default:
				eventChan <- ServerSentEvent{Index: index, Value: value, Timestamp: timestamp}
			}
		}
	}()

	return streams.New(eventChan)
}

// sanitizeSSE sanitizes the given SSE value.
//
// IT MUST NOT BE AN EXPENSIVE OPERATION, otherwise the arrival timestamp of the event won't be correct.
func sanitizeSSE(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "data:")
	return strings.TrimSpace(value)
}
