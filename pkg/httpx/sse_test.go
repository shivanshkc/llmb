package httpx_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shivanshkc/llmb/pkg/httpx"
)

// drainChannel collects all events from the SSE channel until it is closed,
// returning them as a slice. It includes a timeout to prevent tests from
// hanging indefinitely on a failure.
func drainChannel(t *testing.T, ch <-chan httpx.ServerSentEvent) []httpx.ServerSentEvent {
	var events []httpx.ServerSentEvent
	timeout := time.After(2 * time.Second) // Safety net for tests.

	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return events // Channel closed, we are done.
			}
			events = append(events, event)
		case <-timeout:
			t.Fatal("Test timed out waiting for event channel to close.")
		}
	}
}

// TestReadServerSentEvents uses a table-driven approach to test various
// scenarios for the ReadServerSentEvents function.
func TestReadServerSentEvents(t *testing.T) {
	// testCase defines the structure for our table-driven tests.
	type testCase struct {
		name          string
		body          io.ReadCloser
		ctx           context.Context
		expectedItems []httpx.ServerSentEvent
	}

	testCases := []testCase{
		{
			name: "Successful Stream with [DONE] Marker",
			body: newMockReadCloser("data: hello\ndata: world\ndata: [DONE]\n"),
			ctx:  context.Background(),
			expectedItems: []httpx.ServerSentEvent{
				{Index: 0, Value: "hello"},
				{Index: 1, Value: "world"},
			},
		},
		{
			name: "Stream Terminating with EOF",
			body: newMockReadCloser("data: first\ndata: second\n"),
			ctx:  context.Background(),
			expectedItems: []httpx.ServerSentEvent{
				{Index: 0, Value: "first"},
				{Index: 1, Value: "second"},
			},
		},
		{
			name: "Stream with Empty Lines and Different Prefixes",
			body: newMockReadCloser(": a comment\ndata: message1\n  data:  message2 \n\ndata: [DONE]"),
			ctx:  context.Background(),
			expectedItems: []httpx.ServerSentEvent{
				{Index: 0, Value: ": a comment"},
				{Index: 1, Value: "message1"},
				{Index: 2, Value: "message2"},
			},
		},
		{
			name: "Context Cancellation on Blocking Read",
			body: newBlockingReadCloser(),
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
				_ = cancel // The timeout will trigger the cancellation.
				return ctx
			}(),
			expectedItems: []httpx.ServerSentEvent{
				{Index: 0, Error: context.DeadlineExceeded},
			},
		},
		{
			name: "Read Error Mid-Stream",
			body: &mockReadCloser{
				reader: io.MultiReader(
					strings.NewReader("data: first event\n"),
					&errorReader{err: errors.New("simulated network error")},
				),
			},
			ctx: context.Background(),
			expectedItems: []httpx.ServerSentEvent{
				{Index: 0, Value: "first event"},
				{Index: 1, Error: errors.New("simulated network error")},
			},
		},
	}

	// --- Test Runner ---
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Execution.
			eventChan := httpx.ReadServerSentEvents(tc.ctx, tc.body)
			events := drainChannel(t, eventChan)

			// Assertions for events.
			require.Equal(t, len(tc.expectedItems), len(events), "Number of received events should match expected.")

			for i, expected := range tc.expectedItems {
				actual := events[i]
				assert.Equal(t, expected.Index, actual.Index, "Event index should match.")
				assert.Equal(t, expected.Value, actual.Value, "Event value should match.")

				if expected.Error != nil {
					assert.Error(t, actual.Error, "Expected an error but got none.")
					if errors.Is(expected.Error, context.DeadlineExceeded) || errors.Is(expected.Error, context.Canceled) {
						assert.ErrorIs(t, actual.Error, expected.Error, "Expected a specific context error.")
					} else {
						assert.Contains(t, actual.Error.Error(), expected.Error.Error(), "Error message should contain expected text.")
					}
				} else {
					assert.NoError(t, actual.Error, "Expected no error but got one.")
				}
			}

			// Assertion for the body being closed.
			// This verifies the function's contract to always close the body.
			switch b := tc.body.(type) {
			case *mockReadCloser:
				assert.True(t, b.isClosed(), "The response body should have been closed.")
			case *blockingReadCloser:
				assert.True(t, b.isClosed(), "The response body should have been closed.")
			default:
				t.Fatal("Unknown mock type used in test case.")
			}
		})
	}
}
