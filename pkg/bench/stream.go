package bench

import (
	"time"

	"github.com/shivanshkc/llmb/pkg/streams"
)

// Event is the building block of a stream that can be benchmarked.
//
// Only the index and timestamp of an event are required for benchmarking.
// They are used for ordering and latency calculation respectively.
type Event interface {
	Index() int
	Timestamp() time.Time
}

// StreamFunc represents any operation that produces a stream of events.
// It could be a function that reads from a channel, an API call, or any other source.
type StreamFunc func() (*streams.Stream[Event], error)
