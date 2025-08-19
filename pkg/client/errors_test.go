package client

import (
	"context"
	"testing"
)

func TestBWHError(t *testing.T) {
	tests := []struct {
		name                  string
		code                  int
		message               string
		additionalErrorInfo   string
		additionalLockingInfo *AdditionalLockingInfo
		expected              string
	}{
		{
			name:     "error with message",
			code:     700005,
			message:  "Authentication failure",
			expected: "BWH API error 700005: Authentication failure",
		},
		{
			name:     "error without message",
			code:     404,
			message:  "",
			expected: "BWH API error 404",
		},
		{
			name:     "success code",
			code:     0,
			message:  "",
			expected: "BWH API error 0",
		},
		{
			name:                "locked error with additional info",
			code:                788888,
			message:             "VE is currently locked, try again in a few minutes",
			additionalErrorInfo: "OS Reinstall: debian-13-x86_64",
			additionalLockingInfo: &AdditionalLockingInfo{
				LastStatusUpdateSecondsAgo: 19,
				CompletedPercent:           80,
				FriendlyProgressMessage:    "Starting VM",
			},
			expected: "BWH API error 788888: VE is currently locked, try again in a few minutes\nOperation: OS Reinstall: debian-13-x86_64\nProgress: 80% complete - Starting VM (updated 19s ago)",
		},
		{
			name:                "locked error without time update",
			code:                788888,
			message:             "VE is currently locked, try again in a few minutes",
			additionalErrorInfo: "Snapshot Creation",
			additionalLockingInfo: &AdditionalLockingInfo{
				LastStatusUpdateSecondsAgo: 0,
				CompletedPercent:           45,
				FriendlyProgressMessage:    "Creating snapshot",
			},
			expected: "BWH API error 788888: VE is currently locked, try again in a few minutes\nOperation: Snapshot Creation\nProgress: 45% complete - Creating snapshot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &BWHError{
				Code:                  tt.code,
				Message:               tt.message,
				AdditionalErrorInfo:   tt.additionalErrorInfo,
				AdditionalLockingInfo: tt.additionalLockingInfo,
			}
			
			if err.Error() != tt.expected {
				t.Errorf("Expected error message '%s', got '%s'", tt.expected, err.Error())
			}
		})
	}
}

func TestBWHErrorHelpers(t *testing.T) {
	// Test IsBWHError
	bwhErr := &BWHError{Code: 700005, Message: "Authentication failure"}
	normalErr := context.Canceled
	
	if !IsBWHError(bwhErr) {
		t.Error("Expected IsBWHError to return true for BWHError")
	}
	
	if IsBWHError(normalErr) {
		t.Error("Expected IsBWHError to return false for non-BWHError")
	}
	
	// Test GetBWHError
	if extracted, ok := GetBWHError(bwhErr); !ok || extracted.Code != 700005 {
		t.Error("Expected GetBWHError to extract BWHError correctly")
	}
	
	if _, ok := GetBWHError(normalErr); ok {
		t.Error("Expected GetBWHError to return false for non-BWHError")
	}
	
	// Test IsAuthenticationError
	authErr := &BWHError{Code: 700005, Message: "Authentication failure"}
	otherErr := &BWHError{Code: 404, Message: "Not found"}
	
	if !IsAuthenticationError(authErr) {
		t.Error("Expected IsAuthenticationError to return true for auth error")
	}
	
	if IsAuthenticationError(otherErr) {
		t.Error("Expected IsAuthenticationError to return false for non-auth error")
	}
	
	if IsAuthenticationError(normalErr) {
		t.Error("Expected IsAuthenticationError to return false for non-BWH error")
	}
	
	// Test IsLockedError
	lockedErr := &BWHError{Code: 788888, Message: "VE is currently locked, try again in a few minutes"}
	
	if !IsLockedError(lockedErr) {
		t.Error("Expected IsLockedError to return true for locked error")
	}
	
	if IsLockedError(otherErr) {
		t.Error("Expected IsLockedError to return false for non-locked error")
	}
	
	if IsLockedError(normalErr) {
		t.Error("Expected IsLockedError to return false for non-BWH error")
	}
}

