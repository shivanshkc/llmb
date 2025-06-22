package bench

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// StreamBenchmarkResults holds the final aggregated metrics for a benchmark run.
type StreamBenchmarkResults struct {
	TTFT Metrics // Time To First Token.
	TBT  Metrics // Time Between Tokens.
	TT   Metrics // Total Time (end-to-end).
}

// BenchmarkStream concurrently executes a given stream-producing function and
// aggregates timing metrics. It manages concurrency with a semaphore and ensures
// safe, leak-free shutdown using a context and WaitGroup.
func BenchmarkStream(
	ctx context.Context, requestCount, concurrency int, funk StreamFunc,
) (StreamBenchmarkResults, error) {
	// Run all streams and collect results.
	timingsArr, err := runStreams(ctx, requestCount, concurrency, funk)
	if err != nil {
		return StreamBenchmarkResults{}, fmt.Errorf("error while running streams: %w", err)
	}

	// All runs were successful, calculate and return final metrics.
	return StreamBenchmarkResults{
		TTFT: durations(timingsArr.TTFTs()).Metrics(),
		TBT:  durations(timingsArr.TBTs()).Metrics(),
		TT:   durations(timingsArr.TTs()).Metrics(),
	}, nil
}

// runStreams executes the stream-producing function for a total of `requestCount`
// times with the given level of concurrency, and returns the timings information
// of all streams.
func runStreams(ctx context.Context, requestCount, concurrency int, funk StreamFunc,
) (timingsArray, error) {
	// Use a cancellable context to manage the lifecycle of all workers.
	// This context is passed down to every operation.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Channels required for the operation.
	timingsChan := make(chan timings, requestCount)
	errChan := make(chan error, 1) // Channel to capture the first fatal error.
	semaphore := make(chan struct{}, concurrency)

	// WaitGroup ensures that the channels are not closed before all goroutines finish.
	var wg sync.WaitGroup
	wg.Add(requestCount)

	// Launch a goroutine to spawn workers, preventing the main thread from blocking.
	go func() {
		for i := 0; i < requestCount; i++ {
			select {
			case <-ctx.Done(): // Stop launching new workers if context is canceled.
				wg.Done() // Decrement wg for workers that will never be launched.
				continue
			case semaphore <- struct{}{}:
				// Acquired a concurrency spot.
			}

			go func() {
				defer func() { <-semaphore }() // Release spot when done.
				defer wg.Done()

				if t, err := runOneStream(ctx, funk); err != nil {
					// On error, send it without blocking and cancel all other workers.
					select {
					case errChan <- err:
						cancel() // Signal all other goroutines to stop.
					default:
					}
				} else {
					// This won't block as timingsChan has the size equal to the total request count.
					timingsChan <- t
				}
			}()
		}
	}()

	// Launch a final goroutine to wait for all workers to finish and then
	// close the channels. This signals the main goroutine that all results are in.
	go func() {
		wg.Wait()
		close(timingsChan)
		close(errChan)
	}()

	timingsArr := make(timingsArray, 0, requestCount)
	// This loop now safely terminates when timingsChan is closed.
	for t := range timingsChan {
		timingsArr = append(timingsArr, t)
		fmt.Printf("[%d/%d] requests complete.\n", len(timingsArr), requestCount)
	}

	// After collecting all successful results, check if an error occurred.
	if err := <-errChan; err != nil {
		return nil, fmt.Errorf("a stream worker failed: %w", err)
	}

	// All runs were successful.
	return timingsArr, nil
}

// runOneStream executes the stream-producing function once and returns its
// timings or an error.
func runOneStream(ctx context.Context, funk StreamFunc) (timings, error) {
	// Time at which stream started.
	start := time.Now()
	// Begin the stream.
	eventStream, err := funk(ctx)
	// Handle fatal error.
	if err != nil {
		return timings{}, fmt.Errorf("failed to start stream: %w", err)
	}

	// Collect all events.
	events, err := eventStream.Exhaust(ctx)
	if err != nil {
		return timings{}, fmt.Errorf("failed to exhaust stream: %w", err)
	}

	// Time at which stream ended.
	end := time.Now()

	// Sort events by index to ensure correct TTFT and TBT calculations,
	// as concurrency might jumble collection order.
	sort.SliceStable(events, func(i, j int) bool { return events[i].Index() < events[j].Index() })

	// Collect event timestamps.
	eventTimestamps := make([]time.Time, len(events))
	for i, event := range events {
		eventTimestamps[i] = event.Timestamp()
	}

	return timings{Start: start, End: end, Events: eventTimestamps}, nil
}
