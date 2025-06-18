package bench

import (
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"github.com/shivanshkc/llmb/pkg/api"
)

// StreamBenchmarkResult includes all benchmarking results for a stream.
type StreamBenchmarkResult struct {
	TTFT Durations
	TBT  Durations
	TT   Durations
}

type StreamFunc func() (<-chan api.ChatCompletionEvent, error)

func (s StreamFunc) execute() ([]api.ChatCompletionEvent, error) {
	// Start streaming.
	eventChan, err := s()
	if err != nil {
		return nil, fmt.Errorf("error in StreamFunc execution: %w", err)
	}

	// To return.
	var events []api.ChatCompletionEvent

	// Read the stream.
	for event := range eventChan {
		events = append(events, event)
	}

	sort.SliceStable(events, func(i, j int) bool { return events[i].Index() < events[j].Index() })
	return events, nil
}

// BenchmarkStream benchmarks a stream as per the given total and concurrent request count.
func BenchmarkStream(totalCount, concurrencyCount int, streamFunc StreamFunc) (StreamBenchmarkResult, error) {
	// This channel makes sure only `concurrencyCount` requests execute concurrently at a given time.
	semaphore := make(chan struct{}, concurrencyCount)
	defer close(semaphore)
	// Channel to receive fatal errors. These errors immediately halt all operations.
	errFatalChan := make(chan error, 1)
	defer close(errFatalChan)

	// Channel to hold events from all requests.
	eventsAllChan := make(chan [2]any, totalCount)
	defer close(eventsAllChan)

	// Channel to hold start and end time for all requests.
	timesChan := make(chan [2]time.Time, totalCount)
	defer close(timesChan)

	// For progress tracking.
	completionCounter := atomic.Int64{}

	for i := 0; i < totalCount; i++ {
		select {
		// Check if a fatal error has occurred.
		case err := <-errFatalChan:
			return StreamBenchmarkResult{}, err
		// Acquire lock for concurrent request.
		case semaphore <- struct{}{}:
			// Execute a stream concurrently.
			go func(i int) {
				// Release lock for the next concurrent request.
				defer func() { <-semaphore }()

				// This is required for calculate TTFT.
				startTime := time.Now()
				// Run the stream.
				events, err := streamFunc.execute()
				if err != nil {
					errFatalChan <- fmt.Errorf("error in StreamFunc execution for iteration: %d: %w", i, err)
					return
				}
				// This is required to calculate total time taken by the response.
				endTime := time.Now()

				// Progress log.
				completionCounter.Add(1)
				fmt.Printf("[%d/%d] requests complete.\n", completionCounter.Load(), totalCount)

				// Record.
				eventsAllChan <- [2]any{i, events}
				timesChan <- [2]time.Time{startTime, endTime}
			}(i)
		}
	}

	// Collect all events.
	eventsAll := make([][]api.ChatCompletionEvent, totalCount)
	times := make([][2]time.Time, totalCount)

	for range totalCount {
		indexAndEvents := <-eventsAllChan
		index := indexAndEvents[0].(int)
		eventsAll[index] = indexAndEvents[1].([]api.ChatCompletionEvent)
		times[index] = <-timesChan
	}

	ttftList := timeToFirstToken(eventsAll, times)
	tbtList := timeBetweenTokens(eventsAll, times)
	totalTimeList := totalTime(times)
	return StreamBenchmarkResult{TTFT: ttftList, TBT: tbtList, TT: totalTimeList}, nil
}
