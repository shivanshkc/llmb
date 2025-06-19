package bench

import (
	"fmt"
	"sort"
	"time"
)

// StreamBenchmarkResults ...
type StreamBenchmarkResults struct {
	TTFT, TBT, TT Metrics
}

// BenchmarkStream ...
func BenchmarkStream(requestCount, concurrency int, sFunc StreamFunc) (StreamBenchmarkResults, error) {
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
			// Wait for turn.
			semaphore <- struct{}{}
			// Execute a stream concurrently.
			go func(i int) {
				// Release lock for the next concurrent request.
				defer func() { <-semaphore }()
				benchmarkOneStream(sFunc, timingsChan, errFatalChan)
			}(i)
		}
	}()

	timingsArr := make(timingsArray, 0, requestCount)
	// Collect results.
	for i := range requestCount {
		select {
		// The loop above will not be able to catch the error of the final stream.
		case err := <-errFatalChan:
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
func benchmarkOneStream(sFunc StreamFunc, timingsChan chan timings, errFatalChan chan error) {
	// Time at which stream started.
	start := time.Now()
	// Collect all events from the stream.
	events, err := sFunc.Consume()
	// Time at which stream ended.
	end := time.Now()

	// Handle fatal error.
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
