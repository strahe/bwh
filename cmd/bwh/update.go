package main

import (
	"context"
	"fmt"
	"time"

	"github.com/strahe/bwh/internal/updater"
	"github.com/urfave/cli/v3"
)

var updateCmd = &cli.Command{
	Name:  "update",
	Usage: "Check for updates and update BWH CLI to the latest version",
	Description: `Check for and install updates from GitHub releases.

Examples:
  bwh update                    # Check for updates and prompt for confirmation (5m timeout)
  bwh update --check            # Only check for updates, don't install (30s timeout)
  bwh update --force            # Update without confirmation prompt (5m timeout)
  bwh update --timeout 10m      # Update with custom 10-minute timeout
  bwh update --force -t 2m      # Force update with 2-minute timeout`,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "check",
			Aliases: []string{"c"},
			Usage:   "Only check for updates, don't install",
		},
		&cli.BoolFlag{
			Name:    "force",
			Aliases: []string{"f"},
			Usage:   "Update without confirmation prompt",
		},
		&cli.DurationFlag{
			Name:    "timeout",
			Aliases: []string{"t"},
			Usage:   "Timeout for update operations (e.g. 30s, 5m, 10m)",
			Value:   5 * time.Minute,
		},
	},
	Action: runUpdate,
}

func runUpdate(cliCtx context.Context, cmd *cli.Command) error {
	checkOnly := cmd.Bool("check")
	force := cmd.Bool("force")
	timeout := cmd.Duration("timeout")

	// Use shorter timeout for check-only operations
	if checkOnly {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(cliCtx, timeout)
	defer cancel()

	fmt.Printf("Checking for updates...\n")

	info, err := updater.CheckForUpdates(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	fmt.Printf("Current version: %s\n", info.CurrentVersion)
	fmt.Printf("Latest version:  %s\n", info.LatestVersion)

	if !info.HasUpdate {
		fmt.Printf("✅ You are already running the latest version!\n")
		return nil
	}

	fmt.Printf("🎉 New version available: %s\n", info.LatestVersion)
	fmt.Printf("Released: %s\n", info.ReleaseDate.Local().Format("2006-01-02 15:04:05"))
	if info.AssetSize > 0 {
		fmt.Printf("Download size: %.2f MB\n", float64(info.AssetSize)/(1024*1024))
	}

	if checkOnly {
		fmt.Printf("\nRun 'bwh update' to install the update.\n")
		return nil
	}

	// Prompt for confirmation unless --force is used
	if !force {
		fmt.Printf("\nDo you want to update to %s? [y/N]: ", info.LatestVersion)
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			fmt.Printf("Update cancelled.\n")
			return nil
		}

		if response != "y" && response != "Y" && response != "yes" && response != "YES" {
			fmt.Printf("Update cancelled.\n")
			return nil
		}
	}

	fmt.Printf("⬇️  Downloading %s... (timeout: %v)\n", info.LatestVersion, timeout)

	if err := updater.PerformUpdateWithTimeout(ctx, info, timeout); err != nil {
		return fmt.Errorf("failed to perform update: %w", err)
	}

	fmt.Printf("✅ Successfully updated to %s!\n", info.LatestVersion)
	fmt.Printf("Please restart your terminal or run 'bwh version' to verify the update.\n")

	return nil
}
