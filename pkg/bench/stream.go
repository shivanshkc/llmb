package bench

import (
	"time"

	"github.com/shivanshkc/llmb/pkg/streams"
)

// Event is the building block of a stream.
//
// Only the index and timestamp of an event are required for benchmarking.
// They are used to ordering and latency calculation respectively.
type Event interface {
	Index() int
	Timestamp() time.Time
}

// StreamFunc represents a benchmark-able function.
type StreamFunc func() (streams.Stream[Event], error)

// Consume the stream.
func (f StreamFunc) Consume() ([]Event, error) {
	// Start the stream.
	eventStream, err := f()
	if err != nil {
		return nil, err
	}

	// Collect all events.
	var events []Event
	for event := range eventStream.All {
		events = append(events, event)
	}

	return events, nil
}
