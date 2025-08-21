package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/strahe/bwh/internal/progress"
	"github.com/strahe/bwh/pkg/client"
	"github.com/urfave/cli/v3"
)

var snapshotCmd = &cli.Command{
	Name:  "snapshot",
	Usage: "manage VPS snapshots",
	Commands: []*cli.Command{
		snapshotCreateCmd,
		snapshotListCmd,
		snapshotDeleteCmd,
		snapshotRestoreCmd,
		snapshotPinCmd,
		snapshotUnpinCmd,
		snapshotExportCmd,
		snapshotImportCmd,
		snapshotDownloadCmd,
	},
}

var snapshotCreateCmd = &cli.Command{
	Name:  "create",
	Usage: "create a snapshot",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "description",
			Aliases: []string{"d"},
			Usage:   "description for the snapshot",
		},
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "skip confirmation prompt",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		description := cmd.String("description")
		if description == "" {
			description = fmt.Sprintf("Created via bwh CLI on %s", time.Now().Format("2006-01-02 15:04:05"))
		}

		if !cmd.Bool("yes") {
			fmt.Printf("Creating snapshot for instance: %s\n", resolvedName)
			fmt.Printf("Description: %s\n", description)
			fmt.Printf("\n⚠️  WARNING: This operation will create a snapshot of the current VPS state.\n")
			fmt.Printf("The VPS will be AUTOMATICALLY RESTARTED and temporarily locked during snapshot creation.\n")
			fmt.Printf("All running processes will be terminated and services will be interrupted.\n")
			confirmed, err := promptConfirmation("Continue with snapshot creation and VPS restart?")
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Printf("Operation cancelled\n")
				return nil
			}
		}

		fmt.Printf("Creating snapshot for instance: %s\n", resolvedName)
		resp, err := bwhClient.CreateSnapshot(ctx, description)
		if err != nil {
			return fmt.Errorf("failed to create snapshot: %w", err)
		}

		fmt.Printf("✅ Snapshot creation initiated\n")
		if resp.NotificationEmail != "" {
			fmt.Printf("📧 Notification will be sent to: %s\n", resp.NotificationEmail)
		}

		return nil
	},
}

var snapshotListCmd = &cli.Command{
	Name:  "list",
	Usage: "list all snapshots",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "compact",
			Usage: "display snapshots in compact format",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Listing snapshots for instance: %s\n", resolvedName)

		resp, err := bwhClient.ListSnapshots(ctx)
		if err != nil {
			return fmt.Errorf("failed to list snapshots: %w", err)
		}

		if len(resp.Snapshots) == 0 {
			fmt.Printf("No snapshots found\n")
			return nil
		}

		if cmd.Bool("compact") {
			displaySnapshotsCompact(resp.Snapshots)
		} else {
			displaySnapshotsDetailed(resp.Snapshots)
		}

		return nil
	},
}

var snapshotDeleteCmd = &cli.Command{
	Name:      "delete",
	Usage:     "delete a snapshot",
	ArgsUsage: "<filename>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "skip confirmation prompt",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() != 1 {
			return fmt.Errorf("snapshot filename is required")
		}
		fileName := cmd.Args().First()

		if !cmd.Bool("yes") {
			confirmed, err := promptConfirmation(fmt.Sprintf("⚠️  Are you sure you want to delete snapshot '%s'? This cannot be undone.", fileName))
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Printf("Operation cancelled\n")
				return nil
			}
		}

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Deleting snapshot '%s' for instance: %s\n", fileName, resolvedName)

		if err := bwhClient.DeleteSnapshot(ctx, fileName); err != nil {
			return fmt.Errorf("failed to delete snapshot: %w", err)
		}

		fmt.Printf("✅ Snapshot '%s' deleted successfully\n", fileName)

		return nil
	},
}

var snapshotRestoreCmd = &cli.Command{
	Name:      "restore",
	Usage:     "restore a snapshot (WARNING: overwrites all data)",
	ArgsUsage: "<filename>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "skip confirmation prompt",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() != 1 {
			return fmt.Errorf("snapshot filename is required")
		}
		fileName := cmd.Args().First()

		if !cmd.Bool("yes") {
			fmt.Printf("⚠️  WARNING: Restoring snapshot '%s' will OVERWRITE ALL DATA on the VPS!\n", fileName)
			confirmed, err := promptConfirmation("This operation cannot be undone. Are you sure?")
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Printf("Operation cancelled\n")
				return nil
			}
		}

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Restoring snapshot '%s' for instance: %s\n", fileName, resolvedName)

		if err := bwhClient.RestoreSnapshot(ctx, fileName); err != nil {
			return fmt.Errorf("failed to restore snapshot: %w", err)
		}

		fmt.Printf("✅ Snapshot '%s' restoration initiated\n", fileName)

		return nil
	},
}

