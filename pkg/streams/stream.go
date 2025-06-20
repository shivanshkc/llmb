// Package streams provides generic, zero-overhead, pull-based stream iterators.
//
// The primary goal of this package is to offer a type-safe and efficient way to
// process sequences of data, such as items from a channel, without the overhead
// of creating new goroutines and channels for each transformation step.
//
// # The Problem with Channel Adapters
//
// A common pattern in Go for processing data from a channel is to create a
// "channel adapter": a new goroutine that reads from a source channel, transforms
// the data, and pushes it to a new destination channel.
//
//	// A goroutine to convert a channel of ints to a channel of strings.
//	func adapt(intChan <-chan int) <-chan string {
//		stringChan := make(chan string)
//		go func() {
//			defer close(stringChan)
//			for i := range intChan {
//				stringChan <- fmt.Sprintf("Value: %d", i)
//			}
//		}()
//		return stringChan
//	}
//
// While functional, this pattern introduces overhead for each step in a pipeline:
//   - A new goroutine consumes memory and adds work for the Go scheduler.
//   - A new channel adds buffer memory and latency.
//
// # The Stream Solution
//
// This package provides a `Stream` type that avoids this overhead. It allows you to
// build multi-stage processing pipelines that are executed synchronously within the
// consumer's goroutine. Data is "pulled" through the pipeline on demand by calling
// the Next() method, with no intermediate goroutines or channels.
package streams

// Stream represents a lazy, pull-based iterator over a sequence of items of type T.
//
// A Stream is a lightweight object that wraps a function closure. This closure,
// when called, produces the next item in the sequence. Streams are typically
// created from a source (like a channel via New) and then chained
// together using transformation functions like Map.
//
// The zero value of a Stream is not useful and will panic if Next() is called.
type Stream[T any] struct {
	// next is the core of the stream. It's a function that, when called,
	// returns the next item and a boolean indicating if the item is valid.
	next func() (T, bool)
}

// New creates a new Stream from a read-only channel.
//
// This function is the primary entry point for bringing data from Go's
// concurrent channel-based world into the synchronous, pull-based stream
// paradigm. The returned Stream will produce items until the source channel is
// closed and drained.
func New[T any](sourceChan <-chan T) Stream[T] {
	return Stream[T]{
		next: func() (T, bool) {
			val, ok := <-sourceChan
			return val, ok
		},
	}
}

// Map returns a new Stream that applies the conversion function `conv` to each
// item from a source Stream.
//
// This is a lazy operation. The conversion function is not called until the
// Next() method of the returned Stream is invoked. This allows for the creation
// of complex, multi-stage processing pipelines that remain efficient.
//
// The function is fully type-safe, converting a `Stream[T]` to a `Stream[U]`.
func Map[T, U any](sourceStream Stream[T], conv func(T) U) Stream[U] {
	return Stream[U]{
		next: func() (U, bool) {
			// Pull the next item from the upstream source stream.
			val, ok := sourceStream.Next()
			if !ok {
				// The source is exhausted, so this new stream is also exhausted.
				// Return the zero value for U and ok=false.
				var zeroU U
				return zeroU, false
			}
			// An item was received. Apply the conversion and return it.
			return conv(val), true
		},
	}
}

// Next produces the next item from the stream.
//
// It returns the item and a boolean `ok`. The `ok` flag is true if an item was
// successfully produced, and false if the stream is exhausted. The consumer
// MUST check the `ok` flag to correctly terminate iteration.
func (s *Stream[T]) Next() (T, bool) {
	return s.next()
}

// All is a more convenient way of looping over the Stream for Go 1.22+
func (s *Stream[T]) All(yield func(T) bool) {
	for {
		event, ok := s.next()
		if !ok {
			return
		}

		if !yield(event) {
			return
		}
	}
}
