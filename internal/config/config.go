package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/strahe/bwh/pkg/client"
)

var (
	ErrInstanceNotFound  = errors.New("instance not found")
	ErrInstanceExists    = errors.New("instance already exists")
	ErrNoInstances       = errors.New("no instances configured")
	ErrNoDefaultInstance = errors.New("no default instance set")
	ErrInvalidAPIKey     = errors.New("invalid API key format")
	ErrInvalidVeID       = errors.New("invalid VeID format")
)

// Config represents the BWH CLI configuration
type Config struct {
	DefaultInstance string               `yaml:"default_instance,omitempty"`
	Instances       map[string]*Instance `yaml:"instances"`
}

// Instance represents a BWH VPS instance configuration
type Instance struct {
	APIKey      string   `yaml:"api_key"`
	VeID        string   `yaml:"veid"`
	Description string   `yaml:"description,omitempty"`
	Endpoint    string   `yaml:"endpoint,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
}

// Manager handles configuration operations
type Manager struct {
	configPath string
	config     *Config
}

// NewManager creates a new configuration manager
func NewManager(configPath string) (*Manager, error) {
	if configPath == "" {
		var err error
		configPath, err = getDefaultConfigPath()
		if err != nil {
			return nil, fmt.Errorf("failed to get default config path: %w", err)
		}
	}

	m := &Manager{
		configPath: configPath,
		config:     &Config{Instances: make(map[string]*Instance)},
	}

	// Try to load existing config
	if err := m.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return m, nil
}

// getDefaultConfigPath returns the default configuration file path
func getDefaultConfigPath() (string, error) {
	// Check environment variable first
	if path := os.Getenv("BWH_CONFIG_PATH"); path != "" {
		return path, nil
	}

	// Use default ~/.bwh/config.yaml
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(homeDir, ".bwh", "config.yaml"), nil
}

// Load loads the configuration from file
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, m.config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	if m.config.Instances == nil {
		m.config.Instances = make(map[string]*Instance)
	}

	return nil
}

// Save saves the configuration to file with secure permissions
func (m *Manager) Save() error {
	// Create directory if it doesn't exist with secure permissions
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Ensure directory has correct permissions (in case it already existed)
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("failed to set directory permissions: %w", err)
	}

	data, err := yaml.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with restricted permissions (600)
	if err := os.WriteFile(m.configPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// AddInstance adds a new instance to the configuration
func (m *Manager) AddInstance(name string, instance *Instance, setDefault bool) error {
	if err := validateInstanceName(name); err != nil {
		return err
	}

	if err := validateInstance(instance); err != nil {
		return err
	}

	if _, exists := m.config.Instances[name]; exists {
		return fmt.Errorf("%w: %s", ErrInstanceExists, name)
	}

	m.config.Instances[name] = instance

	// Set as default if requested or if it's the first instance
	if setDefault || len(m.config.Instances) == 1 {
		m.config.DefaultInstance = name
	}

	return m.Save()
}

// RemoveInstance removes an instance from the configuration
func (m *Manager) RemoveInstance(name string) error {
	if _, exists := m.config.Instances[name]; !exists {
		return fmt.Errorf("%w: %s", ErrInstanceNotFound, name)
	}

	delete(m.config.Instances, name)

	// Clear default if this was the default instance
	if m.config.DefaultInstance == name {
		m.config.DefaultInstance = ""
		// Auto-set new default if there's only one instance left
		if len(m.config.Instances) == 1 {
			for instanceName := range m.config.Instances {
				m.config.DefaultInstance = instanceName
				break
			}
		}
	}

	return m.Save()
}

// SetDefault sets the default instance
func (m *Manager) SetDefault(name string) error {
	if _, exists := m.config.Instances[name]; !exists {
		return fmt.Errorf("%w: %s", ErrInstanceNotFound, name)
	}

	m.config.DefaultInstance = name
	return m.Save()
}

// GetInstance returns the configuration for a specific instance
func (m *Manager) GetInstance(name string) (*Instance, error) {
	instance, exists := m.config.Instances[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrInstanceNotFound, name)
	}
	return instance, nil
}

// ListInstances returns all configured instances
func (m *Manager) ListInstances() map[string]*Instance {
	return m.config.Instances
}

// GetDefaultInstance returns the default instance name
func (m *Manager) GetDefaultInstance() string {
	return m.config.DefaultInstance
}

// ResolveInstance resolves the instance to use based on priority:
// 1. Explicit instance name parameter
// 2. BWH_INSTANCE environment variable
// 3. Default instance from config
// 4. If only one instance exists, use it
func (m *Manager) ResolveInstance(instanceName string) (*Instance, string, error) {
	if len(m.config.Instances) == 0 {
		return nil, "", ErrNoInstances
	}

	// Priority 1: Explicit instance name
	if instanceName != "" {
		instance, err := m.GetInstance(instanceName)
		return instance, instanceName, err
	}

	// Priority 2: Environment variable
	if envInstance := os.Getenv("BWH_INSTANCE"); envInstance != "" {
		instance, err := m.GetInstance(envInstance)
		return instance, envInstance, err
	}

	// Priority 3: Default instance
	if m.config.DefaultInstance != "" {
		instance, err := m.GetInstance(m.config.DefaultInstance)
		return instance, m.config.DefaultInstance, err
	}

	// Priority 4: Single instance
	if len(m.config.Instances) == 1 {
		for name, instance := range m.config.Instances {
			return instance, name, nil
		}
	}

	return nil, "", ErrNoDefaultInstance
}

// ValidateInstance validates an instance by testing the API connection
func (m *Manager) ValidateInstance(instanceName string) error {
	instance, err := m.GetInstance(instanceName)
	if err != nil {
		return err
	}

	client := client.NewClient(instance.APIKey, instance.VeID)
	if instance.Endpoint != "" {
		client.SetBaseURL(instance.Endpoint)
	}

	ctx := context.Background()
	_, err = client.GetServiceInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate instance connection: %w", err)
	}

	return nil
}

// GetAvailableInstances returns a list of available instance names
func (m *Manager) GetAvailableInstances() []string {
	names := make([]string, 0, len(m.config.Instances))
	for name := range m.config.Instances {
		names = append(names, name)
	}
	return names
}

func validateInstanceName(name string) error {
	if name == "" {
		return errors.New("instance name cannot be empty")
	}
	if strings.ContainsAny(name, " \t\n\r") {
		return errors.New("instance name cannot contain whitespace")
	}
	return nil
}

func validateInstance(instance *Instance) error {
	if instance.APIKey == "" {
		return ErrInvalidAPIKey
	}
	if instance.VeID == "" {
		return ErrInvalidVeID
	}

	// Enhanced API key validation
	if len(instance.APIKey) < 10 || len(instance.APIKey) > 256 {
		return ErrInvalidAPIKey
	}

	// Check for common patterns that might indicate invalid keys
	if strings.Contains(instance.APIKey, " ") ||
		strings.Contains(instance.APIKey, "\t") ||
		strings.Contains(instance.APIKey, "\n") {
		return ErrInvalidAPIKey
	}

	// Basic VeID validation (should be numeric or alphanumeric)
	if len(instance.VeID) == 0 || len(instance.VeID) > 32 {
		return ErrInvalidVeID
	}

	return nil
}
