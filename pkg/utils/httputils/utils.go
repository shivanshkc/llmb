package httputils

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

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

// ReadServerSentEvents reads the given response body assuming it is a stream of Server-Sent events
// and returns a channel for the caller to consume the events.
//
// The channel is automatically closed after all events have been published.
func ReadServerSentEvents(responseBody io.Reader) (<-chan ServerSentEvent, error) {
	// Channel to return.
	eventChan := make(chan ServerSentEvent)

	// All processing happens without blocking. So, the caller gets the channel immediately.
	go func() {
		// Required to make sure that the channel is closed only after all events have been published.
		var wg sync.WaitGroup

		// Cleanups.
		defer func() {
			// Wait for all events to get published.
			wg.Wait()
			// Channel closes upon return.
			close(eventChan)
		}()

		// Events will be read line by line.
		reader := bufio.NewReader(responseBody)

		for index := 0; true; index++ {
			value, err := reader.ReadString('\n')
			// Record reception timestamp as quickly as possible.
			timestamp := time.Now()

			// Break the loop if the stream has ended.
			if errors.Is(err, io.EOF) {
				break
			}

			wg.Add(1)
			// Goroutine because this operation should not block the loop.
			go func(index int, value string, err error, timestamp time.Time) {
				defer wg.Done()
				parsed := parseServerSentEvent(index, value, err, timestamp)
				if parsed.Error != nil || parsed.Value != nil {
					eventChan <- parsed
				}
			}(index, value, err, timestamp)
		}
	}()

	return eventChan, nil
}

// ServerSentEvent represents a single event sent by the server.
type ServerSentEvent struct {
	Index     int
	Value     []byte
	Error     error
	Timestamp time.Time
}

// parseServerSentEvent is a "smart" constructor function for the ServerSentEvent type.
//
// It has two key behaviours:
//  1. If the provided error is not nil, the value is ignored.
//  2. If the value has a "data: " prefix, it is trimmed.
func parseServerSentEvent(index int, value string, err error, timestamp time.Time) ServerSentEvent {
	event := ServerSentEvent{Index: index, Timestamp: timestamp}
	if err != nil {
		event.Error = fmt.Errorf("failed to read event: %w", err)
		return event
	}

	// Sanitize the value and remove the "data:" prefix if present.
	value = strings.TrimSpace(value)
	value = strings.TrimSpace(strings.TrimPrefix(value, "data:"))
	if value == "[DONE]" || value == "" {
		return ServerSentEvent{}
	}

	event.Value = []byte(value)
	return event
}
