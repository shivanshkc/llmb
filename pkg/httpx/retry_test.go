package httpx_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shivanshkc/llmb/pkg/httpx"
)

// mockRoundTripper is a mock implementation of http.RoundTripper.
// It allows us to simulate different network responses (success, failure)
// for each attempt, without making real network calls.
type mockRoundTripper struct {
	// A slice of functions, where each function represents the outcome
	// of one `Do` attempt.
	responses []func(*http.Request) (*http.Response, error)
	// attempt tracks the current call number.
	attempt int
}

// RoundTrip satisfies the http.RoundTripper interface. It invokes the response
// function corresponding to the current attempt number.
func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Ensure we don't go out of bounds. If we do, it likely means
	// the retry logic is attempting more calls than we've configured our mock for.
	if m.attempt >= len(m.responses) {
		return nil, errors.New("mockRoundTripper: too many attempts")
	}

	// Get the response function for the current attempt.
	responseFunc := m.responses[m.attempt]
	m.attempt++ // Increment for the next call.
	return responseFunc(req)
}

// TestRetryClient_DoRetry verifies the core logic of the DoRetry method.
// It uses a table-driven approach with a mockRoundTripper to simulate
// various network conditions and client behaviors.
func TestRetryClient_DoRetry(t *testing.T) {
	// A standard request body to be used in tests.
	requestBody := `{"message":"hello"}`

	// testCase defines the structure for our table-driven tests.
	type testCase struct {
		name          string
		maxAttempts   int
		delay         time.Duration
		roundTripper  http.RoundTripper
		ctx           context.Context
		expectSuccess bool
		expectedErr   string
	}

	// --- Test Cases ---
	testCases := []testCase{
		{
			name:        "Success on First Attempt",
			maxAttempts: 3,
			delay:       10 * time.Millisecond,
			roundTripper: &mockRoundTripper{
				responses: []func(*http.Request) (*http.Response, error){
					// Attempt 1: Success.
					func(r *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader("success")),
						}, nil
					},
				},
			},
			ctx:           context.Background(),
			expectSuccess: true,
		},
		{
			name:        "Success on Second Attempt",
			maxAttempts: 3,
			delay:       10 * time.Millisecond,
			roundTripper: &mockRoundTripper{
				responses: []func(*http.Request) (*http.Response, error){
					// Attempt 1: Failure.
					func(r *http.Request) (*http.Response, error) {
						return nil, errors.New("network error")
					},
					// Attempt 2: Success.
					func(r *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader("success")),
						}, nil
					},
				},
			},
			ctx:           context.Background(),
			expectSuccess: true,
		},
		{
			name:        "Failure After All Retries Exhausted",
			maxAttempts: 3,
			delay:       10 * time.Millisecond,
			roundTripper: &mockRoundTripper{
				responses: []func(*http.Request) (*http.Response, error){
					// All three attempts fail.
					func(r *http.Request) (*http.Response, error) { return nil, errors.New("attempt 1 failed") },
					func(r *http.Request) (*http.Response, error) { return nil, errors.New("attempt 2 failed") },
					func(r *http.Request) (*http.Response, error) { return nil, errors.New("attempt 3 failed") },
				},
			},
			ctx:           context.Background(),
			expectSuccess: false,
			expectedErr:   "all 3 attempts failed",
		},
		{
			name:        "Context Canceled During Retry Delay",
			maxAttempts: 3,
			// Use a longer delay to ensure the context cancellation happens first.
			delay: 100 * time.Millisecond,
			roundTripper: &mockRoundTripper{
				responses: []func(*http.Request) (*http.Response, error){
					// The first attempt must fail to trigger the delay.
					func(r *http.Request) (*http.Response, error) { return nil, errors.New("transient error") },
				},
			},
			// This context is set to cancel itself after a short time.
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
				// We don't need to call cancel ourselves. The timeout will trigger it.
				_ = cancel
				return ctx
			}(),
			expectSuccess: false,
			expectedErr:   "context deadline exceeded",
		},
	}

	// --- Test Runner ---
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup: Create a RetryClient with our mock transport.
			client := &httpx.RetryClient{
				Client: &http.Client{
					Transport: tc.roundTripper,
				},
			}

			// Setup: Create a request with a rewindable body.
			req := httptest.NewRequestWithContext(tc.ctx, http.MethodPost, "https://example.com/test", strings.NewReader(requestBody))
			req.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader([]byte(requestBody))), nil
			}
			*req = *req.WithContext(tc.ctx)

			// Execution: Call the method under test.
			resp, err := client.DoRetry(req, tc.maxAttempts, tc.delay)

			// Assertion.
			if tc.expectSuccess {
				assert.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				// It is the caller's responsibility to close the body on success.
				_ = resp.Body.Close()
			} else {
				assert.Error(t, err)
				assert.Nil(t, resp)
				assert.Contains(t, err.Error(), tc.expectedErr)
			}
		})
	}
}

// TestRetryClient_DoRetry_NoGetBody validates that the function correctly
// rejects requests that cannot be retried because they lack a GetBody method.
func TestRetryClient_DoRetry_NoGetBody(t *testing.T) {
	// Setup: A client with a default transport.
	client := &httpx.RetryClient{Client: http.DefaultClient}
	// Setup: A request *without* GetBody set.
	req := httptest.NewRequest(http.MethodPost, "http://example.com/test", nil)

	// Execution & Assertion.
	resp, err := client.DoRetry(req, 3, 10*time.Millisecond)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "GetBody function must be set")
}
