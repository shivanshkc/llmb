// Package bench provides tools for benchmarking streaming operations, particularly
// those that produce a sequence of events over time.
//
// It is designed to measure key performance indicators for language models and
// other stream-based APIs, such as:
//   - Time To First Token (TTFT): How quickly the stream begins producing data.
//   - Time Between Tokens (TBT): The latency between consecutive data events.
//   - Total Time (TT): The end-to-end duration of the entire stream.
//
// The package manages concurrent execution of benchmark tasks and aggregates the
// results into statistical metrics (Avg, Min, Max, P90, P95, etc.).
package bench

import (
	"sort"
	"time"
)

// Metrics holds a collection of standard statistical measurements for a set of
// timing durations. All values are expressed as time.Duration.
type Metrics struct {
	Avg time.Duration // The average (mean) duration.
	Min time.Duration // The minimum (fastest) duration.
	Med time.Duration // The median (50th percentile) duration.
	Max time.Duration // The maximum (slowest) duration.
	P90 time.Duration // The 90th percentile duration.
	P95 time.Duration // The 95th percentile duration.
}

// durations represents a slice of time measurements, forming the raw data
// for calculating performance metrics.
type durations []time.Duration

// Metrics calculates and returns all the statistical metrics for the given set
// of durations. It sorts the data once to efficiently calculate all percentile-based
// metrics.
func (ds durations) Metrics() Metrics {
	if len(ds) == 0 {
		return Metrics{}
	}

	// Create a sorted copy to avoid modifying the original slice and to
	// perform all calculations efficiently.
	sorted := make(durations, len(ds))
	copy(sorted, ds)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	return Metrics{
		Avg: ds.average(), // Average does not require sorting.
		Min: sorted[0],
		Med: sorted.median(),
		Max: sorted[len(sorted)-1],
		P90: sorted.percentile(90),
		P95: sorted.percentile(95),
	}
}

// average calculates the mean of a slice of time.Duration values.
func (ds durations) average() time.Duration {
	if len(ds) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range ds {
		total += d
	}
	return total / time.Duration(len(ds))
}

// median finds the middle value of a *sorted* slice of time.Duration.
// The receiver slice must be sorted before calling this method.
func (ds durations) median() time.Duration {
	mid := len(ds) / 2
	if len(ds)%2 == 0 {
		// Even number of elements, average the two middle ones.
		return (ds[mid-1] + ds[mid]) / 2
	}
	// Odd number of elements, return the middle one.
	return ds[mid]
}

// percentile calculates the Pxx value for a *sorted* slice of time.Duration.
// The receiver slice must be sorted before calling this method.
// The given percentile should be between 0 and 100.
func (ds durations) percentile(percentile float64) time.Duration {
	if percentile < 0 {
		percentile = 0
	}
	if percentile > 100 {
		percentile = 100
	}

	// Use the Nearest Rank method.
	index := int(float64(len(ds)-1) * (percentile / 100.0))
	return ds[index]
}
