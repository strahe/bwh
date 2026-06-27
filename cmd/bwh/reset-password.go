package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/strahe/bwh/pkg/client"
	"github.com/urfave/cli/v3"
)

func generateRandomFileName() (string, error) {
	result := make([]byte, 8)
	if _, err := rand.Read(result); err != nil {
		return "", fmt.Errorf("failed to generate output filename: %w", err)
	}
	return fmt.Sprintf("password_%x.txt", result), nil
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
		generatedPath, err := generateRandomFileName()
		if err != nil {
			return err
		}
		filePath = generatedPath
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

	output, err := openPasswordOutputFile(filePath, fileExists)
	if err != nil {
		return err
	}
	keepOutput := false
	defer func() {
		if !keepOutput {
			output.abort()
		}
	}()

	fmt.Printf("Resetting root password for instance: %s\n", resolvedName)

	result, err := api.ResetRootPassword(ctx)
	if err != nil {
		return fmt.Errorf("failed to reset root password: %w", err)
	}

	passwordContent := fmt.Sprintf("Root Password for BWH Instance: %s\n", resolvedName)
	passwordContent += fmt.Sprintf("Generated at: %s\n", time.Now().Format("2006-01-02 15:04:05 MST"))
	passwordContent += fmt.Sprintf("Password: %s\n", result.Password)

	if err := output.write(passwordContent); err != nil {
		return fmt.Errorf("failed to write password to file: %w", err)
	}
	keepOutput = true

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

type passwordOutputFile struct {
	file       *os.File
	targetPath string
	tempPath   string
	closed     bool
}

func openPasswordOutputFile(filePath string, fileExists bool) (*passwordOutputFile, error) {
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	file, err := os.CreateTemp(dir, "."+base+".tmp-")
	if err != nil {
		if fileExists {
			return nil, fmt.Errorf("failed to create temporary output file before resetting password: %w", err)
		}
		return nil, fmt.Errorf("failed to create output file before resetting password: %w", err)
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		if fileExists {
			return nil, fmt.Errorf("failed to prepare temporary output file before resetting password: %w", err)
		}
		return nil, fmt.Errorf("failed to prepare output file before resetting password: %w", err)
	}

	return &passwordOutputFile{file: file, targetPath: filePath, tempPath: file.Name()}, nil
}

func (o *passwordOutputFile) write(content string) error {
	if _, err := o.file.WriteString(content); err != nil {
		return err
	}
	if err := o.file.Close(); err != nil {
		o.closed = true
		return err
	}
	o.closed = true
	if err := os.Rename(o.tempPath, o.targetPath); err != nil {
		return err
	}
	return nil
}

func (o *passwordOutputFile) abort() {
	if o == nil {
		return
	}
	if !o.closed {
		_ = o.file.Close()
		o.closed = true
	}
	_ = os.Remove(o.tempPath)
}
