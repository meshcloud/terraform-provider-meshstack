package internal

import (
	"fmt"
	"net/http"
)

// HttpError represents an HTTP error response with status code.
// This error is returned when an HTTP request fails with a non-2XX status code.
type HttpError struct {
	StatusCode   int
	ResponseBody []byte
}

func (e HttpError) Error() string {
	return fmt.Sprintf("http error %d, response '%s'", e.StatusCode, string(e.ResponseBody))
}

// IsForbidden returns true if the error is a 403 Forbidden response.
func (e HttpError) IsForbidden() bool {
	return e.StatusCode == http.StatusForbidden
}

// IsNotFound returns true if the error is a 404 Not Found response.
func (e HttpError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}