func TestWrapError(t *testing.T) {
	// Test success case
	resp := "test response"
	result, err := wrapError(resp, 0, "")
	if err != nil {
		t.Errorf("Expected no error for success case, got %v", err)
	}
	if result != resp {
		t.Errorf("Expected result '%s', got '%s'", resp, result)
	}
	
	// Test error case
	_, err = wrapError("", 700005, "Authentication failure")
	if err == nil {
		t.Fatal("Expected error for non-zero code")
	}
	
	bwhErr, ok := err.(*BWHError)
	if !ok {
		t.Fatal("Expected BWHError type")
	}
	
	if bwhErr.Code != 700005 {
		t.Errorf("Expected code 700005, got %d", bwhErr.Code)
	}
	
	if bwhErr.Message != "Authentication failure" {
		t.Errorf("Expected message 'Authentication failure', got '%s'", bwhErr.Message)
	}
}

func TestWrapOnlyError(t *testing.T) {
	// Test success case
	err := wrapOnlyError(0, "")
	if err != nil {
		t.Errorf("Expected no error for success case, got %v", err)
	}
	
	// Test error case
	err = wrapOnlyError(700005, "Authentication failure")
	if err == nil {
		t.Fatal("Expected error for non-zero code")
	}
	
	bwhErr, ok := err.(*BWHError)
	if !ok {
		t.Fatal("Expected BWHError type")
	}
	
	if bwhErr.Code != 700005 {
		t.Errorf("Expected code 700005, got %d", bwhErr.Code)
	}
	
	if bwhErr.Message != "Authentication failure" {
		t.Errorf("Expected message 'Authentication failure', got '%s'", bwhErr.Message)
	}
}

func TestClient_StructuredErrors_Mock(t *testing.T) {
	server := createMockServer()
	defer server.Close()
	
	// Test with invalid API key to trigger structured error
	client := NewClient("invalid_key", "123456")
	client.SetBaseURL(server.URL)
	
	_, err := client.GetServiceInfo(context.Background())
	if err == nil {
		t.Fatal("Expected error for invalid API key")
	}
	
	// Check that it's a BWHError
	if !IsBWHError(err) {
		t.Fatalf("Expected BWHError, got %T: %v", err, err)
	}
	
	// Check that it's specifically an authentication error
	if !IsAuthenticationError(err) {
		t.Error("Expected authentication error")
	}
	
	// Extract and verify error details
	bwhErr, ok := GetBWHError(err)
	if !ok {
		t.Fatal("Failed to extract BWHError")
	}
	
	if bwhErr.Code != 700005 {
		t.Errorf("Expected error code 700005, got %d", bwhErr.Code)
	}
	
	if bwhErr.Message != "Authentication failure" {
		t.Errorf("Expected message 'Authentication failure', got '%s'", bwhErr.Message)
	}
	
	// Test the error message format
	expectedMsg := "BWH API error 700005: Authentication failure"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestEnhancedErrorDisplay(t *testing.T) {
	// Test enhanced error display with mock locked error data
	lockedError := &BWHError{
		Code:                788888,
		Message:             "VE is currently locked, try again in a few minutes",
		AdditionalErrorInfo: "OS Reinstall: debian-13-x86_64",
		AdditionalLockingInfo: &AdditionalLockingInfo{
			LastStatusUpdateSecondsAgo: 19,
			CompletedPercent:          80,
			FriendlyProgressMessage:    "Starting VM",
		},
	}

	expectedMsg := "BWH API error 788888: VE is currently locked, try again in a few minutes\nOperation: OS Reinstall: debian-13-x86_64\nProgress: 80% complete - Starting VM (updated 19s ago)"
	
	if lockedError.Error() != expectedMsg {
		t.Errorf("Expected enhanced error message:\n%s\n\nGot:\n%s", expectedMsg, lockedError.Error())
	}

	// Verify IsLockedError works correctly
	if !IsLockedError(lockedError) {
		t.Error("Expected IsLockedError to return true for locked error")
	}
}