var snapshotPinCmd = &cli.Command{
	Name:      "pin",
	Usage:     "pin a snapshot (make it sticky - never purged)",
	ArgsUsage: "<filename_or_index>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "skip confirmation prompt",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() != 1 {
			return fmt.Errorf("snapshot filename or index is required")
		}
		return toggleSnapshotSticky(ctx, cmd, cmd.Args().First(), true)
	},
}

var snapshotUnpinCmd = &cli.Command{
	Name:      "unpin",
	Usage:     "unpin a snapshot (remove sticky - can be purged)",
	ArgsUsage: "<filename_or_index>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "skip confirmation prompt",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() != 1 {
			return fmt.Errorf("snapshot filename or index is required")
		}
		return toggleSnapshotSticky(ctx, cmd, cmd.Args().First(), false)
	},
}

var snapshotExportCmd = &cli.Command{
	Name:      "export",
	Usage:     "export a snapshot for transfer to another instance",
	ArgsUsage: "<filename>",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() != 1 {
			return fmt.Errorf("snapshot filename is required")
		}
		fileName := cmd.Args().First()

		bwhClient, instance, resolvedName, err := createBWHClientWithInstance(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Exporting snapshot '%s' for instance: %s\n", fileName, resolvedName)

		resp, err := bwhClient.ExportSnapshot(ctx, fileName)
		if err != nil {
			return fmt.Errorf("failed to export snapshot: %w", err)
		}

		fmt.Printf("✅ Snapshot export completed\n")
		fmt.Printf("\n📋 EXPORT DETAILS\n")
		fmt.Printf("   Source VEID  : %s\n", instance.VeID)
		fmt.Printf("   Source Token : %s\n", resp.Token)
		fmt.Printf("\n💡 Use these values with 'bwh snapshot import <source_veid> <source_token>' on the target instance\n")

		return nil
	},
}

var snapshotImportCmd = &cli.Command{
	Name:      "import",
	Usage:     "import a snapshot from another instance",
	ArgsUsage: "<source_veid> <source_token>",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() != 2 {
			return fmt.Errorf("source VEID and source token are required")
		}
		sourceVeid := cmd.Args().Get(0)
		sourceToken := cmd.Args().Get(1)

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Importing snapshot from VEID '%s' to instance: %s\n", sourceVeid, resolvedName)

		if err := bwhClient.ImportSnapshot(ctx, sourceVeid, sourceToken); err != nil {
			return fmt.Errorf("failed to import snapshot: %w", err)
		}

		fmt.Printf("✅ Snapshot import initiated successfully\n")

		return nil
	},
}

