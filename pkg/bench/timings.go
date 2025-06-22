package bench

import (
	"time"
)

// timings holds the complete timing information of a single stream run.
type timings struct {
	Start, End time.Time
	Events     []time.Time
}

// timingsArray represents the collection of timing information from multiple
// parallel stream runs.
type timingsArray []timings

// TTFTs accumulates the Time To First Token (TTFT) for each stream run into a
// single slice for statistical analysis.
func (a timingsArray) TTFTs() []time.Duration {
	out := make([]time.Duration, 0, len(a))
	for _, t := range a {
		// Safely handle streams that produced no events.
		if len(t.Events) > 0 {
			out = append(out, t.Events[0].Sub(t.Start))
		}
	}
	return out
}

// TBTs accumulates the Time Between Tokens (TBT) for all events across all
// stream runs into a single slice.
func (a timingsArray) TBTs() []time.Duration {
	// Pre-calculate the total number of TBT values for efficient allocation.
	var totalTBTs int
	for _, t := range a {
		if len(t.Events) > 1 {
			totalTBTs += len(t.Events) - 1
		}
	}
	if totalTBTs == 0 {
		return nil
	}

	out := make([]time.Duration, 0, totalTBTs)
	for _, t := range a {
		// Safely handle streams with fewer than two events.
		for i := 1; i < len(t.Events); i++ {
			out = append(out, t.Events[i].Sub(t.Events[i-1]))
		}
	}
	return out
}

// TTs accumulates the Total Time (TT) for each stream run into a single slice.
func (a timingsArray) TTs() []time.Duration {
	out := make([]time.Duration, len(a))
	for i, t := range a {
		out[i] = t.End.Sub(t.Start)
	}
	return out
}
