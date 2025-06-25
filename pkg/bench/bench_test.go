package bench_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shivanshkc/llmb/pkg/bench"
	"github.com/shivanshkc/llmb/pkg/streams"
)

// mockEvent implements the bench.Event interface for testing.
type mockEvent struct {
	index     int
	timestamp time.Time
}

func (m mockEvent) Index() int           { return m.index }
func (m mockEvent) Timestamp() time.Time { return m.timestamp }

// newSuccessfulStreamFunc creates a StreamFunc that successfully produces a
// stream of mock events with a configurable delay.
func newSuccessfulStreamFunc(delay time.Duration, eventCount int) bench.StreamFunc {
	return func(ctx context.Context) (*streams.Stream[bench.Event], error) {
		timer := time.NewTimer(delay)
		defer timer.Stop() // It's good practice to stop the timer.

		select {
		case <-ctx.Done():
			return nil, ctx.Err() // Abort early if context is canceled.
		case <-timer.C:
			// Delay has passed, continue.
		}

		ch := make(chan bench.Event, eventCount)
		go func() {
			defer close(ch)
			for i := 0; i < eventCount; i++ {
				ch <- mockEvent{index: i, timestamp: time.Now()}
			}
		}()

		// Adapt the channel to a stream.
		return streams.New(ch), nil
	}
}

// newFailingStreamFunc creates a StreamFunc that returns an error.
func newFailingStreamFunc(err error) bench.StreamFunc {
	return func(ctx context.Context) (*streams.Stream[bench.Event], error) {
		return nil, err
	}
}

// TestBenchmarkStream verifies the behavior of the main benchmark orchestrator.
func TestBenchmarkStream(t *testing.T) {
	t.Run("Successful Run", func(t *testing.T) {
		// A fast stream func for a simple success case.
		streamFunc := newSuccessfulStreamFunc(10*time.Millisecond, 5)
		requestCount := 10
		concurrency := 3

		results, err := bench.BenchmarkStream(context.Background(), requestCount, concurrency, streamFunc)

		assert.NoError(t, err)
		// A simple sanity check on the results. We can't know the exact values.
		assert.NotZero(t, results.TTFT.Avg, "TTFT Avg should not be zero")
		assert.NotZero(t, results.TT.Max, "Total Time Max should not be zero")
	})

	t.Run("Run with Zero Requests", func(t *testing.T) {
		streamFunc := newSuccessfulStreamFunc(10*time.Millisecond, 5)
		results, err := bench.BenchmarkStream(context.Background(), 0, 5, streamFunc)
		assert.NoError(t, err)
		assert.Equal(t, bench.StreamBenchmarkResults{}, results, "Results should be zero for zero requests")
	})

	t.Run("Immediate Failure on First Request", func(t *testing.T) {
		// This stream func will always fail immediately.
		expectedErr := errors.New("permanent configuration error")
		streamFunc := newFailingStreamFunc(expectedErr)

		results, err := bench.BenchmarkStream(context.Background(), 10, 5, streamFunc)

		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		assert.Equal(t, bench.StreamBenchmarkResults{}, results, "Results should be zero on immediate failure")
	})

	t.Run("Fail-Fast on Worker Error", func(t *testing.T) {
		// Create a stream func that fails on the third attempt.
		var callCount int32
		failingErr := errors.New("simulated API error")
		streamFunc := func(ctx context.Context) (*streams.Stream[bench.Event], error) {
			if atomic.AddInt32(&callCount, 1) == 3 {
				return nil, failingErr
			}
			return newSuccessfulStreamFunc(50*time.Millisecond, 2)(ctx)
		}

		start := time.Now()
		_, err := bench.BenchmarkStream(context.Background(), 10, 5, streamFunc)
		duration := time.Since(start)

		require.Error(t, err)
		assert.Contains(t, err.Error(), failingErr.Error())

		// Crucially, the test should finish quickly, not after all 10 requests would have run.
		// It should take roughly the time of one successful run plus a small margin.
		assert.Less(t, duration, 200*time.Millisecond, "Benchmark should fail fast and not wait for all requests")
	})

	t.Run("Context Cancellation", func(t *testing.T) {
		// Use a slow stream func so cancellation is guaranteed to happen mid-flight.
		streamFunc := newSuccessfulStreamFunc(5*time.Second, 10)

		// Create a context that will be canceled shortly after the test starts.
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, err := bench.BenchmarkStream(ctx, 10, 3, streamFunc)
		duration := time.Since(start)

		require.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded, "Error should be from context cancellation")

		// The test should terminate quickly due to cancellation.
		assert.Less(t, duration, 150*time.Millisecond, "Benchmark should respect context cancellation")
	})
}
