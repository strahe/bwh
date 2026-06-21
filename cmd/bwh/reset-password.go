package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/strahe/bwh/pkg/client"
	"github.com/urfave/cli/v3"
)

func generateRandomFileName() string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, 8)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return fmt.Sprintf("password_%s.txt", string(result))
}

var resetPasswordCmd = &cli.Command{
	Name:  "reset-password",
	Usage: "reset the root password",
	Flags: writeFlags(
		&cli.StringFlag{
			Name:    "output",
			Usage:   "output password to specified file (creates random file if not specified)",
			Aliases: []string{"o"},
		},
	),
	Action: func(ctx context.Context, cmd *cli.Command) error {
		outputFile := cmd.String("output")

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		return runResetPassword(ctx, bwhClient, resolvedName, outputFile, cmd.Bool("dry-run"), skipConfirm(cmd), promptConfirmation)
	},
}

type resetPasswordAPI interface {
	ResetRootPassword(context.Context) (*client.ResetRootPasswordResponse, error)
}

func runResetPassword(ctx context.Context, api resetPasswordAPI, resolvedName, outputFile string, dryRun, skipConfirm bool, confirm confirmationFunc) error {
	filePath := outputFile
	if filePath == "" {
		filePath = generateRandomFileName()
	}
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	fileExists, err := preflightPasswordOutput(filePath)
	if err != nil {
		return err
	}

	if dryRun {
		detail := fmt.Sprintf("output: %s", absPath)
		if fileExists {
			detail += " (would overwrite existing file)"
		}
		printDryRun("resetRootPassword", resolvedName, detail)
		return nil
	}

	if fileExists {
		confirmed, err := confirmWrite(fmt.Sprintf("Output file '%s' already exists. Overwrite?", filePath), skipConfirm, confirm)
		if err != nil {
			return err
		}
		if !confirmed {
			return nil
		}
	}

	confirmed, err := confirmWrite(fmt.Sprintf("Reset root password for VPS '%s'?", resolvedName), skipConfirm, confirm)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	fmt.Printf("Resetting root password for instance: %s\n", resolvedName)

	result, err := api.ResetRootPassword(ctx)
	if err != nil {
		return fmt.Errorf("failed to reset root password: %w", err)
	}

	passwordContent := fmt.Sprintf("Root Password for BWH Instance: %s\n", resolvedName)
	passwordContent += fmt.Sprintf("Generated at: %s\n", time.Now().Format("2006-01-02 15:04:05 MST"))
	passwordContent += fmt.Sprintf("Password: %s\n", result.Password)

	err = os.WriteFile(filePath, []byte(passwordContent), 0o600)
	if err != nil {
		return fmt.Errorf("failed to write password to file: %w", err)
	}

	fmt.Printf("\n✅ Root password reset successfully!\n")
	fmt.Printf("🔑 Password saved to: %s\n", absPath)

	return nil
}

func preflightPasswordOutput(filePath string) (bool, error) {
	info, err := os.Stat(filePath)
	if err == nil {
		if info.IsDir() {
			return false, fmt.Errorf("output path is a directory: %s", filePath)
		}
		file, err := os.OpenFile(filePath, os.O_WRONLY, 0)
		if err != nil {
			return false, fmt.Errorf("output file is not writable: %w", err)
		}
		if err := file.Close(); err != nil {
			return false, fmt.Errorf("failed to close output file: %w", err)
		}
		return true, nil
	}
	if !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to check output file: %w", err)
	}

	parent := filepath.Dir(filePath)
	info, err = os.Stat(parent)
	if err != nil {
		return false, fmt.Errorf("failed to check output directory: %w", err)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("output parent path is not a directory: %s", parent)
	}
	return false, nil
}
