package httpx

import (
	"fmt"
	"net/http"
	"time"
)

// RetryClient is an extension of the standard HTTP client.
// It provides a DoRetry method that keeps executing the given request until it succeeds.
// Here, success means the `Do` method does not return a transient error.
type RetryClient struct {
	*http.Client
}

// DoRetry internally calls the `Do` method of the standard HTTP client on the given request.
// If `Do` returns an error, the operation is retried up to maxAttempts times.
func (rc *RetryClient) DoRetry(req *http.Request, maxAttempts int, delay time.Duration) (*http.Response, error) {
	// Request must be rewindable for retries.
	if req.GetBody == nil {
		return nil, fmt.Errorf("GetBody function must be set on the request for retrying")
	}

	// This will hold the error that will be returned of all retries fail.
	var errFinal error

	for i := 0; i < maxAttempts; i++ {
		// Clone the request for each attempt.
		reqClone := req.Clone(req.Context())
		reqClone.RequestURI = ""

		// Create a fresh body for this attempt.
		bodyReader, err := req.GetBody()
		if err != nil {
			return nil, fmt.Errorf("error in the GetBody call: %w", err)
		}
		reqClone.Body = bodyReader

		// Attempt the request.
		response, err := rc.Do(reqClone)
		if err == nil {
			// Success! The caller is now responsible for closing the response body.
			return response, nil
		}

		// Record the error. If this is the final retry, this error will be returned.
		errFinal = err
		// Don't execute the waiting code if this is the last iteration.
		if i == maxAttempts-1 {
			break
		}

		// Timer to wait before next retry.
		timer := time.NewTimer(delay)
		// Wait before the next retry while respecting the request's context.
		select {
		case <-reqClone.Context().Done():
			timer.Stop()                         // Cleanup the timer. `time.After` does not allow this optimization.
			return nil, reqClone.Context().Err() // Return the context's error.
		case <-timer.C:
			// Continue to the next attempt.
		}
	}

	return nil, fmt.Errorf("all %d attempts failed, last error: %w", maxAttempts, errFinal)
}
