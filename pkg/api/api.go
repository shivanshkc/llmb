package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/shivanshkc/llmb/pkg/utils/httputils"
)

// Client represents an LLM REST API client.
type Client struct {
	baseURL    string
	httpClient *httputils.RetryClient
}

// ChatMessage represents a single message in the LLM chat.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NewClient returns a new Client instance.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &httputils.RetryClient{Client: &http.Client{}},
	}
}

// ChatCompletionStream is a wrapper for the /chat/completions API with stream enabled.
func (c *Client) ChatCompletionStream(
	ctx context.Context, model string, messages []ChatMessage,
) (<-chan ChatCompletionEvent, error) {
	// Form the API endpoint URL.
	endpoint, err := url.JoinPath(c.baseURL, "v1/chat/completions")
	if err != nil {
		return nil, fmt.Errorf("failed to form API endpoint URL: %w", err)
	}

	// Marshal messages to include in the response body.
	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal messages: %w", err)
	}

	// Server-Sent Events are enabled by "stream": true.
	requestBody := []byte(`{ "stream": true, "model": "` + model + `", "messages": ` + string(messagesJSON) + ` }`)
	requestBodyReader := bytes.NewReader(requestBody)

	// Create the HTTP request.
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, requestBodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Body is a JSON.
	httpRequest.Header.Set("Content-Type", "application/json")
	// Make the request retryable.
	httpRequest.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(requestBody)), nil
	}

	// Execute request with retries.
	response, err := c.httpClient.DoRetry(httpRequest, 20, time.Millisecond*50)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	// In case of error, return the status code with the body.
	if response.StatusCode != http.StatusOK {
		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			responseBody = []byte("failed to read response body: " + err.Error())
		}
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", response.StatusCode, string(responseBody))
	}

	// Start reading the events.
	sseChan, err := httputils.ReadServerSentEvents(ctx, response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read server events: %w", err)
	}

	// Channel to which the stream will be piped.
	eventChan := make(chan ChatCompletionEvent, 100)
	// Process events without blocking.
	go func() {
		defer close(eventChan)
		defer func() { _ = response.Body.Close() }()

		for sse := range sseChan {
			eventChan <- convertSSE(sse)
		}
	}()

	return eventChan, nil
}

// convertSSE converts the given Server-Sent Event to a ChatCompletionEvent type.
func convertSSE(sse httputils.ServerSentEvent) ChatCompletionEvent {
	event := ChatCompletionEvent{index: sse.Index, timestamp: sse.Timestamp}

	if sse.Error != nil {
		event.err = fmt.Errorf("failed to read server-sent event: %w", sse.Error)
		return event
	}

	if err := json.Unmarshal([]byte(sse.Value), &event); err != nil {
		event.err = fmt.Errorf("failed to unmarshal server-sent event: %w", err)
		return event
	}

	return event
}
