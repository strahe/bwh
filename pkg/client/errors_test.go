package client

import (
	"context"
	"testing"
)

func TestBWHError(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		message  string
		expected string
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &BWHError{
				Code:    tt.code,
				Message: tt.message,
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
	
	// Note: Only test IsAuthenticationError since it's the only one we have verified data for
	// Other error type functions (NotFound, RateLimit) would need actual BWH API error data to be reliable
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