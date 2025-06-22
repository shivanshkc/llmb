package bench

import (
	"context"
	"time"

	"github.com/shivanshkc/llmb/pkg/streams"
)

// Event is the minimal interface a stream's event must implement to be benchmarked.
// It provides the essential data needed for ordering and latency calculations.
type Event interface {
	Index() int           // The sequential index of the event, for stable sorting.
	Timestamp() time.Time // The time the event was produced or received.
}

// StreamFunc represents any operation that produces a cancellable stream of events.
// This is the primary input to the benchmark runner.
type StreamFunc func(ctx context.Context) (*streams.Stream[Event], error)
