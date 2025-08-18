package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/strahe/bwh/internal/config"
	"github.com/strahe/bwh/pkg/client"
	"github.com/urfave/cli/v3"
)

// printJSON prints an object as formatted JSON
func printJSON(obj any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(obj)
}

// getConfigManager creates a new config manager with error handling
func getConfigManager(configPath string) (*config.Manager, error) {
	manager, err := config.NewManager(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize config manager: %w", err)
	}
	return manager, nil
}

// resolveInstanceWithFallback resolves the instance to use with helpful error messages
func resolveInstanceWithFallback(manager *config.Manager, instanceName string) (*config.Instance, string, error) {
	instance, resolvedName, err := manager.ResolveInstance(instanceName)
	if err != nil {
		switch err {
		case config.ErrNoInstances:
			return nil, "", fmt.Errorf("no instances configured. Run 'bwh node add <name>' to add one")
		case config.ErrNoDefaultInstance:
			availableInstances := manager.GetAvailableInstances()
			return nil, "", fmt.Errorf("no default instance set and multiple instances available: %v. Use --instance flag or run 'bwh node set-default <name>'", availableInstances)
		case config.ErrInstanceNotFound:
			availableInstances := manager.GetAvailableInstances()
			return nil, "", fmt.Errorf("instance not found. Available instances: %v", availableInstances)
		default:
			return nil, "", err
		}
	}
	return instance, resolvedName, nil
}

// formatBytes converts bytes to human readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// validateBackupToken validates the format of a backup token
func validateBackupToken(token string) error {
	// Backup tokens are 40-character hexadecimal strings
	if len(token) != 40 {
		return fmt.Errorf("invalid backup token format: expected 40 characters, got %d", len(token))
	}

	matched, err := regexp.MatchString("^[a-f0-9]{40}$", token)
	if err != nil {
		return fmt.Errorf("failed to validate backup token: %w", err)
	}

	if !matched {
		return fmt.Errorf("invalid backup token format: must be 40 hexadecimal characters")
	}

	return nil
}

// promptConfirmation prompts user for yes/no confirmation with better error handling
func promptConfirmation(prompt string) (bool, error) {
	fmt.Printf("%s [y/N]: ", prompt)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			fmt.Printf("\n")
			return false, fmt.Errorf("operation cancelled (EOF)")
		}
		return false, fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

// createConfigManager creates a config manager from CLI command flags.
// It extracts the config path from the command and initializes a new config.Manager.
// Returns the configured manager and any error encountered during initialization.
func createConfigManager(cmd *cli.Command) (*config.Manager, error) {
	configPath := cmd.String("config")
	return getConfigManager(configPath)
}

// createBWHClient creates a BWH API client with configuration resolution.
// It handles config loading, instance resolution, and client setup in one call.
// This is the primary helper function for most VPS operations that only need the client.
//
// Returns:
//   - Configured BWH client ready for API calls
//   - Resolved instance name for user feedback
//   - Error if configuration or client setup fails
func createBWHClient(cmd *cli.Command) (*client.Client, string, error) {
	bwhClient, _, resolvedName, err := createBWHClientWithInstance(cmd)
	return bwhClient, resolvedName, err
}

// createBWHClientWithInstance creates a BWH API client with configuration resolution.
// Similar to createBWHClient but also returns the resolved instance configuration.
// Use this when commands need access to instance-specific settings beyond the client.
//
// Returns:
//   - Configured BWH client ready for API calls
//   - Instance configuration with API key, VeID, endpoint, etc.
//   - Resolved instance name for user feedback
//   - Error if configuration or client setup fails
func createBWHClientWithInstance(cmd *cli.Command) (*client.Client, *config.Instance, string, error) {
	manager, err := createConfigManager(cmd)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create config manager: %w", err)
	}

	instanceName := cmd.String("instance")
	instance, resolvedName, err := resolveInstanceWithFallback(manager, instanceName)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to resolve instance: %w", err)
	}

	bwhClient := client.NewClient(instance.APIKey, instance.VeID)
	if instance.Endpoint != "" {
		bwhClient.SetBaseURL(instance.Endpoint)
	}

	return bwhClient, instance, resolvedName, nil
}
