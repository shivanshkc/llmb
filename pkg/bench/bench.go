package bench

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// StreamBenchmarkResults ...
type StreamBenchmarkResults struct {
	TTFT, TBT, TT Metrics
}

// BenchmarkStream ...
func BenchmarkStream(ctx context.Context, requestCount, concurrency int, sFunc StreamFunc) (StreamBenchmarkResults, error) {
	// Context for managing local goroutines.
	localCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// This will hold all timing information for all streams.
	timingsChan := make(chan timings, requestCount)
	defer close(timingsChan)

	// This channel makes sure only `concurrency` requests execute concurrently at a given time.
	semaphore := make(chan struct{}, concurrency)
	defer close(semaphore)

	// Channel to receive fatal errors. These errors immediately halt all operations.
	errFatalChan := make(chan error, 1)
	defer close(errFatalChan)

	go func() {
		for i := 0; i < requestCount; i++ {
			select {
			// Either a fatal error occurred or the parent context was canceled.
			case <-localCtx.Done():
				return
			// Acquire a spot.
			case semaphore <- struct{}{}:
			}

			// Execute a stream concurrently.
			go func(i int) {
				benchmarkOneStream(localCtx, sFunc, timingsChan, errFatalChan)
				// Release the spot for the next concurrent request.
				<-semaphore
			}(i)
		}
	}()

	timingsArr := make(timingsArray, 0, requestCount)
	// Collect results.
	for i := range requestCount {
		select {
		// Halt all operations upon an error.
		case err := <-errFatalChan:
			cancel()
			return StreamBenchmarkResults{}, err
		case st := <-timingsChan:
			fmt.Printf("[%d/%d] requests complete.\n", i+1, requestCount)
			timingsArr = append(timingsArr, st)
		}
	}

	return StreamBenchmarkResults{
		TTFT: durations(timingsArr.TTFTs()).Metrics(),
		TBT:  durations(timingsArr.TBTs()).Metrics(),
		TT:   durations(timingsArr.TTs()).Metrics(),
	}, nil
}

// benchmarkOneStream executes the given stream once. If there's an error, it is sent to the error channel.
// Otherwise, the result is sent to the timestamps channel.
//
// It is designed to publish results to channels instead of returning them to keep the master function cleaner.
func benchmarkOneStream(ctx context.Context, sFunc StreamFunc, timingsChan chan timings, errFatalChan chan error) {
	// Time at which stream started.
	start := time.Now()
	// Begin the stream.
	eventStream, err := sFunc()
	// Time at which stream ended.
	end := time.Now()

	// Handle fatal error.
	if err != nil {
		errFatalChan <- err
		return
	}

	// Collect all events.
	events, err := eventStream.Exhaust(ctx)
	if err != nil {
		errFatalChan <- err
		return
	}

	// Sort events as their order may have jumbled because of various concurrent operations.
	sort.SliceStable(events, func(i, j int) bool { return events[i].Index() < events[j].Index() })

	// Collect event timestamps.
	eventTimestamps := make([]time.Time, len(events))
	for i, event := range events {
		eventTimestamps[i] = event.Timestamp()
	}

	// Publish results.
	timingsChan <- timings{Start: start, End: end, Events: eventTimestamps}
}
