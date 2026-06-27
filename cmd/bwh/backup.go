package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/strahe/bwh/pkg/client"
	"github.com/urfave/cli/v3"
)

var backupCmd = &cli.Command{
	Name:  "backup",
	Usage: "manage VPS backups",
	Commands: []*cli.Command{
		backupListCmd,
		backupCopyToSnapshotCmd,
	},
}

var backupListCmd = &cli.Command{
	Name:  "list",
	Usage: "list all backups",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "compact",
			Usage: "display backups in compact format",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Listing backups for instance: %s\n", resolvedName)

		resp, err := bwhClient.ListBackups(ctx)
		if err != nil {
			return fmt.Errorf("failed to list backups: %w", err)
		}

		if len(resp.Backups) == 0 {
			fmt.Printf("No backups found\n")
			return nil
		}

		// Convert map to slice for sorting
		var backups []client.BackupInfo
		for token, backup := range resp.Backups {
			backup.Token = token
			backups = append(backups, backup)
		}

		// Sort by timestamp (newest first)
		sort.Slice(backups, func(i, j int) bool {
			return backups[i].Timestamp > backups[j].Timestamp
		})

		if cmd.Bool("compact") {
			displayBackupsCompact(backups)
		} else {
			displayBackupsDetailed(backups)
		}

		return nil
	},
}

func displayBackupsDetailed(backups []client.BackupInfo) {
	fmt.Printf("\n💾 BACKUPS\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════════════════════\n")

	for i, backup := range backups {
		fmt.Printf("\n💾 BACKUP %d\n", i+1)
		fmt.Printf("   Token        : %s\n", backup.Token)
		fmt.Printf("   OS           : %s\n", backup.OS)
		fmt.Printf("   Size         : %s\n", formatBytes(backup.Size))
		fmt.Printf("   MD5 Hash     : %s\n", backup.MD5)
		fmt.Printf("   Created      : %s\n", time.Unix(backup.Timestamp, 0).Format("2006-01-02 15:04:05"))

		if i < len(backups)-1 {
			fmt.Printf("─────────────────────────────────────────────────────────────────────────────\n")
		}
	}
	fmt.Printf("\n")
}

func displayBackupsCompact(backups []client.BackupInfo) {
	fmt.Printf("\nBackups (%d):\n", len(backups))

	for _, backup := range backups {
		createdTime := time.Unix(backup.Timestamp, 0).Format("2006-01-02 15:04")
		fmt.Printf("├─ %s (%s)\n", backup.Token, formatBytes(backup.Size))
		fmt.Printf("│  ├─ OS: %s\n", backup.OS)
		fmt.Printf("│  └─ Created: %s\n", createdTime)
	}
	fmt.Printf("\n")
}

var backupCopyToSnapshotCmd = &cli.Command{
	Name:      "copy-to-snapshot",
	Aliases:   []string{"cts"},
	Usage:     "copy a backup to a restorable snapshot",
	ArgsUsage: "<backup_token>",
	Flags:     writeFlags(),
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() != 1 {
			return fmt.Errorf("backup token is required")
		}
		backupToken := cmd.Args().First()

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		return runBackupCopyToSnapshot(ctx, bwhClient, resolvedName, backupToken, cmd.Bool("dry-run"), skipConfirm(cmd), promptConfirmation)
	},
}

type backupCopyAPI interface {
	ListBackups(context.Context) (*client.BackupListResponse, error)
	CopyBackupToSnapshot(context.Context, string) error
}

func runBackupCopyToSnapshot(ctx context.Context, api backupCopyAPI, resolvedName, backupToken string, dryRun, skipConfirm bool, confirm confirmationFunc) error {
	if err := validateBackupToken(backupToken); err != nil {
		return err
	}

	backupsResp, err := api.ListBackups(ctx)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	backup, exists := backupsResp.Backups[backupToken]
	if !exists {
		return fmt.Errorf("backup with token '%s' not found", maskSensitive(backupToken))
	}

	fmt.Printf("Target backup for instance '%s':\n", resolvedName)
	fmt.Printf("   Token        : %s\n", maskSensitive(backupToken))
	fmt.Printf("   OS           : %s\n", backup.OS)
	fmt.Printf("   Size         : %s\n", formatBytes(backup.Size))
	fmt.Printf("   MD5 Hash     : %s\n", backup.MD5)
	fmt.Printf("   Created      : %s\n", time.Unix(backup.Timestamp, 0).Format("2006-01-02 15:04:05"))

	if dryRun {
		printDryRun("backup/copyToSnapshot", resolvedName, fmt.Sprintf("backupToken: %s", maskSensitive(backupToken)))
		return nil
	}

	if !skipConfirm {
		fmt.Printf("\n⚠️  Are you sure you want to copy this backup to a snapshot?\n")
		fmt.Printf("This will create a new restorable snapshot from the backup.\n")
	}
	confirmed, err := confirmWrite("Continue?", skipConfirm, confirm)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	fmt.Printf("\nCopying backup to snapshot for instance: %s\n", resolvedName)

	if err := api.CopyBackupToSnapshot(ctx, backupToken); err != nil {
		return fmt.Errorf("failed to copy backup to snapshot: %w", err)
	}

	fmt.Printf("✅ Backup successfully copied to snapshot\n")
	fmt.Printf("💡 Use 'bwh snapshot list' to see the new snapshot\n")

	return nil
}
