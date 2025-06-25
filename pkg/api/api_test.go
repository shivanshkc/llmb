package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shivanshkc/llmb/pkg/httpx"
)

// mockRoundTripper is a mock implementation of http.RoundTripper.
// It allows us to simulate different HTTP responses for each attempt,
// controlling the status code, body, and errors without making real network calls.
type mockRoundTripper struct {
	responseFunc func(*http.Request) (*http.Response, error)
}

// RoundTrip satisfies the http.RoundTripper interface. It invokes the mock's
// configured response function.
func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.responseFunc(req)
}

// TestClient_ChatCompletionStream uses a table-driven approach to test the
// main API client method across various scenarios.
func TestClient_ChatCompletionStream(t *testing.T) {
	// --- Test Case Definitions ---
	type testCase struct {
		name string
		// baseURL for the API client.
		baseURL string
		// roundTripper is the mock HTTP transport that simulates server responses.
		roundTripper http.RoundTripper
		// ctx is the context to be passed to the function under test.
		ctx context.Context
		// expectedEvents are the successfully parsed events we expect from the stream.
		// We only check the delta content for simplicity.
		expectedDeltas []string
		// expectedErr is the error we expect the function itself to return.
		expectedErr error
		// expectedStreamErr indicates if we expect an error within the stream itself.
		expectedStreamErr bool
	}

	testCases := []testCase{
		{
			name:    "Successful Stream",
			baseURL: "http://localhost:8080",
			roundTripper: &mockRoundTripper{
				responseFunc: func(r *http.Request) (*http.Response, error) {
					body := `
data: {"choices":[{"delta":{"content":"Hello"}}]}
data: {"choices":[{"delta":{"content":" world"}}]}
data: [DONE]`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(body)),
					}, nil
				},
			},
			ctx:            context.Background(),
			expectedDeltas: []string{"Hello", " world"},
			expectedErr:    nil,
		},
		{
			name:    "API Error with Non-200 Status",
			baseURL: "http://localhost:8080",
			roundTripper: &mockRoundTripper{
				responseFunc: func(r *http.Request) (*http.Response, error) {
					body := `{"error": "bad request"}`
					return &http.Response{
						StatusCode: http.StatusBadRequest,
						Body:       io.NopCloser(strings.NewReader(body)),
					}, nil
				},
			},
			ctx:         context.Background(),
			expectedErr: errors.New("unexpected status code: 400"),
		},
		{
			name:    "Network Error from HTTP Client",
			baseURL: "http://localhost:8080",
			roundTripper: &mockRoundTripper{
				responseFunc: func(r *http.Request) (*http.Response, error) {
					return nil, errors.New("connection refused")
				},
			},
			ctx:         context.Background(),
			expectedErr: errors.New("failed to execute HTTP request"),
		},
		{
			name:    "Malformed Base URL",
			baseURL: "https://invalid-url-\x7f.com",
			roundTripper: &mockRoundTripper{
				responseFunc: func(r *http.Request) (*http.Response, error) { return nil, nil },
			},
			ctx:         context.Background(),
			expectedErr: errors.New("failed to form API endpoint URL"),
		},
		{
			name:    "Stream with Malformed JSON Event",
			baseURL: "http://localhost:8080",
			roundTripper: &mockRoundTripper{
				responseFunc: func(r *http.Request) (*http.Response, error) {
					body := `
data: {"choices":[{"delta":{"content":"Good"}}]}
data: {"choices":` // Malformed JSON
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(body)),
					}, nil
				},
			},
			ctx:               context.Background(),
			expectedDeltas:    []string{"Good"},
			expectedErr:       nil,
			expectedStreamErr: true, // Expect an error during stream consumption.
		},
	}

	// --- Test Runner ---
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup: Create a client and inject our mock transport directly into
			// the unexported httpClient field.
			client := NewClient(tc.baseURL)
			client.httpClient = &httpx.RetryClient{
				Client: &http.Client{Transport: tc.roundTripper},
			}

			// Execution: Call the method under test.
			stream, err := client.ChatCompletionStream(tc.ctx, "test-model", nil)

			// Assertion for the function's direct return value.
			if tc.expectedErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr.Error())
				assert.Nil(t, stream)
				return // Test is complete if an immediate error was expected.
			}

			require.NoError(t, err)
			require.NotNil(t, stream)

			// Execution & Assertion for the stream's content.
			events, exhaustErr := stream.Exhaust(context.Background())
			assert.NoError(t, exhaustErr, "Draining the stream should not cause a primary error")

			// Collect deltas and check for processing errors within the events.
			var deltas []string
			var streamErr error
			for _, event := range events {
				// The testable Error() method is added to the ChatCompletionEvent struct.
				if event.err != nil {
					streamErr = event.err
				}
				if len(event.Choices) > 0 {
					deltas = append(deltas, event.Choices[0].Delta.Content)
				}
			}

			assert.Equal(t, tc.expectedDeltas, deltas, "The collected deltas should match the expected deltas.")
			if tc.expectedStreamErr {
				assert.Error(t, streamErr, "Expected a processing error within the stream.")
			} else {
				assert.NoError(t, streamErr, "Did not expect a processing error within the stream.")
			}
		})
	}
}

// Test_convertSSE verifies the logic of the SSE-to-ChatCompletionEvent converter.
func Test_convertSSE(t *testing.T) {
	t.Run("Valid SSE", func(t *testing.T) {
		sse := httpx.ServerSentEvent{Value: `{"choices":[{"delta":{"content":" test "}}]}`}
		event := convertSSE(sse)
		assert.NoError(t, event.err)
		require.Len(t, event.Choices, 1)
		assert.Equal(t, " test ", event.Choices[0].Delta.Content)
	})

	t.Run("SSE with Error", func(t *testing.T) {
		expectedErr := errors.New("read error")
		sse := httpx.ServerSentEvent{Error: expectedErr}
		event := convertSSE(sse)
		assert.ErrorIs(t, event.err, expectedErr)
	})

	t.Run("SSE with Malformed JSON", func(t *testing.T) {
		sse := httpx.ServerSentEvent{Value: `{invalid-json}`}
		event := convertSSE(sse)
		assert.Error(t, event.err)
		assert.Contains(t, event.err.Error(), "failed to unmarshal")
	})
}
