package client

import (
	"encoding/json"
	"testing"
)

func TestFlexibleInt_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{
			name:    "integer value",
			input:   `123456`,
			want:    123456,
			wantErr: false,
		},
		{
			name:    "string value with valid number",
			input:   `"789012"`,
			want:    789012,
			wantErr: false,
		},
		{
			name:    "string value with zero",
			input:   `"0"`,
			want:    0,
			wantErr: false,
		},
		{
			name:    "negative integer",
			input:   `-123`,
			want:    -123,
			wantErr: false,
		},
		{
			name:    "negative string number",
			input:   `"-456"`,
			want:    -456,
			wantErr: false,
		},
		{
			name:    "invalid string value",
			input:   `"not-a-number"`,
			want:    0,
			wantErr: false, // FlexibleInt design: fallback to 0 without error
		},
		{
			name:    "null value",
			input:   `null`,
			want:    0,
			wantErr: false, // FlexibleInt design: fallback to 0 without error
		},
		{
			name:    "boolean value",
			input:   `true`,
			want:    0,
			wantErr: false, // FlexibleInt design: fallback to 0 without error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f FlexibleInt
			err := json.Unmarshal([]byte(tt.input), &f)

			if (err != nil) != tt.wantErr {
				t.Errorf("FlexibleInt.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && f.Value != tt.want {
				t.Errorf("FlexibleInt.UnmarshalJSON() value = %v, want %v", f.Value, tt.want)
			}
		})
	}
}

func TestBaseResponse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr int
		wantMsg string
	}{
		{
			name:    "success response",
			input:   `{"error": 0}`,
			wantErr: 0,
			wantMsg: "",
		},
		{
			name:    "error response with message",
			input:   `{"error": 1, "message": "API key invalid"}`,
			wantErr: 1,
			wantMsg: "API key invalid",
		},
		{
			name:    "error response without message",
			input:   `{"error": 2}`,
			wantErr: 2,
			wantMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp BaseResponse
			err := json.Unmarshal([]byte(tt.input), &resp)
			if err != nil {
				t.Errorf("BaseResponse unmarshal error = %v", err)
				return
			}

			if resp.Error != tt.wantErr {
				t.Errorf("BaseResponse.Error = %v, want %v", resp.Error, tt.wantErr)
			}

			if resp.Message != tt.wantMsg {
				t.Errorf("BaseResponse.Message = %v, want %v", resp.Message, tt.wantMsg)
			}
		})
	}
}

func TestServiceInfo_JSONParsing(t *testing.T) {
	// Test parsing a minimal ServiceInfo response
	input := `{
		"error": 0,
		"vm_type": "kvm",
		"hostname": "test-server",
		"plan": "test-plan",
		"os": "ubuntu-20.04",
		"email": "test@example.com",
		"plan_disk": 1073741824,
		"plan_ram": 2147483648,
		"data_counter": 1234567890,
		"ip_addresses": ["192.168.1.1", "2001:db8::/64"],
		"ssh_port": 22
	}`

	var info ServiceInfo
	err := json.Unmarshal([]byte(input), &info)
	if err != nil {
		t.Fatalf("ServiceInfo unmarshal error = %v", err)
	}

	if info.Error != 0 {
		t.Errorf("ServiceInfo.Error = %v, want 0", info.Error)
	}

	if info.VMType != "kvm" {
		t.Errorf("ServiceInfo.VMType = %v, want kvm", info.VMType)
	}

	if info.Hostname != "test-server" {
		t.Errorf("ServiceInfo.Hostname = %v, want test-server", info.Hostname)
	}

	// Test int64 fields parsed from string JSON
	if info.PlanRAM != 2147483648 {
		t.Errorf("ServiceInfo.PlanRAM = %v, want 2147483648", info.PlanRAM)
	}

	if info.DataCounter != 1234567890 {
		t.Errorf("ServiceInfo.DataCounter = %v, want 1234567890", info.DataCounter)
	}

	if len(info.IPAddresses) != 2 {
		t.Errorf("ServiceInfo.IPAddresses length = %v, want 2", len(info.IPAddresses))
	}
}
