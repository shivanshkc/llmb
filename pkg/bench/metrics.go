package bench

import (
	"sort"
	"time"

	"github.com/shivanshkc/llmb/pkg/api"
)

// timeToFirstToken calculates the "Time to First Token" metric for all the given event lists.
func timeToFirstToken(events [][]api.ChatCompletionEvent, times [][2]time.Time) []time.Duration {
	// List of all TTFT.
	ttft := make([]time.Duration, len(events))
	for i, eventList := range events {
		// Only concerned with the first event.
		if len(eventList) != 0 {
			ttft[i] = eventList[0].Received.Sub(times[i][0])
		}
	}
	return ttft
}

func timeBetweenTokens(events [][]api.ChatCompletionEvent, times [][2]time.Time) []time.Duration {
	var durations []time.Duration
	for _, eventList := range events {
		for i := 0; i < len(eventList)-1; i++ {
			durations = append(durations, eventList[i+1].Received.Sub(eventList[i].Received))
		}
	}
	return durations
}

func totalTime(times [][2]time.Time) []time.Duration {
	var totalDurations []time.Duration
	for _, t := range times {
		totalDurations = append(totalDurations, t[1].Sub(t[0]))
	}
	return totalDurations
}

type Durations []time.Duration

// Average calculates the mean of a slice of time.Duration values.
func (ds Durations) Average() time.Duration {
	if len(ds) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range ds {
		total += d
	}
	return total / time.Duration(len(ds))
}

// Minimum finds the smallest time.Duration in the slice.
func (ds Durations) Minimum() time.Duration {
	if len(ds) == 0 {
		return 0
	}

	m := ds[0]
	for _, d := range ds {
		if d < m {
			m = d
		}
	}
	return m
}

// Median finds the middle value of a sorted slice of time.Duration.
func (ds Durations) Median() time.Duration {
	if len(ds) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(ds))
	copy(sorted, ds)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

// Maximum finds the largest time.Duration in the slice.
func (ds Durations) Maximum() time.Duration {
	if len(ds) == 0 {
		return 0
	}

	m := ds[0]
	for _, d := range ds {
		if d > m {
			m = d
		}
	}
	return m
}

// Percentile calculates the Pxx value for a slice of time.Duration.
// Given percentile should be between 0 and 100.
func (ds Durations) Percentile(percentile float64) time.Duration {
	if len(ds) == 0 || percentile < 0 || percentile > 100 {
		return 0
	}

	sorted := make([]time.Duration, len(ds))
	copy(sorted, ds)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	index := int(float64(len(sorted)-1) * (percentile / 100.0))
	return sorted[index]
}
