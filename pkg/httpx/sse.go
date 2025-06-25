package httpx

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"
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
func ReadServerSentEvents(ctx context.Context, body io.ReadCloser) <-chan ServerSentEvent {
	eventChan := make(chan ServerSentEvent, 100)

	// producerCtx is a local context for managing the producer's lifecycle.
	// When the producer goroutine finishes (for any reason), it calls cancel(),
	// which signals the context watcher goroutine to exit.
	producerCtx, cancel := context.WithCancel(ctx)

	// Use sync.Once to ensure the body is closed exactly once.
	// This is required because an io.ReadCloser implementation may not be safe for concurrent closing,
	// and we need to attempt closure from two goroutines.
	var closeOnce sync.Once
	closeBodyFunc := func() {
		closeOnce.Do(func() { _ = body.Close() })
	}

	// This goroutine listens for the parent context's cancellation
	// and closes the body to unblock the reader in the following goroutine.
	go func() {
		// Producer finished or parent context was canceled.
		<-producerCtx.Done()
		// Force the reader to unblock.
		closeBodyFunc()
	}()

	// The producer goroutine.
	// It starts a loop to read events and produces them to the returned channel.
	go func() {
		defer close(eventChan) // Close the returned channel once producer is done.
		// This line guarantees that by the time eventChan closes, the body is closed.
		// The context-watcher goroutine above closes the body too, but it can't produce this guarantee.
		// Note that the context-watcher goroutine is still required for correct functioning.
		defer closeBodyFunc()
		defer cancel() // Signal all related goroutines to clean up.

		// For reading events from the body stream.
		reader := bufio.NewReader(body)

		for index := 0; ; index++ {
			line, err := reader.ReadString('\n')
			timestamp := time.Now() // Capture timestamp immediately after read.

			if err != nil {
				// If the error is due to context cancellation, report it.
				if ctx.Err() != nil {
					eventChan <- ServerSentEvent{Index: index, Error: ctx.Err(), Timestamp: timestamp}
					return
				}

				// If the error is not EOF, report it.
				if !errors.Is(err, io.EOF) { // Don't send EOF as a discrete error event.
					eventChan <- ServerSentEvent{Index: index, Error: err, Timestamp: timestamp}
					return
				}

				// The error is EOF. Since the line may contain data, let the switch-case handle it.
			}

			switch value := sanitizeSSE(line); value {
			case "":
				// Continue only if there was no EOF.
				if err == nil {
					continue
				}
			case "[DONE]":
				// Stream signaled completion.
				return
			default:
				eventChan <- ServerSentEvent{Index: index, Value: value, Timestamp: timestamp}
			}

			// If there was an error (which can only be EOF here), end processing.
			if err != nil {
				return
			}
		}
	}()

	return eventChan
}

// sanitizeSSE sanitizes the given SSE value.
//
// IT MUST NOT BE AN EXPENSIVE OPERATION, otherwise the arrival timestamp of the event won't be correct.
func sanitizeSSE(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "data:")
	return strings.TrimSpace(value)
}