var snapshotDownloadCmd = &cli.Command{
	Name:      "download",
	Usage:     "download a snapshot file",
	ArgsUsage: "<filename_or_index> [output_path]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "output directory or filename",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() < 1 {
			return fmt.Errorf("snapshot filename/index is required")
		}
		identifier := cmd.Args().Get(0)

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		// Get snapshots to resolve identifier
		snapshotsResp, err := bwhClient.ListSnapshots(ctx)
		if err != nil {
			return fmt.Errorf("failed to list snapshots: %w", err)
		}

		var targetSnapshot *client.SnapshotInfo

		// Check if identifier is a number (index)
		if index, err := strconv.Atoi(identifier); err == nil {
			if index < 1 || index > len(snapshotsResp.Snapshots) {
				return fmt.Errorf("invalid snapshot index: %d (must be between 1 and %d)", index, len(snapshotsResp.Snapshots))
			}
			targetSnapshot = &snapshotsResp.Snapshots[index-1]
		} else {
			// Treat as filename
			for i, snapshot := range snapshotsResp.Snapshots {
				if snapshot.FileName == identifier {
					targetSnapshot = &snapshotsResp.Snapshots[i]
					break
				}
			}
			if targetSnapshot == nil {
				return fmt.Errorf("snapshot '%s' not found", identifier)
			}
		}

		// Check if download links are available
		if targetSnapshot.DownloadLink == "" && targetSnapshot.DownloadLinkSSL == "" {
			return fmt.Errorf("no download links available for snapshot '%s'", targetSnapshot.FileName)
		}

		// Determine download URL (prefer HTTPS)
		downloadURL := targetSnapshot.DownloadLinkSSL
		if downloadURL == "" {
			downloadURL = targetSnapshot.DownloadLink
			fmt.Printf("⚠️  Using HTTP download (HTTPS not available)\n")
		}

		// Determine output path
		var outputPath string
		if output := cmd.String("output"); output != "" {
			outputPath = output
		} else if cmd.Args().Len() > 1 {
			outputPath = cmd.Args().Get(1)
		} else {
			// Default to current directory with snapshot filename
			outputPath = targetSnapshot.FileName
		}

		// If outputPath is a directory, append the filename
		if stat, err := os.Stat(outputPath); err == nil && stat.IsDir() {
			outputPath = filepath.Join(outputPath, targetSnapshot.FileName)
		}

		// Show download info
		fmt.Printf("Downloading snapshot for instance '%s':\n", resolvedName)
		fmt.Printf("   File Name    : %s\n", targetSnapshot.FileName)
		fmt.Printf("   OS           : %s\n", targetSnapshot.OS)
		if targetSnapshot.Description != "" {
			description := decodeDescription(targetSnapshot.Description)
			fmt.Printf("   Description  : %s\n", description)
		}
		fmt.Printf("   Size         : %s\n", progress.FormatBytes(targetSnapshot.Size.Value))
		fmt.Printf("   Download URL : %s\n", downloadURL)
		fmt.Printf("   Output Path  : %s\n", outputPath)

		// Check if file already exists
		if _, err := os.Stat(outputPath); err == nil {
			confirmed, err := promptConfirmation(fmt.Sprintf("⚠️  File '%s' already exists. Overwrite?", outputPath))
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Printf("Download cancelled\n")
				return nil
			}
		}

		// Create output directory if needed
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Download the file with fallback
		fmt.Printf("\n🔽 Starting download...\n")
		if err := downloadFileWithFallback(ctx, targetSnapshot, outputPath); err != nil {
			return fmt.Errorf("download failed: %w", err)
		}

		fmt.Printf("✅ Download completed: %s\n", outputPath)
		return nil
	},
}

func displaySnapshotsDetailed(snapshots []client.SnapshotInfo) {
	fmt.Printf("\n📸 SNAPSHOTS\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════════════════════\n")

	for i, snapshot := range snapshots {
		fmt.Printf("\n📸 SNAPSHOT %d\n", i+1)
		fmt.Printf("   File Name    : %s\n", snapshot.FileName)
		fmt.Printf("   OS           : %s\n", snapshot.OS)
		if snapshot.Description != "" {
			description := decodeDescription(snapshot.Description)
			fmt.Printf("   Description  : %s\n", description)
		}
		fmt.Printf("   Size         : %s", progress.FormatBytes(snapshot.Size.Value))
		if snapshot.Uncompressed.Value > 0 {
			fmt.Printf(" (compressed from %s)", progress.FormatBytes(snapshot.Uncompressed.Value))
		}
		fmt.Printf("\n")
		fmt.Printf("   MD5 Hash     : %s\n", snapshot.MD5)
		if snapshot.Sticky {
			fmt.Printf("   Sticky       : ✅ Yes (never purged)\n")
		} else {
			fmt.Printf("   Sticky       : ❌ No\n")
			if snapshot.PurgesIn.Value > 0 {
				fmt.Printf("   Purges In    : %s\n", progress.FormatDuration(snapshot.PurgesIn.Value))
			}
		}
		if snapshot.DownloadLink != "" {
			fmt.Printf("   Download     : Available\n")
			fmt.Printf("     HTTP       : %s\n", snapshot.DownloadLink)
			if snapshot.DownloadLinkSSL != "" {
				fmt.Printf("     HTTPS      : %s\n", snapshot.DownloadLinkSSL)
			}
		}

		if i < len(snapshots)-1 {
			fmt.Printf("─────────────────────────────────────────────────────────────────────────────\n")
		}
	}
	fmt.Printf("\n")
}

