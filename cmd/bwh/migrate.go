package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/strahe/bwh/pkg/client"
	"github.com/urfave/cli/v3"
)

var migrateCmd = &cli.Command{
	Name:  "migrate",
	Usage: "migrate VPS to a different datacenter location (WARNING: IPv4 will change)",
	Commands: []*cli.Command{
		migrateLocationsCmd,
		migrateStartCmd,
	},
}

var migrateLocationsCmd = &cli.Command{
	Name:  "locations",
	Usage: "list possible migration locations",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Fetching migration locations for instance: %s\n", resolvedName)

		resp, err := bwhClient.GetMigrateLocations(ctx)
		if err != nil {
			return fmt.Errorf("failed to get migration locations: %w", err)
		}

		fmt.Printf("Current Location: %s\n\n", resp.CurrentLocation)

		// Sort locations for stable output
		locs := append([]string{}, resp.Locations...)
		sort.Strings(locs)

		fmt.Printf("Available Locations:\n")
		for _, id := range locs {
			desc := resp.Descriptions[id]
			mult := resp.DataTransferMultipliers[id]
			if desc == "" {
				desc = "(no description)"
			}
			fmt.Printf("  • %-10s  %s  (multiplier: %d)\n", id, desc, mult)
		}

		return nil
	},
}

func splitIPsByFamily(ips []string) (ipv4 []string, ipv6 []string) {
	for _, ip := range ips {
		if strings.Contains(ip, ":") {
			ipv6 = append(ipv6, ip)
		} else {
			ipv4 = append(ipv4, ip)
		}
	}
	return
}

var migrateStartCmd = &cli.Command{
	Name:      "start",
	Usage:     "start VPS migration to new location (IPv4 will be replaced)",
	ArgsUsage: "<location_id>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "skip confirmation prompt",
		},
		&cli.StringFlag{
			Name:  "timeout",
			Usage: "request timeout (e.g. 10m, 30m). Default: 15m",
			Value: "15m",
		},
		&cli.BoolFlag{
			Name:  "wait",
			Usage: "wait until VE unlocks and show live progress",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() != 1 {
			return fmt.Errorf("migrate start requires exactly one argument: <location_id>")
		}

		locationID := cmd.Args().Get(0)
		if locationID == "" {
			return fmt.Errorf("location_id cannot be empty")
		}

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		// Warn user and confirm unless --yes
		if !cmd.Bool("yes") {
			fmt.Printf("⚠️  Starting migration will REPLACE all IPv4 addresses of VPS '%s'.\n", resolvedName)
			fmt.Printf("⚠️  Downtime is expected during migration.\n")
			if !confirmAction("restart", resolvedName) { // reuse yes/no prompt semantics
				fmt.Println("Operation cancelled.")
				return nil
			}
		}

		// Parse timeout
		timeoutStr := cmd.String("timeout")
		d, err := time.ParseDuration(timeoutStr)
		if err != nil || d <= 0 {
			return fmt.Errorf("invalid timeout: %s", timeoutStr)
		}

		fmt.Printf("Starting migration to '%s' for instance: %s (timeout: %s)\n", locationID, resolvedName, d)

		wait := cmd.Bool("wait")
		if !wait {
			// Immediate return after API acceptance
			resp, err := bwhClient.StartMigrationWithTimeout(ctx, locationID, d)
			if err != nil {
				return fmt.Errorf("failed to start migration: %w", err)
			}
			fmt.Printf("\n✅ Migration task accepted\n")
			if resp.NotificationEmail != "" {
				fmt.Printf("Notification will be sent to: %s\n", resp.NotificationEmail)
			}
			if len(resp.NewIPs) > 0 {
				ipv4, ipv6 := splitIPsByFamily(resp.NewIPs)
				fmt.Printf("New IP addresses (after completion):\n")
				if len(ipv4) > 0 {
					fmt.Printf("  IPv4:\n")
					for _, ip := range ipv4 {
						fmt.Printf("    • %s\n", ip)
					}
				}
				if len(ipv6) > 0 {
					fmt.Printf("  IPv6:\n")
					for _, ip := range ipv6 {
						fmt.Printf("    • %s\n", ip)
					}
				}
			}
			return nil
		}

		// Wait mode: accept then poll until unlock
		migCtx, cancel := context.WithTimeout(ctx, d)
		defer cancel()

		resultCh := make(chan *client.MigrateStartResponse, 1)
		errCh := make(chan error, 1)

		go func() {
			resp, err := bwhClient.StartMigrationWithTimeout(migCtx, locationID, d)
			if err != nil {
				errCh <- err
				return
			}
			resultCh <- resp
		}()

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		lastPercent := -1
		lastMsg := ""
		lastOperation := ""
		var acceptResp *client.MigrateStartResponse

		for {
			select {
			case <-ticker.C:
				if resp, perr := bwhClient.GetMigrateLocations(ctx); perr != nil {
					if bwhErr, ok := client.GetBWHError(perr); ok && client.IsLockedError(perr) {
						if bwhErr.AdditionalErrorInfo != "" && bwhErr.AdditionalErrorInfo != lastOperation {
							fmt.Printf("%s\n", bwhErr.AdditionalErrorInfo)
							lastOperation = bwhErr.AdditionalErrorInfo
						}
						if info := bwhErr.AdditionalLockingInfo; info != nil {
							p := info.CompletedPercent
							msg := info.FriendlyProgressMessage
							updated := info.LastStatusUpdateSecondsAgo
							if p != lastPercent || msg != lastMsg {
								if updated > 0 {
									fmt.Printf("Progress: %d%% complete - %s (updated %ds ago)\n", p, msg, updated)
								} else {
									fmt.Printf("Progress: %d%% complete - %s\n", p, msg)
								}
								lastPercent = p
								lastMsg = msg
							}
						}
					}
				} else {
					fmt.Printf("\n✅ VE unlocked. Current location: %s\n", resp.CurrentLocation)
					if acceptResp != nil && len(acceptResp.NewIPs) > 0 {
						ipv4, ipv6 := splitIPsByFamily(acceptResp.NewIPs)
						fmt.Printf("New IP addresses (after completion):\n")
						if len(ipv4) > 0 {
							fmt.Printf("  IPv4:\n")
							for _, ip := range ipv4 {
								fmt.Printf("    • %s\n", ip)
							}
						}
						if len(ipv6) > 0 {
							fmt.Printf("  IPv6:\n")
							for _, ip := range ipv6 {
								fmt.Printf("    • %s\n", ip)
							}
						}
					}
					return nil
				}
			case resp := <-resultCh:
				acceptResp = resp
				fmt.Printf("\n✅ Migration task accepted\n")
				if resp.NotificationEmail != "" {
					fmt.Printf("Notification will be sent to: %s\n", resp.NotificationEmail)
				}
			case e := <-errCh:
				if client.IsLockedError(e) {
					continue
				}
				return fmt.Errorf("migration failed: %w", e)
			case <-migCtx.Done():
				return fmt.Errorf("migration timed out after %s", d)
			}
		}
	},
}
