package main

import (
	"context"
	"fmt"
	"net"
	"sort"
	"time"

	"github.com/strahe/bwh/pkg/client"
	"github.com/urfave/cli/v3"
)

var auditCmd = &cli.Command{
	Name:  "audit",
	Usage: "display audit log entries",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "compact",
			Usage: "display audit log in compact format",
		},
		&cli.IntFlag{
			Name:  "limit",
			Usage: "limit number of entries to display",
			Value: 10,
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		compact := cmd.Bool("compact")
		limit := cmd.Int("limit")

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Getting audit log for instance: %s\n", resolvedName)

		auditLog, err := bwhClient.GetAuditLog(ctx)
		if err != nil {
			return fmt.Errorf("failed to get audit log: %w", err)
		}

		if len(auditLog.LogEntries) == 0 {
			fmt.Printf("No audit log entries found for instance: %s\n", resolvedName)
			return nil
		}

		entries := auditLog.LogEntries
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Timestamp > entries[j].Timestamp
		})

		if limit > 0 && len(entries) > limit {
			entries = entries[:limit]
		}

		if compact {
			displayCompactAuditLog(entries, resolvedName)
		} else {
			displayDetailedAuditLog(entries, resolvedName)
		}

		return nil
	},
}

func displayDetailedAuditLog(entries []client.AuditLogEntry, instanceName string) {
	fmt.Printf("\n")
	fmt.Printf("┌─────────────────────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│                     Audit Log: %-35s │\n", instanceName)
	fmt.Printf("└─────────────────────────────────────────────────────────────────────────┘\n")

	fmt.Printf("\n📜 AUDIT LOG ENTRIES - %d entries\n", len(entries))

	for i, entry := range entries {
		timestamp := time.Unix(entry.Timestamp, 0).Local()
		ipAddr := intToIP(entry.RequestorIPv4)

		fmt.Printf("\n[%d] %s\n", i+1, timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("    IP:      %s\n", ipAddr)
		fmt.Printf("    Summary: %s\n", entry.Summary)

		if i < len(entries)-1 {
			fmt.Printf("    %s\n", "─────────────────────────────────────────────")
		}
	}
}

func displayCompactAuditLog(entries []client.AuditLogEntry, instanceName string) {
	fmt.Printf("\nAudit Log: %s\n", instanceName)
	fmt.Printf("├─ %d entries (newest first)\n", len(entries))

	for _, entry := range entries {
		timestamp := time.Unix(entry.Timestamp, 0).Local()
		ipAddr := intToIP(entry.RequestorIPv4)

		timeStr := timestamp.Format("01-02 15:04")
		fmt.Printf("├─ [%s] %-15s | %s\n", timeStr, ipAddr, entry.Summary)
	}

	fmt.Printf("└─ End of audit log\n")
}

func intToIP(ipInt uint32) string {
	ip := make(net.IP, 4)
	ip[0] = byte(ipInt >> 24)
	ip[1] = byte(ipInt >> 16)
	ip[2] = byte(ipInt >> 8)
	ip[3] = byte(ipInt)
	return ip.String()
}
