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

	// Create a map for marshalling. This makes the JSON formation injection-proof.
	requestBodyMap := map[string]any{"stream": true, "model": model, "messages": messages}
	requestBody, err := json.Marshal(requestBodyMap)
	if err != nil {
		return nil, fmt.Errorf("failed to form API request body: %w", err)
	}

	// Create the HTTP request.
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Body is a JSON.
	request.Header.Set("Content-Type", "application/json")
	// Make the request retryable.
	request.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(requestBody)), nil
	}

	// Execute request with retries.
	response, err := c.httpClient.DoRetry(request, 20, time.Millisecond*50)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	// In case of error, return the status code with the body.
	if response.StatusCode != http.StatusOK {
		defer func() { _ = response.Body.Close() }()
		// Include body in the error for debug purposes.
		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			responseBody = []byte("failed to read response body: " + err.Error())
		}
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", response.StatusCode, string(responseBody))
	}

	// Start reading the events.
	sseChan := httputils.ReadServerSentEvents(ctx, response.Body)
	// Channel to which the stream will be piped.
	eventChan := make(chan ChatCompletionEvent, 100)

	// Process events without blocking.
	go func() {
		defer close(eventChan)
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
