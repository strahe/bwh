package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateInstance(t *testing.T) {
	tests := []struct {
		name     string
		instance *Instance
		wantErr  bool
		errType  error
	}{
		{
			name: "valid instance",
			instance: &Instance{
				APIKey: "valid-api-key-123456789",
				VeID:   "123456",
			},
			wantErr: false,
		},
		{
			name: "empty API key",
			instance: &Instance{
				APIKey: "",
				VeID:   "123456",
			},
			wantErr: true,
			errType: ErrInvalidAPIKey,
		},
		{
			name: "API key too short",
			instance: &Instance{
				APIKey: "short",
				VeID:   "123456",
			},
			wantErr: true,
			errType: ErrInvalidAPIKey,
		},
		{
			name: "API key with whitespace",
			instance: &Instance{
				APIKey: "key with space",
				VeID:   "123456",
			},
			wantErr: true,
			errType: ErrInvalidAPIKey,
		},
		{
			name: "VeID too long",
			instance: &Instance{
				APIKey: "valid-api-key-123456789",
				VeID:   strings.Repeat("1", 50),
			},
			wantErr: true,
			errType: ErrInvalidVeID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInstance(tt.instance)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateInstance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != tt.errType {
				t.Errorf("validateInstance() error = %v, want %v", err, tt.errType)
			}
		})
	}
}

func TestConfigFileSecurity(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	testInstance := &Instance{
		APIKey: "test-api-key-123456789",
		VeID:   "123456",
	}

	err = manager.AddInstance("test-instance", testInstance, true)
	if err != nil {
		t.Fatalf("AddInstance() error = %v", err)
	}

	// Check file permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("Config file permissions = %o, want %o", info.Mode().Perm(), 0o600)
	}

	dirInfo, err := os.Stat(filepath.Dir(configPath))
	if err != nil {
		t.Fatalf("Failed to stat config directory: %v", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Errorf("Config directory permissions = %o, want %o", dirInfo.Mode().Perm(), 0o700)
	}
}

func TestInstanceResolution(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Test with no instances
	_, _, err = manager.ResolveInstance("")
	if err != ErrNoInstances {
		t.Errorf("ResolveInstance() with no instances error = %v, want %v", err, ErrNoInstances)
	}

	testInstance1 := &Instance{APIKey: "key1-123456789", VeID: "123456"}
	testInstance2 := &Instance{APIKey: "key2-123456789", VeID: "654321"}

	err = manager.AddInstance("instance1", testInstance1, true)
	if err != nil {
		t.Fatalf("AddInstance() error = %v", err)
	}
	err = manager.AddInstance("instance2", testInstance2, false)
	if err != nil {
		t.Fatalf("AddInstance() error = %v", err)
	}

	instance, name, err := manager.ResolveInstance("instance2")
	if err != nil {
		t.Fatalf("ResolveInstance() error = %v", err)
	}
	if name != "instance2" {
		t.Errorf("ResolveInstance() name = %v, want instance2", name)
	}
	if instance.VeID != "654321" {
		t.Errorf("ResolveInstance() wrong instance returned")
	}

	_, name, err = manager.ResolveInstance("")
	if err != nil {
		t.Fatalf("ResolveInstance() error = %v", err)
	}
	if name != "instance1" {
		t.Errorf("ResolveInstance() name = %v, want instance1 (default)", name)
	}
}
