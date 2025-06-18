package bench

import (
	"time"
)

// timings holds the complete timing information of a stream.
type timings struct {
	Start  time.Time
	End    time.Time
	Events []time.Time
}

// timingsArray represents the collection of the timing information of multiple streams.
type timingsArray []timings

// TTFTs accumulates TTFT values for each stream into one time.Duration slice.
func (a timingsArray) TTFTs() []time.Duration {
	out := make([]time.Duration, len(a))
	for i, t := range a {
		out[i] = t.Events[0].Sub(t.Start)
	}
	return out
}

// TBTs accumulates TBT values for all streams into one time.Duration slice.
func (a timingsArray) TBTs() []time.Duration {
	var out []time.Duration
	for _, t := range a {
		o := make([]time.Duration, len(t.Events)-1)
		for i := 1; i < len(t.Events); i++ {
			o[i-1] = t.Events[i].Sub(t.Events[i-1])
		}
		out = append(out, o...)
	}
	return out
}

// TTs accumulates TT values for all streams into one time.Duration slice.
func (a timingsArray) TTs() []time.Duration {
	out := make([]time.Duration, len(a))
	for i, t := range a {
		out[i] = t.End.Sub(t.Start)
	}
	return out
}
