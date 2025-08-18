package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockResponse holds mock response data for different API endpoints
type mockResponse struct {
	StatusCode int
	Body       string
}

// loadMockFile loads mock data from file
func loadMockFile(filename string) ([]byte, error) {
	// Find the mock directory (in the same package)
	mockPath := filepath.Join("mock", filename)
	return os.ReadFile(mockPath)
}

// createMockServer creates an HTTP test server that responds with mock data
func createMockServer() *httptest.Server {
	// Load mock data from files
	mockResponses := make(map[string]mockResponse)
	
	// Load service info mock
	if serviceInfoData, err := loadMockFile("getServiceInfo.json"); err == nil {
		mockResponses["getServiceInfo"] = mockResponse{
			StatusCode: 200,
			Body:       string(serviceInfoData),
		}
	}
	
	// Load live service info mock
	if liveServiceInfoData, err := loadMockFile("getLiveServiceInfo.json"); err == nil {
		mockResponses["getLiveServiceInfo"] = mockResponse{
			StatusCode: 200,
			Body:       string(liveServiceInfoData),
		}
	}
	
	// Load rate limit mock
	if rateLimitData, err := loadMockFile("getRateLimitStatus.json"); err == nil {
		mockResponses["getRateLimitStatus"] = mockResponse{
			StatusCode: 200,
			Body:       string(rateLimitData),
		}
	}
	
	// Load error response mock
	if errorData, err := loadMockFile("error.json"); err == nil {
		mockResponses["error"] = mockResponse{
			StatusCode: 200, // BWH API always returns 200, errors are in JSON
			Body:       string(errorData),
		}
	}
	
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract endpoint from URL path
		path := strings.TrimPrefix(r.URL.Path, "/")
		
		// Always set 200 status and JSON content type (BWH API pattern)
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		
		// Check for error conditions based on API key
		apiKey := r.URL.Query().Get("api_key")
		if apiKey == "invalid_key" {
			if response, exists := mockResponses["error"]; exists {
				w.Write([]byte(response.Body)) //nolint:errcheck
				return
			}
		}
		
		// Route to appropriate mock response
		if response, exists := mockResponses[path]; exists {
			w.Write([]byte(response.Body)) //nolint:errcheck
		} else {
			// Default error response (still 200 status, error in JSON)
			w.Write([]byte(`{"error": 404, "message": "Endpoint not found"}`)) //nolint:errcheck
		}
	}))
}

func TestClient_GetServiceInfo_Mock(t *testing.T) {
	server := createMockServer()
	defer server.Close()
	
	client := NewClient("valid_key", "123456")
	client.SetBaseURL(server.URL)
	
	info, err := client.GetServiceInfo(context.Background())
	if err != nil {
		t.Fatalf("GetServiceInfo() error = %v", err)
	}
	
	// Verify response structure
	if info.Error != 0 {
		t.Errorf("Expected error = 0, got %d", info.Error)
	}
	
	if info.VMType != "kvm" {
		t.Errorf("Expected vm_type = kvm, got %s", info.VMType)
	}
	
	if info.Hostname != "test-hostname" {
		t.Errorf("Expected hostname = test-hostname, got %s", info.Hostname)
	}
	
	if info.Plan != "kvmv5-megabox-pro-40g-2048m-2000g-dc1" {
		t.Errorf("Expected specific plan, got %s", info.Plan)
	}
	
	if len(info.IPAddresses) != 2 {
		t.Errorf("Expected 2 IP addresses, got %d", len(info.IPAddresses))
	}
	
	// Test specific fields that should be parsed correctly
	if info.PlanRAM != 2168455168 {
		t.Errorf("Expected plan_ram = 2168455168, got %d", info.PlanRAM)
	}
	
	if info.DataCounter != 611537718433 {
		t.Errorf("Expected data_counter = 611537718433, got %d", info.DataCounter)
	}
}

