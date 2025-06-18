package bench

import (
	"sort"
	"time"
)

// Metrics ...
type Metrics struct {
	Avg, Min, Med, Max, P90, P95 time.Duration
}

// durations represents the time taken by each instance of a repeating process.
type durations []time.Duration

// Metrics calculates and returns all the metrics for the given set of durations.
func (ds durations) Metrics() Metrics {
	return Metrics{
		Avg: ds.Average(),
		Min: ds.Minimum(),
		Med: ds.Median(),
		Max: ds.Maximum(),
		P90: ds.Percentile(90),
		P95: ds.Percentile(95),
	}
}

// Average calculates the mean of a slice of time.Duration values.
func (ds durations) Average() time.Duration {
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
func (ds durations) Minimum() time.Duration {
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
func (ds durations) Median() time.Duration {
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
func (ds durations) Maximum() time.Duration {
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
func (ds durations) Percentile(percentile float64) time.Duration {
	if len(ds) == 0 || percentile < 0 || percentile > 100 {
		return 0
	}

	sorted := make([]time.Duration, len(ds))
	copy(sorted, ds)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	index := int(float64(len(sorted)-1) * (percentile / 100.0))
	return sorted[index]
}
