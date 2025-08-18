package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

var resetPasswordCmd = &cli.Command{
	Name:  "reset-password",
	Usage: "reset the root password",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "yes",
			Usage:   "skip confirmation prompt",
			Aliases: []string{"y"},
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		skipConfirm := cmd.Bool("yes")

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

		fmt.Printf("Resetting root password for instance: %s\n", resolvedName)

		result, err := bwhClient.ResetRootPassword(ctx)
		if err != nil {
			return fmt.Errorf("failed to reset root password: %w", err)
		}

		fmt.Printf("\n✅ Root password reset successfully!\n")
		fmt.Printf("\n🔑 NEW ROOT PASSWORD: %s\n", result.Password)
		fmt.Printf("\n🔒 IMPORTANT: Please save this password securely!\n")
		fmt.Printf("   This password will not be shown again.\n")
		fmt.Printf("\n💡 To set a custom password, use SSH or the Interactive Root Shell.\n")

		return nil
	},
}