func displaySnapshotsCompact(snapshots []client.SnapshotInfo) {
	fmt.Printf("\nSnapshots (%d):\n", len(snapshots))

	for _, snapshot := range snapshots {
		stickyIcon := "📌"
		if !snapshot.Sticky {
			stickyIcon = "  "
		}

		fmt.Printf("├─ %s %s (%s)\n", stickyIcon, snapshot.FileName, progress.FormatBytes(snapshot.Size.Value))
		fmt.Printf("│  ├─ OS: %s\n", snapshot.OS)
		if snapshot.Description != "" {
			description := decodeDescription(snapshot.Description)
			fmt.Printf("│  ├─ Description: %s\n", description)
		}
		if !snapshot.Sticky && snapshot.PurgesIn.Value > 0 {
			fmt.Printf("│  └─ Purges in: %s\n", progress.FormatDuration(snapshot.PurgesIn.Value))
		} else if snapshot.Sticky {
			fmt.Printf("│  └─ Sticky (never purged)\n")
		} else {
			fmt.Printf("│  └─ MD5: %s\n", snapshot.MD5)
		}
	}
	fmt.Printf("\n")
}

// decodeDescription attempts to decode base64 description, returns original if not base64
func decodeDescription(description string) string {
	if decoded, err := base64.StdEncoding.DecodeString(description); err == nil {
		// Check if decoded string contains only printable characters
		decodedStr := string(decoded)
		if isPrintableASCII(decodedStr) {
			return decodedStr
		}
	}
	return description
}

// isPrintableASCII checks if string contains only printable ASCII characters
func isPrintableASCII(s string) bool {
	for _, r := range s {
		if r < 32 || r > 126 {
			return false
		}
	}
	return true
}

// downloadFileWithFallback attempts to download using HTTPS first, then falls back to HTTP
func downloadFileWithFallback(ctx context.Context, snapshot *client.SnapshotInfo, outputPath string) error {
	// Try HTTPS first if available
	if snapshot.DownloadLinkSSL != "" {
		fmt.Printf("🔒 Attempting HTTPS download...\n")
		err := downloadFile(ctx, snapshot.DownloadLinkSSL, outputPath, snapshot.Size.Value)
		if err == nil {
			return nil
		}

		// Check if it's a TLS-related error
		if strings.Contains(err.Error(), "tls:") || strings.Contains(err.Error(), "handshake") {
			fmt.Printf("⚠️  HTTPS download failed due to TLS issues: %v\n", err)
			if snapshot.DownloadLink != "" {
				fmt.Printf("🔄 Falling back to HTTP download...\n")
				return downloadFile(ctx, snapshot.DownloadLink, outputPath, snapshot.Size.Value)
			}
		}
		return err
	}

	// Only HTTP available
	if snapshot.DownloadLink != "" {
		fmt.Printf("📡 Using HTTP download (HTTPS not available)\n")
		return downloadFile(ctx, snapshot.DownloadLink, outputPath, snapshot.Size.Value)
	}

	return fmt.Errorf("no download links available")
}

// downloadFile downloads a file from URL with progress indication
func downloadFile(ctx context.Context, downloadURL, filepath string, expectedSize int64) error {
	// Check if we need to disable TLS verification for IP-based HTTPS URLs
	skipTLSVerify := shouldSkipTLSVerify(downloadURL)

	// Create HTTP client with appropriate TLS settings
	tlsConfig := &tls.Config{
		InsecureSkipVerify: skipTLSVerify,
	}

	// For IP-based HTTPS URLs, use more permissive TLS settings
	if skipTLSVerify {
		tlsConfig.MinVersion = tls.VersionTLS12 // Only support secure TLS versions
		tlsConfig.MaxVersion = tls.VersionTLS13 // Support newest TLS versions
		tlsConfig.CipherSuites = nil            // Use default cipher suites
	}

	client := &http.Client{
		Timeout: 30 * time.Minute, // Set a reasonable timeout for large downloads
		Transport: &http.Transport{
			TLSClientConfig:     tlsConfig,
			DisableCompression:  true, // Avoid compression for large files
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 30 * time.Second,
		},
	}

	if skipTLSVerify {
		fmt.Printf("🔒 Using HTTPS with IP address (TLS verification disabled)\n")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to start download: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d %s", resp.StatusCode, resp.Status)
	}

	// Create output file
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if err := out.Close(); err != nil {
			fmt.Printf("Warning: failed to close output file: %v\n", err)
		}
	}()

	// Get file size from response or use expected size
	fileSize := resp.ContentLength
	if fileSize <= 0 {
		fileSize = expectedSize
	}

	// Create progress writer
	progressWriter := progress.NewWriter(fileSize)

	// Copy with progress
	_, err = io.Copy(out, progress.TeeReader(resp.Body, progressWriter))
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	// Final progress update
	progressWriter.Finish()

	return nil
}

