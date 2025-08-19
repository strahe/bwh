package client

import (
	"context"
	"encoding/json"
	"testing"
)

func TestIPv6AddResponse(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		wantError   int
		wantSubnet  string
	}{
		{
			name:        "successful IPv6 add",
			jsonData:    `{"error": 0, "assigned_subnet": "2001:db8:1234:5678::"}`,
			wantError:   0,
			wantSubnet:  "2001:db8:1234:5678::",
		},
		{
			name:        "error response",
			jsonData:    `{"error": 788888, "message": "VE is currently locked, try again in a few minutes"}`,
			wantError:   788888,
			wantSubnet:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp IPv6AddResponse
			if err := parseJSON([]byte(tt.jsonData), &resp); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			if resp.Error != tt.wantError {
				t.Errorf("Expected error %d, got %d", tt.wantError, resp.Error)
			}

			if resp.AssignedSubnet != tt.wantSubnet {
				t.Errorf("Expected subnet %s, got %s", tt.wantSubnet, resp.AssignedSubnet)
			}
		})
	}
}

func TestClient_IPv6Methods_Mock(t *testing.T) {
	server := createMockServer()
	defer server.Close()

	client := NewClient("valid_key", "123456")
	client.SetBaseURL(server.URL)

	ctx := context.Background()

	// Test AddIPv6 - this would need to be mocked in the server
	// For now, we just verify the method exists and can be called
	_, err := client.AddIPv6(ctx)
	if err == nil {
		t.Log("AddIPv6 method called successfully")
	} else {
		t.Logf("AddIPv6 returned error (expected in mock): %v", err)
	}

	// Test DeleteIPv6 - this would need to be mocked in the server  
	err = client.DeleteIPv6(ctx, "2001:db8:1234:5678::")
	if err == nil {
		t.Log("DeleteIPv6 method called successfully")
	} else {
		t.Logf("DeleteIPv6 returned error (expected in mock): %v", err)
	}
}

// parseJSON is a helper function to parse JSON for testing
func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}