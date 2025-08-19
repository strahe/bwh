package client

import "fmt"

// BWHError represents a BWH API error with structured information
type BWHError struct {
	Code                   int                     `json:"error"`
	Message                string                  `json:"message"`
	AdditionalErrorInfo    string                  `json:"additionalErrorInfo,omitempty"`
	AdditionalLockingInfo  *AdditionalLockingInfo  `json:"additionalLockingInfo,omitempty"`
}

// AdditionalLockingInfo contains detailed locking status information
type AdditionalLockingInfo struct {
	LastStatusUpdateSecondsAgo int    `json:"last_status_update_s_ago"`
	CompletedPercent          int    `json:"completed_percent"`
	FriendlyProgressMessage   string `json:"friendly_progress_message"`
}

// Error implements the error interface
func (e *BWHError) Error() string {
	var msg string
	if e.Message != "" {
		msg = fmt.Sprintf("BWH API error %d: %s", e.Code, e.Message)
	} else {
		msg = fmt.Sprintf("BWH API error %d", e.Code)
	}

	// Add additional error info if available
	if e.AdditionalErrorInfo != "" {
		msg += fmt.Sprintf("\nOperation: %s", e.AdditionalErrorInfo)
	}

	// Add locking details if available
	if e.AdditionalLockingInfo != nil {
		info := e.AdditionalLockingInfo
		msg += fmt.Sprintf("\nProgress: %d%% complete - %s", info.CompletedPercent, info.FriendlyProgressMessage)
		if info.LastStatusUpdateSecondsAgo > 0 {
			msg += fmt.Sprintf(" (updated %ds ago)", info.LastStatusUpdateSecondsAgo)
		}
	}

	return msg
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

// IsLockedError checks if the error is due to VPS being locked
// Based on observed BWH API behavior: error code 788888
func IsLockedError(err error) bool {
	if bwhErr, ok := GetBWHError(err); ok {
		return bwhErr.Code == 788888 // VE is currently locked (verified from mock data)
	}
	return false
}