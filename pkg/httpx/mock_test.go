package httpx_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
)

// mockRoundTripper is a mock implementation of http.RoundTripper.
// It allows us to simulate different network responses (success, failure)
// for each attempt, without making real network calls.
type mockRoundTripper struct {
	// A slice of functions, where each function represents the outcome
	// of one `Do` attempt.
	responses []func(*http.Request) (*http.Response, error)
	// attempt tracks the current call number.
	attempt int
}

// RoundTrip satisfies the http.RoundTripper interface. It invokes the response
// function corresponding to the current attempt number.
func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Ensure we don't go out of bounds. If we do, it likely means
	// the retry logic is attempting more calls than we've configured our mock for.
	if m.attempt >= len(m.responses) {
		return nil, errors.New("mockRoundTripper: too many attempts")
	}

	// Get the response function for the current attempt.
	responseFunc := m.responses[m.attempt]
	m.attempt++ // Increment for the next call.
	return responseFunc(req)
}

// mockReadCloser is a mock implementation of io.ReadCloser. It is used to
// simulate various behaviors of a response body, such as returning specific
// data, errors, or blocking, without making real network calls. It also
// tracks whether its Close method has been called.
type mockReadCloser struct {
	reader io.Reader
	mu     sync.Mutex
	closed bool
}

// newMockReadCloser creates a new mock body from a string.
func newMockReadCloser(data string) *mockReadCloser {
	return &mockReadCloser{reader: strings.NewReader(data)}
}

// Read satisfies the io.Reader interface, delegating to the internal reader.
func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	return m.reader.Read(p)
}

// Close satisfies the io.Closer interface. It records that it has been called.
func (m *mockReadCloser) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// isClosed safely checks if the Close method has been called.
func (m *mockReadCloser) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// blockingReadCloser is a mock designed to accurately simulate a network
// connection. It blocks on Read until its Close method is called from another
// goroutine, at which point it unblocks and returns an error.
type blockingReadCloser struct {
	mu        sync.Mutex
	closed    bool
	closeChan chan struct{} // A channel to signal that Close has been called.
}

// newBlockingReadCloser creates an instance of the blocking mock.
func newBlockingReadCloser() *blockingReadCloser {
	return &blockingReadCloser{
		closeChan: make(chan struct{}),
	}
}

// Read satisfies the io.Reader interface. It blocks until the closeChan receives
// a signal (from the Close method) or until the test times out.
func (m *blockingReadCloser) Read(p []byte) (n int, err error) {
	// Block until Close() is called.
	<-m.closeChan
	// Once unblocked, return an error that simulates a closed connection.
	return 0, io.ErrClosedPipe
}

// Close satisfies the io.Closer interface. It records that it has been called
// and, crucially, closes the closeChan to unblock any pending Read calls.
func (m *blockingReadCloser) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil // Already closed.
	}
	m.closed = true

	// This is the critical part: signal any blocked Read calls to unblock.
	close(m.closeChan)

	return nil
}

// isClosed safely checks if the Close method has been called.
func (m *blockingReadCloser) isClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// errorReader is a helper that implements io.Reader and always returns an error.
type errorReader struct {
	err error
}

func (e *errorReader) Read([]byte) (n int, err error) {
	return 0, e.err
}
