package streams_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shivanshkc/llmb/pkg/streams"
)

// TestStream_NextContext is the primary integration test for the Stream.
// It uses a table-driven approach to verify the core functionality across
// several key scenarios, including successful iteration, chaining maps, and
// context cancellation.
func TestStream_NextContext(t *testing.T) {
	// testCase defines the structure for our table-driven tests.
	type testCase struct {
		name          string
		setupStream   func() *streams.Stream[string]
		ctx           context.Context
		expectedItems []string
		expectedErr   error
	}

	// --- Test Cases ---
	testCases := []testCase{
		{
			name: "Successful Iteration of a Simple Stream",
			setupStream: func() *streams.Stream[string] {
				// Create a simple stream of 3 integers and map them to strings.
				ch := make(chan int, 3)
				ch <- 1
				ch <- 2
				ch <- 3
				close(ch)
				intStream := streams.New(ch)
				return streams.Map(intStream, func(i int) string {
					return fmt.Sprintf("item-%d", i)
				})
			},
			ctx:           context.Background(),
			expectedItems: []string{"item-1", "item-2", "item-3"},
			expectedErr:   nil,
		},
		{
			name: "Successful Iteration of a Chained Stream",
			setupStream: func() *streams.Stream[string] {
				// A more complex pipeline: int -> float64 -> string
				ch := make(chan int, 2)
				ch <- 10
				ch <- 20
				close(ch)
				intStream := streams.New(ch)
				floatStream := streams.Map(intStream, func(i int) float64 {
					return float64(i) * 1.5
				})
				return streams.Map(floatStream, func(f float64) string {
					return fmt.Sprintf("%.2f", f)
				})
			},
			ctx:           context.Background(),
			expectedItems: []string{"15.00", "30.00"},
			expectedErr:   nil,
		},
		{
			name: "Iteration of an Empty Stream",
			setupStream: func() *streams.Stream[string] {
				// The source channel is created and immediately closed.
				ch := make(chan int)
				close(ch)
				intStream := streams.New(ch)
				return streams.Map(intStream, func(i int) string { return "should-not-happen" })
			},
			ctx:           context.Background(),
			expectedItems: nil, // Expect no items.
			expectedErr:   nil,
		},
		{
			name: "Context Cancellation on a Blocking Stream",
			setupStream: func() *streams.Stream[string] {
				// CRITICAL TEST: The source channel is never written to, forcing
				// the stream to block indefinitely without cancellation.
				ch := make(chan int)
				intStream := streams.New(ch)
				return streams.Map(intStream, func(i int) string { return "should-not-happen" })
			},
			// The context will time out, unblocking the NextContext call.
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
				// We don't need to call cancel, the timeout will do it.
				_ = cancel
				return ctx
			}(),
			expectedItems: nil,
			expectedErr:   context.DeadlineExceeded,
		},
	}

	// --- Test Runner ---
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup.
			stream := tc.setupStream()
			require.NotNil(t, stream)

			var items []string
			var finalErr error

			// Execution: Consume the stream.
			for {
				item, ok, err := stream.NextContext(tc.ctx)
				if err != nil {
					finalErr = err
					break
				}
				if !ok {
					break
				}
				items = append(items, item)
			}

			// Assertion.
			assert.Equal(t, tc.expectedItems, items, "Collected items should match the expected items.")
			if tc.expectedErr != nil {
				assert.ErrorIs(t, finalErr, tc.expectedErr, "Final error should match the expected error.")
			} else {
				assert.NoError(t, finalErr, "Expected no error, but got one.")
			}
		})
	}
}

// TestStream_Exhaust verifies the behavior of the convenience Exhaust method.
func TestStream_Exhaust(t *testing.T) {
	t.Run("Successful Exhaust", func(t *testing.T) {
		// Setup.
		ch := make(chan int, 2)
		ch <- 100
		ch <- 200
		close(ch)
		stream := streams.New(ch)

		// Execution.
		items, err := stream.Exhaust(context.Background())

		// Assertion.
		assert.NoError(t, err)
		assert.Equal(t, []int{100, 200}, items)
	})

	t.Run("Exhaust with Context Cancellation", func(t *testing.T) {
		// Setup.
		blockingChan := make(chan int) // This channel will block.
		stream := streams.New(blockingChan)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		// Execution.
		items, err := stream.Exhaust(ctx)

		// Assertion.
		assert.Nil(t, items, "Items should be nil on error.")
		assert.ErrorIs(t, err, context.DeadlineExceeded, "Error should be context.DeadlineExceeded.")
	})
}

// TestStream_Next tests the non-cancellable convenience method.
func TestStream_Next(t *testing.T) {
	// Setup
	ch := make(chan string, 1)
	ch <- "hello"
	close(ch)
	stream := streams.New(ch)

	// Execution & Assertion.
	item, ok := stream.Next()
	assert.True(t, ok)
	assert.Equal(t, "hello", item)

	// Second call should indicate the stream is exhausted.
	item, ok = stream.Next()
	assert.False(t, ok)
	assert.Equal(t, "", item, "Exhausted stream should return zero value.")
}
