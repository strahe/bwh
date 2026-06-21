package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/strahe/bwh/pkg/client"
	"github.com/urfave/cli/v3"
)

var sshCmd = &cli.Command{
	Name:  "ssh",
	Usage: "manage SSH keys",
	Commands: []*cli.Command{
		{
			Name:  "list",
			Usage: "list SSH keys",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "full",
					Usage: "show full SSH keys instead of shortened versions",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				bwhClient, resolvedName, err := createBWHClient(cmd)
				if err != nil {
					return err
				}

				keys, err := bwhClient.GetSshKeys(ctx)
				if err != nil {
					return fmt.Errorf("failed to get SSH keys: %w", err)
				}

				fmt.Printf("SSH Keys for %s:\n\n", resolvedName)

				showFull := cmd.Bool("full")

				// VM-level keys
				fmt.Printf("VM-level keys (Hypervisor Vault):\n")
				if showFull {
					printKeys(keys.GetSshKeysVeidSlice())
				} else {
					printKeys(keys.GetShortenedSshKeysVeidSlice())
				}

				// Account-level keys
				fmt.Printf("\nAccount-level keys (Billing Portal):\n")
				if showFull {
					printKeys(keys.GetSshKeysUserSlice())
				} else {
					printKeys(keys.GetShortenedSshKeysUserSlice())
				}

				// Preferred keys (what will actually be used)
				fmt.Printf("\nKeys used during reinstallOS:\n")
				if showFull {
					printKeys(keys.GetSshKeysPreferredSlice())
				} else {
					printKeys(keys.GetShortenedSshKeysPreferredSlice())
				}

				if len(keys.GetSshKeysVeidSlice()) > 0 {
					fmt.Printf("\nNote: VM-level keys override account-level keys during reinstallOS.\n")
				}

				return nil
			},
		},
		{
			Name:      "set",
			Usage:     "set VM-level SSH keys (replaces all existing keys)",
			ArgsUsage: "<key1> [key2] [key3]...",
			Flags: writeFlags(
				&cli.StringFlag{
					Name:  "file",
					Usage: "read SSH keys from file (one per line)",
				},
			),
			Action: func(ctx context.Context, cmd *cli.Command) error {
				bwhClient, resolvedName, err := createBWHClient(cmd)
				if err != nil {
					return err
				}

				var sshKeys []string

				if filename := cmd.String("file"); filename != "" {
					// Read from file
					keys, err := readSshKeysFromFile(filename)
					if err != nil {
						return fmt.Errorf("failed to read SSH keys from file: %w", err)
					}
					sshKeys = keys
				} else {
					// Read from command line arguments
					sshKeys = cmd.Args().Slice()
				}

				if len(sshKeys) == 0 {
					return fmt.Errorf("no SSH keys provided")
				}

				return runUpdateSshKeys(ctx, bwhClient, resolvedName, sshKeys, cmd.Bool("dry-run"), skipConfirm(cmd), promptConfirmation)
			},
		},
		{
			Name:  "clear",
			Usage: "clear all VM-level SSH keys",
			Flags: writeFlags(),
			Action: func(ctx context.Context, cmd *cli.Command) error {
				bwhClient, resolvedName, err := createBWHClient(cmd)
				if err != nil {
					return err
				}

				return runUpdateSshKeys(ctx, bwhClient, resolvedName, []string{}, cmd.Bool("dry-run"), skipConfirm(cmd), promptConfirmation)
			},
		},
	},
}

type sshKeysAPI interface {
	GetSshKeys(context.Context) (*client.SshKeysResponse, error)
	UpdateSshKeys(context.Context, []string) error
}

func runUpdateSshKeys(ctx context.Context, api sshKeysAPI, resolvedName string, sshKeys []string, dryRun, skipConfirm bool, confirm confirmationFunc) error {
	for i, key := range sshKeys {
		if !isValidSshKey(key) {
			return fmt.Errorf("invalid SSH key format at position %d", i+1)
		}
	}

	current, err := api.GetSshKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to get SSH keys: %w", err)
	}
	currentKeys := current.GetSshKeysVeidSlice()
	if sameStringSlices(currentKeys, sshKeys) {
		fmt.Printf("✅ VM-level SSH keys are already in the requested state (no change needed)\n")
		return nil
	}

	fmt.Printf("VM-level SSH keys for %s: %d current -> %d requested\n", resolvedName, len(currentKeys), len(sshKeys))
	for i, key := range sshKeys {
		fmt.Printf("   Requested key %d: %s\n", i+1, maskSSHKey(key))
	}

	if dryRun {
		operation := "replace"
		if len(sshKeys) == 0 {
			operation = "clear"
		}
		printDryRun("updateSshKeys", resolvedName, fmt.Sprintf("operation: %s", operation), fmt.Sprintf("keys: %d -> %d", len(currentKeys), len(sshKeys)))
		return nil
	}

	prompt := fmt.Sprintf("Replace VM-level SSH keys for '%s'?", resolvedName)
	if len(sshKeys) == 0 {
		prompt = fmt.Sprintf("Clear all VM-level SSH keys for '%s'?", resolvedName)
	}
	confirmed, err := confirmWrite(prompt, skipConfirm, confirm)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	if len(sshKeys) == 0 {
		fmt.Printf("Clearing all VM-level SSH keys for %s...\n", resolvedName)
	} else {
		fmt.Printf("Setting %d SSH key(s) for %s...\n", len(sshKeys), resolvedName)
	}
	if err := api.UpdateSshKeys(ctx, sshKeys); err != nil {
		return fmt.Errorf("failed to update SSH keys: %w", err)
	}

	if len(sshKeys) == 0 {
		fmt.Printf("✅ VM-level SSH keys cleared successfully\n")
		fmt.Printf("Note: Account-level keys (if any) will still be used during reinstallOS.\n")
	} else {
		fmt.Printf("✅ SSH keys updated successfully\n")
		fmt.Printf("Note: Keys will be applied during the next reinstallOS operation.\n")
	}
	return nil
}

func printKeys(keys []string) {
	if len(keys) == 0 {
		fmt.Printf("  (none)\n")
		return
	}
	for i, key := range keys {
		fmt.Printf("  %d. %s\n", i+1, key)
	}
}

func readSshKeysFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close() // Ignore close error
	}()

	var keys []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			keys = append(keys, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}

func isValidSshKey(key string) bool {
	parts := strings.Fields(key)
	if len(parts) < 2 {
		return false
	}

	// Check if it starts with a known SSH key type
	keyType := parts[0]
	validTypes := []string{
		"ssh-rsa", "ssh-dss", "ssh-ed25519",
		"ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521",
		"sk-ecdsa-sha2-nistp256@openssh.com", "sk-ssh-ed25519@openssh.com",
	}

	for _, validType := range validTypes {
		if keyType == validType {
			return true
		}
	}

	return false
}
