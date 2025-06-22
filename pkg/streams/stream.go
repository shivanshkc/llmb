// Package streams provides generic, zero-overhead, pull-based stream iterators
// that are fully context-aware and cancellable.
//
// The primary goal of this package is to offer a type-safe and efficient way to
// process sequences of data, such as items from a channel, without the overhead
// of creating new goroutines and channels for each transformation step.
//
// # The Stream Solution
//
// This package provides a `Stream` type that avoids the overhead of traditional
// channel adapters. It allows you to build multi-stage processing pipelines that
// are executed synchronously within the consumer's goroutine. Data is "pulled"
// through the pipeline on demand by calling the NextContext() method.
//
// # A Context-Aware Channel Abstraction
//
// In addition to being a zero-overhead iterator, a `Stream` can also be thought
// of as a higher-level, cancellable channel. A common Go pattern for a
// cancellable read from a channel involves a `select` statement:
//
//	var value T
//	select {
//	case <-ctx.Done():
//		// Handle cancellation
//		return ctx.Err()
//	case v, ok := <-myChan:
//		if !ok {
//			// Handle closed channel
//			return
//		}
//		value = v
//	}
//
// The `Stream` type encapsulates this boilerplate. The same operation can be
// achieved with a single, clean call to NextContext:
//
//	value, ok, err := myStream.NextContext(ctx)
//	if err != nil {
//		// Handles cancellation
//	}
//	if !ok {
//		// Handles closed stream
//	}
//
// This abstraction makes consumer code significantly cleaner and less error-prone,
// while also making the cancellable behavior effortlessly composable through
// functions like `Map`.
package streams

import (
	"context"
)

// Stream represents a lazy, pull-based, cancellable iterator over a sequence of
// items of type T.
//
// A Stream is a lightweight object that wraps a function closure. This closure,
// when called, produces the next item in the sequence. Streams are typically
// created from a source (like a channel via New) and then chained
// together using transformation functions like Map.
//
// The zero value of a Stream is not useful and will panic if Next() is called.
type Stream[T any] struct {
	// next is the core of the stream. It's a context-aware method that returns
	// the next event, a boolean indicating if the item is valid, and an error
	// if the context was canceled during the operation.
	next func(ctx context.Context) (T, bool, error)
}

// New creates a new Stream from a read-only channel.
//
// This function is the primary entry point for bringing data from Go's
// concurrent channel-based world into the synchronous, pull-based stream
// paradigm. The returned Stream will produce items until the source channel is
// closed and drained, or until the provided context is canceled.
func New[T any](sourceChan <-chan T) *Stream[T] {
	return &Stream[T]{
		next: func(ctx context.Context) (T, bool, error) {
			select {
			case <-ctx.Done():
				var zeroT T
				return zeroT, false, ctx.Err()
			case val, ok := <-sourceChan:
				return val, ok, nil
			}
		},
	}
}

// Map returns a new Stream that applies the conversion function `conv` to each
// item from a source Stream.
//
// This is a lazy operation. The conversion function is not called until the
// NextContext() method of the returned Stream is invoked. This ensures that
// context cancellation is respected throughout the entire pipeline.
func Map[T, U any](sourceStream *Stream[T], conv func(T) U) *Stream[U] {
	return &Stream[U]{
		next: func(ctx context.Context) (U, bool, error) {
			var zeroU U

			// Pull the item from the upstream source.
			val, ok, err := sourceStream.next(ctx)
			if err != nil {
				return zeroU, false, err
			}

			// End of stream.
			if !ok {
				return zeroU, false, nil
			}

			// An item was received. Apply the conversion and return it.
			return conv(val), true, nil
		},
	}
}

// Next is a convenience method that produces the next item from the stream
// using a background context. It is not cancellable. For cancellable
// iteration, use NextContext.
func (s *Stream[T]) Next() (T, bool) {
	val, ok, _ := s.next(context.Background())
	return val, ok
}

// NextContext produces the next item from the stream, respecting context
// cancellation.
//
// It returns the item, a boolean `ok` (which is false if the stream is
// exhausted), and an error if the context was canceled while waiting for
// the next item. The consumer MUST check `ok` to terminate a loop correctly.
func (s *Stream[T]) NextContext(ctx context.Context) (T, bool, error) {
	return s.next(ctx)
}

// Exhaust blocks until all events are collected from the stream or until the
// context is canceled. It provides a simple way to collect all results into a
// slice.
func (s *Stream[T]) Exhaust(ctx context.Context) ([]T, error) {
	// Pre-allocate with a reasonable capacity to reduce re-allocations.
	items := make([]T, 0, 100)

	for {
		// Pull the next item, respecting the context.
		item, ok, err := s.next(ctx)
		if err != nil {
			return nil, err
		}

		// End of stream.
		if !ok {
			return items, nil
		}

		// Collect the item.
		items = append(items, item)
	}
}
