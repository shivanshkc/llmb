package httputils

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RetryClient is an extension of the standard HTTP client.
// It provides a DoRetry method that keeps executing the given request until it succeeds.
// Here, success means the `Do` method does not return a transient error.
type RetryClient struct {
	*http.Client
}

// DoRetry internally calls the `Do` method of the standard HTTP client on the given request.
// If `Do` returns an error, the operation is retried up to maxAttempts times.
func (rc *RetryClient) DoRetry(req *http.Request, maxAttempts int, delay time.Duration) (*http.Response, error) {
	// Request must be rewindable for retries.
	if req.GetBody == nil {
		return nil, fmt.Errorf("GetBody function must be set on the request for retrying")
	}

	// This will hold the error that will be returned of all retries fail.
	var errFinal error

	for i := 0; i < maxAttempts; i++ {
		// Create a fresh body for this attempt.
		bodyReader, err := req.GetBody()
		if err != nil {
			return nil, fmt.Errorf("error in the GetBody call: %w", err)
		}
		req.Body = bodyReader

		// Attempt the request.
		response, err := rc.Do(req)
		if err == nil {
			// Success! The caller is now responsible for closing the response body.
			return response, nil
		}

		// Record the error. If this is the final retry, this error will be returned.
		errFinal = err
		// Don't execute the waiting code if this is the last iteration.
		if i == maxAttempts-1 {
			break
		}

		// Timer to wait before next retry.
		timer := time.NewTimer(delay)
		// Wait before the next retry while respecting the request's context.
		select {
		case <-req.Context().Done():
			timer.Stop()                    // Cleanup the timer. `time.After` does not allow this optimization.
			return nil, req.Context().Err() // Return the context's error.
		case <-timer.C:
			// Continue to the next attempt.
		}
	}

	return nil, fmt.Errorf("all %d attempts failed, last error: %w", maxAttempts, errFinal)
}

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
	var closeOnce sync.Once
	closeBody := func() { closeOnce.Do(func() { _ = body.Close() }) }

	// This goroutine listens for the parent context's cancellation
	// and closes the body to unblock the reader.
	go func() {
		// Producer finished or parent context was canceled.
		<-producerCtx.Done()
		// Force the reader to unblock.
		closeBody()
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
