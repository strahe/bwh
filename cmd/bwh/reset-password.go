package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

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
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "yes",
			Usage:   "skip confirmation prompt",
			Aliases: []string{"y"},
		},
		&cli.StringFlag{
			Name:    "output",
			Usage:   "output password to specified file (creates random file if not specified)",
			Aliases: []string{"o"},
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		skipConfirm := cmd.Bool("yes")
		outputFile := cmd.String("output")

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		if !skipConfirm {
			if !confirmAction("reset root password", resolvedName) {
				fmt.Println("Operation cancelled.")
				return nil
			}
		}

		var filePath string
		if outputFile == "" {
			filePath = generateRandomFileName()
		} else {
			filePath = outputFile
		}
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			absPath = filePath
		}

		fmt.Printf("Resetting root password for instance: %s\n", resolvedName)

		result, err := bwhClient.ResetRootPassword(ctx)
		if err != nil {
			return fmt.Errorf("failed to reset root password: %w", err)
		}

		passwordContent := fmt.Sprintf("Root Password for BWH Instance: %s\n", resolvedName)
		passwordContent += fmt.Sprintf("Generated at: %s\n", time.Now().Format("2006-01-02 15:04:05 MST"))
		passwordContent += fmt.Sprintf("Password: %s\n", result.Password)

		err = os.WriteFile(filePath, []byte(passwordContent), 0600)
		if err != nil {
			return fmt.Errorf("failed to write password to file: %w", err)
		}

		fmt.Printf("\n✅ Root password reset successfully!\n")
		fmt.Printf("🔑 Password saved to: %s\n", absPath)

		return nil
	},
}