// shouldSkipTLSVerify determines if TLS verification should be skipped for a URL
// Returns true only for HTTPS URLs with IP addresses as hostnames
func shouldSkipTLSVerify(downloadURL string) bool {
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return false
	}

	// Only consider HTTPS URLs
	if parsedURL.Scheme != "https" {
		return false
	}

	// Extract hostname (remove port if present)
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return false
	}

	// Check if hostname is an IP address
	return net.ParseIP(hostname) != nil
}

// toggleSnapshotSticky is a helper function to handle pin/unpin operations
func toggleSnapshotSticky(ctx context.Context, cmd *cli.Command, identifier string, sticky bool) error {
	bwhClient, resolvedName, err := createBWHClient(cmd)
	if err != nil {
		return err
	}

	// Get snapshots to resolve identifier
	snapshotsResp, err := bwhClient.ListSnapshots(ctx)
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	var targetSnapshot *client.SnapshotInfo
	var fileName string

	// Check if identifier is a number (index)
	if index, err := strconv.Atoi(identifier); err == nil {
		if index < 1 || index > len(snapshotsResp.Snapshots) {
			return fmt.Errorf("invalid snapshot index: %d (must be between 1 and %d)", index, len(snapshotsResp.Snapshots))
		}
		targetSnapshot = &snapshotsResp.Snapshots[index-1]
		fileName = targetSnapshot.FileName
	} else {
		// Treat as filename
		fileName = identifier
		for i, snapshot := range snapshotsResp.Snapshots {
			if snapshot.FileName == identifier {
				targetSnapshot = &snapshotsResp.Snapshots[i]
				break
			}
		}
		if targetSnapshot == nil {
			return fmt.Errorf("snapshot '%s' not found", identifier)
		}
	}

	// Show snapshot info for confirmation
	action := "unpin"
	newState := "will be subject to automatic purging"
	if sticky {
		action = "pin"
		newState = "will never be purged automatically"
	}

	fmt.Printf("Target snapshot for instance '%s':\n", resolvedName)
	fmt.Printf("   File Name    : %s\n", targetSnapshot.FileName)
	fmt.Printf("   OS           : %s\n", targetSnapshot.OS)
	if targetSnapshot.Description != "" {
		description := decodeDescription(targetSnapshot.Description)
		fmt.Printf("   Description  : %s\n", description)
	}
	fmt.Printf("   Size         : %s", progress.FormatBytes(targetSnapshot.Size.Value))
	if targetSnapshot.Uncompressed.Value > 0 {
		fmt.Printf(" (compressed from %s)", progress.FormatBytes(targetSnapshot.Uncompressed.Value))
	}
	fmt.Printf("\n")
	if targetSnapshot.Sticky {
		fmt.Printf("   Status       : 📌 Pinned (never purged)\n")
	} else {
		fmt.Printf("   Status       : 📌 Unpinned\n")
		if targetSnapshot.PurgesIn.Value > 0 {
			fmt.Printf("   Purges In    : %s\n", progress.FormatDuration(targetSnapshot.PurgesIn.Value))
		}
	}

	// Check if the operation is redundant
	if targetSnapshot.Sticky == sticky {
		if sticky {
			fmt.Printf("\n✅ Snapshot is already pinned (no change needed)\n")
		} else {
			fmt.Printf("\n✅ Snapshot is already unpinned (no change needed)\n")
		}
		return nil
	}

	if !cmd.Bool("yes") {
		fmt.Printf("\n⚠️  Are you sure you want to %s this snapshot?\n", action)
		fmt.Printf("After this change, the snapshot %s.\n", newState)
		confirmed, err := promptConfirmation("Continue?")
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Printf("Operation cancelled\n")
			return nil
		}
	}

	if err := bwhClient.ToggleSnapshotSticky(ctx, fileName, sticky); err != nil {
		return fmt.Errorf("failed to %s snapshot: %w", action, err)
	}

	if sticky {
		fmt.Printf("✅ Snapshot '%s' is now pinned (will never be purged)\n", fileName)
	} else {
		fmt.Printf("✅ Snapshot '%s' is now unpinned (subject to purging)\n", fileName)
	}

	return nil
}
