package client

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		veid   string
	}{
		{
			name:   "valid client creation",
			apiKey: "test-api-key-123456789",
			veid:   "123456",
		},
		{
			name:   "empty values",
			apiKey: "",
			veid:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.apiKey, tt.veid)

			if client == nil {
				t.Fatal("NewClient() returned nil")
			}

			if client.apiKey != tt.apiKey {
				t.Errorf("NewClient() apiKey = %v, want %v", client.apiKey, tt.apiKey)
			}

			if client.veid != tt.veid {
				t.Errorf("NewClient() veid = %v, want %v", client.veid, tt.veid)
			}

			if client.baseURL != defaultBaseURL {
				t.Errorf("NewClient() baseURL = %v, want %v", client.baseURL, defaultBaseURL)
			}

			if client.httpClient == nil {
				t.Error("NewClient() httpClient is nil")
			}

			// Check default timeout
			if client.httpClient.Timeout.Seconds() != 30 {
				t.Errorf("NewClient() timeout = %v seconds, want 30", client.httpClient.Timeout.Seconds())
			}
		})
	}
}

func TestSetBaseURL(t *testing.T) {
	client := NewClient("test-key", "123456")
	customURL := "https://custom-api.example.com/v1"

	client.SetBaseURL(customURL)

	if client.baseURL != customURL {
		t.Errorf("SetBaseURL() baseURL = %v, want %v", client.baseURL, customURL)
	}
}
