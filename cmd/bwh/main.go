package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/strahe/bwh/internal/config"
	"github.com/strahe/bwh/internal/updater"
	"github.com/strahe/bwh/internal/version"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:                  "bwh",
		Usage:                 "manage your BWH instances",
		Version:               version.GetVersion(),
		EnableShellCompletion: true,
		ShellComplete:         shellComplete,
		Before:                showUpdateNotificationHook,
		After:                 checkForUpdatesHook,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "path to config file",
				Aliases: []string{"c"},
			},
			&cli.StringFlag{
				Name:    "instance",
				Usage:   "BWH instance to use",
				Aliases: []string{"i"},
			},
		},
		Commands: []*cli.Command{
			nodeCmd,
			infoCmd,
			rateLimitCmd,
			connectCmd,
			sshCmd,
			startCmd,
			stopCmd,
			restartCmd,
			killCmd,
			hostnameCmd,
			setPTRCmd,
			isoCmd,
			reinstallCmd,
			usageCmd,
			auditCmd,
			resetPasswordCmd,
			snapshotCmd,
			backupCmd,
			migrateCmd,
			ipv6Cmd,
			privateIPCmd,
			mcpCmd,
			updateCmd,
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func shellComplete(ctx context.Context, cmd *cli.Command) {
	args := os.Args

	// Check if completing instance flag value
	for i, arg := range args {
		if (arg == "--instance" || arg == "-i") && i+1 < len(args) {
			configManager, err := config.NewManager(cmd.String("config"))
			if err != nil {
				return
			}
			for _, instance := range configManager.GetAvailableInstances() {
				fmt.Println(instance)
			}
			return
		}
	}
}

func showUpdateNotificationHook(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	if len(os.Args) > 1 && os.Args[1] == "update" {
		return ctx, nil
	}
	showCachedUpdateNotification()
	return ctx, nil
}

func checkForUpdatesHook(ctx context.Context, cmd *cli.Command) error {
	if len(os.Args) > 1 && os.Args[1] == "update" {
		return nil
	}

	if !shouldCheckForUpdates() {
		return nil
	}

	updateLastCheckTime()

	checkCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if info, err := updater.CheckForUpdatesWithTimeout(checkCtx, 2*time.Second); err == nil && info.HasUpdate {
		cacheUpdateInfo(info)
	}

	return nil
}

func shouldCheckForUpdates() bool {
	lastCheckFile := getLastCheckFilePath()
	if stat, err := os.Stat(lastCheckFile); err == nil {
		if time.Since(stat.ModTime()) < 24*time.Hour {
			return false
		}
	}
	return true
}

func showCachedUpdateNotification() {
	updateCacheFile := getUpdateCacheFilePath()
	data, err := os.ReadFile(updateCacheFile)
	if err != nil {
		return
	}

	var info updater.UpdateInfo
	if err := json.Unmarshal(data, &info); err != nil {
		os.Remove(updateCacheFile)
		return
	}

	// Check if user has already upgraded
	currentVersion := version.GetVersion()
	if updater.CompareVersions(currentVersion, info.LatestVersion) >= 0 {
		os.Remove(updateCacheFile) // User has upgraded, remove cache
		return
	}

	fmt.Fprintf(os.Stderr, "\n┌─────────────────────────────────────────────────────────────┐\n")
	fmt.Fprintf(os.Stderr, "│ 🎉 BWH CLI %s is available! Current: %-15s    │\n", info.LatestVersion, info.CurrentVersion)
	fmt.Fprintf(os.Stderr, "│    Run 'bwh update' to upgrade                              │\n")
	fmt.Fprintf(os.Stderr, "└─────────────────────────────────────────────────────────────┘\n\n")
}

func cacheUpdateInfo(info *updater.UpdateInfo) {
	updateCacheFile := getUpdateCacheFilePath()
	if err := os.MkdirAll(filepath.Dir(updateCacheFile), 0755); err != nil {
		return
	}
	data, err := json.Marshal(info)
	if err != nil {
		return
	}
	os.WriteFile(updateCacheFile, data, 0644) //nolint:errcheck
}

func updateLastCheckTime() {
	lastCheckFile := getLastCheckFilePath()
	if err := os.MkdirAll(filepath.Dir(lastCheckFile), 0755); err != nil {
		return
	}
	os.WriteFile(lastCheckFile, []byte(time.Now().Format(time.RFC3339)), 0644) //nolint:errcheck
}

func getUpdateCacheFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".bwh", ".update_available")
}

func getLastCheckFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".bwh", ".last_check")
}
