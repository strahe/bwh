package client

import "fmt"

// BWHError represents a BWH API error with structured information
type BWHError struct {
	Code    int    `json:"error"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e *BWHError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("BWH API error %d: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("BWH API error %d", e.Code)
}

// IsBWHError checks if an error is a BWH API error
func IsBWHError(err error) bool {
	_, ok := err.(*BWHError)
	return ok
}

// GetBWHError extracts BWH error details from an error
func GetBWHError(err error) (*BWHError, bool) {
	bwhErr, ok := err.(*BWHError)
	return bwhErr, ok
}

// IsAuthenticationError checks if the error is an authentication failure
// Based on observed BWH API behavior: error code 700005
func IsAuthenticationError(err error) bool {
	if bwhErr, ok := GetBWHError(err); ok {
		return bwhErr.Code == 700005 // Authentication failure (verified from mock data)
	}
	return false
}