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
// Here, success does not mean a 2xx. It means that the `Do` method does not return an error.
type RetryClient struct {
	*http.Client
}

// DoRetry internally calls `net.http.Client.Do` on the given request.
// If `Do` returns an error, the operation is retried. This method requires the request to have the GetBody method.
func (rc *RetryClient) DoRetry(req *http.Request, retries int, delay time.Duration) (*http.Response, error) {
	// Request should be rewindable.
	if req.GetBody == nil {
		return nil, fmt.Errorf("GetBody function should be set for retrying")
	}

	// In case all retries are exhausted, this error will be returned.
	var errFinal error

	for i := 0; i < retries; i++ {
		// Body needs to be reassigned upon every retry.
		bodyReader, err := req.GetBody()
		if err != nil {
			return nil, fmt.Errorf("failed to get request body from GetBody: %w", err)
		}
		// Reassign body.
		req.Body = bodyReader

		// Execute request. If no error, return early.
		response, err := rc.Do(req)
		if err == nil {
			return response, nil
		}

		// Record error for returning.
		errFinal = err

		// Wait before retrying.
		time.Sleep(delay)
	}

	return nil, fmt.Errorf("retries exhausted, error: %w", errFinal)
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
