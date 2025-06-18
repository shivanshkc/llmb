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
// The channel is automatically closed after all events have been published.
func ReadServerSentEvents(responseBody io.Reader) (<-chan ServerSentEvent, error) {
	// Channel to return.
	eventChan := make(chan ServerSentEvent, 100)

	// All processing happens without blocking. So, the caller gets the channel immediately.
	go func() {
		var wg sync.WaitGroup  // Helps avoid channel write after closure.
		defer close(eventChan) // Channel closes upon return.
		defer wg.Wait()        // Wait for all events to get published.

		// Events will be read line by line.
		reader := bufio.NewReader(responseBody)

		for index := 0; true; index++ {
			value, err := reader.ReadString('\n')
			// Record timestamp even before checking the error.
			sse := ServerSentEvent{Index: index, Value: value, Error: err, Timestamp: time.Now()}
			// Break the loop if the stream has ended.
			if errors.Is(err, io.EOF) {
				break
			}

			// Push to channel without blocking.
			wg.Add(1)
			go func(sse ServerSentEvent) {
				defer wg.Done()
				sanitizeAndPushSSE(sse, eventChan)
			}(sse)
		}
	}()

	return eventChan, nil
}

// sanitizeAndPushSSE sanitizes the given SSE object and, if found valid, pushes it to the given channel.
func sanitizeAndPushSSE(sse ServerSentEvent, eventChan chan<- ServerSentEvent) {
	// If it is an error event, no further sanitization is required.
	if sse.Error != nil {
		eventChan <- sse
		return
	}

	// Sanitize the value and remove the "data:" prefix if present.
	sse.Value = strings.TrimSpace(sse.Value)
	sse.Value = strings.TrimSpace(strings.TrimPrefix(sse.Value, "data:"))
	if sse.Value == "[DONE]" {
		sse.Value = ""
	}

	eventChan <- sse
}