func TestClient_GetLiveServiceInfo_Mock(t *testing.T) {
	server := createMockServer()
	defer server.Close()
	
	client := NewClient("valid_key", "123456")
	client.SetBaseURL(server.URL)
	
	liveInfo, err := client.GetLiveServiceInfo(context.Background())
	if err != nil {
		t.Fatalf("GetLiveServiceInfo() error = %v", err)
	}
	
	// Verify it contains both ServiceInfo and LiveServiceInfo fields
	if liveInfo.Error != 0 {
		t.Errorf("Expected error = 0, got %d", liveInfo.Error)
	}
	
	if liveInfo.VeStatus != "running" {
		t.Errorf("Expected ve_status = running, got %s", liveInfo.VeStatus)
	}
	
	if liveInfo.VeMac1 != "02:00:00:00:00:01" {
		t.Errorf("Expected specific MAC address, got %s", liveInfo.VeMac1)
	}
	
	if liveInfo.LiveHostname != "test-hostname" {
		t.Errorf("Expected live_hostname = test-hostname, got %s", liveInfo.LiveHostname)
	}
	
	// Test FlexibleInt fields that can be strings in the JSON
	expectedDiskSpace := int64(6285897728)
	if liveInfo.VeUsedDiskSpaceB.Value != expectedDiskSpace {
		t.Errorf("Expected ve_used_disk_space_b = %d, got %d", expectedDiskSpace, liveInfo.VeUsedDiskSpaceB.Value)
	}
	
	expectedDiskQuota := int64(41)
	if liveInfo.VeDiskQuotaGB.Value != expectedDiskQuota {
		t.Errorf("Expected ve_disk_quota_gb = %d, got %d", expectedDiskQuota, liveInfo.VeDiskQuotaGB.Value)
	}
	
	// Test FlexibleInt fields that are empty strings (should default to 0)
	if liveInfo.IsCPUThrottled.Value != 0 {
		t.Errorf("Expected is_cpu_throttled = 0 (empty string), got %d", liveInfo.IsCPUThrottled.Value)
	}
	
	if liveInfo.IsDiskThrottled.Value != 0 {
		t.Errorf("Expected is_disk_throttled = 0 (empty string), got %d", liveInfo.IsDiskThrottled.Value)
	}
}

func TestClient_GetRateLimitStatus_Mock(t *testing.T) {
	server := createMockServer()
	defer server.Close()
	
	client := NewClient("valid_key", "123456")
	client.SetBaseURL(server.URL)
	
	rateLimit, err := client.GetRateLimitStatus(context.Background())
	if err != nil {
		t.Fatalf("GetRateLimitStatus() error = %v", err)
	}
	
	if rateLimit.Error != 0 {
		t.Errorf("Expected error = 0, got %d", rateLimit.Error)
	}
	
	if rateLimit.RemainingPoints15Min != 997 {
		t.Errorf("Expected remaining_points_15min = 997, got %d", rateLimit.RemainingPoints15Min)
	}
	
	if rateLimit.RemainingPoints24H != 19852 {
		t.Errorf("Expected remaining_points_24h = 19852, got %d", rateLimit.RemainingPoints24H)
	}
}

func TestClient_ErrorResponse_Mock(t *testing.T) {
	server := createMockServer()
	defer server.Close()
	
	// Use invalid API key to trigger error response
	client := NewClient("invalid_key", "123456")
	client.SetBaseURL(server.URL)
	
	_, err := client.GetServiceInfo(context.Background())
	if err == nil {
		t.Fatal("Expected error for invalid API key, got none")
	}
	
	// Verify it's a structured BWH error
	if !IsBWHError(err) {
		t.Fatalf("Expected BWHError, got %T: %v", err, err)
	}
	
	// Verify error message format
	expectedMsg := "BWH API error 700005: Authentication failure"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
	
	// Verify error details
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
}

func TestClient_UnknownEndpoint_Mock(t *testing.T) {
	server := createMockServer()
	defer server.Close()
	
	client := NewClient("valid_key", "123456")
	client.SetBaseURL(server.URL)
	
	// Try to call an endpoint that doesn't exist in our mock
	// Use a specific result type that includes error handling
	ctx := context.Background()
	var result struct {
		Error   int    `json:"error"`
		Message string `json:"message"`
	}
	
	err := client.doRequest(ctx, "unknownEndpoint", nil, &result)
	
	// doRequest itself should succeed (200 response), but result should contain error
	if err != nil {
		t.Fatalf("doRequest failed: %v", err)
	}
	
	if result.Error == 0 {
		t.Fatal("Expected error code in response, got 0")
	}
	
	// Verify the error details
	if result.Error != 404 {
		t.Errorf("Expected error code 404, got %d", result.Error)
	}
	
	if result.Message != "Endpoint not found" {
		t.Errorf("Expected message 'Endpoint not found', got '%s'", result.Message)
	}
}

// Test helper function to verify JSON parsing edge cases
func TestFlexibleInt_MockDataScenarios(t *testing.T) {
	testCases := []struct {
		name     string
		json     string
		expected int64
	}{
		{
			name:     "string number",
			json:     `{"value": "41"}`,
			expected: 41,
		},
		{
			name:     "integer number",
			json:     `{"value": 1234567890}`,
			expected: 1234567890,
		},
		{
			name:     "empty string",
			json:     `{"value": ""}`,
			expected: 0,
		},
		{
			name:     "zero string",
			json:     `{"value": "0"}`,
			expected: 0,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result struct {
				Value FlexibleInt `json:"value"`
			}
			
			err := json.Unmarshal([]byte(tc.json), &result)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}
			
			if result.Value.Value != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result.Value.Value)
			}
		})
	}
}