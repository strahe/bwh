package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/strahe/bwh/pkg/client"
	"github.com/urfave/cli/v3"
)

var notificationsCmd = &cli.Command{
	Name:  "notifications",
	Usage: "inspect KiwiVM notification preferences",
	Commands: []*cli.Command{
		notificationsListCmd,
	},
}

var notificationsListCmd = &cli.Command{
	Name:  "list",
	Usage: "list KiwiVM notification preferences",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Getting notification preferences for instance: %s\n", resolvedName)
		resp, err := bwhClient.GetNotificationPreferences(ctx)
		if err != nil {
			return fmt.Errorf("failed to get notification preferences: %w", err)
		}

		displayNotificationPreferences(resp)
		return nil
	},
}

func displayNotificationPreferences(resp *client.NotificationPreferencesResponse) {
	fmt.Printf("\n📧 NOTIFICATION PREFERENCES\n")
	if resp.NotificationEmail != "" {
		fmt.Printf("   Email: %s\n", resp.NotificationEmail)
	}

	if len(resp.EmailPreferences) == 0 {
		fmt.Printf("\nNo notification preferences found.\n")
		return
	}

	categories := make([]string, 0, len(resp.EmailPreferences))
	for category := range resp.EmailPreferences {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	for _, category := range categories {
		prefs := resp.EmailPreferences[category]
		ids := make([]string, 0, len(prefs))
		for id := range prefs {
			ids = append(ids, id)
		}
		sort.Strings(ids)

		fmt.Printf("\n%s\n", category)
		for _, id := range ids {
			pref := prefs[id]
			fmt.Printf("  • %s\n", id)
			fmt.Printf("    Status     : %s\n", enabledStatus(pref.IsEnabled))
			if pref.FriendlyDescription != "" {
				fmt.Printf("    Description: %s\n", pref.FriendlyDescription)
			}
			if pref.ChangedTimestamp > 0 {
				fmt.Printf("    Updated    : %s\n", time.Unix(pref.ChangedTimestamp, 0).Format("2006-01-02 15:04:05"))
			}
			if pref.SValue != "" {
				fmt.Printf("    Value      : %s\n", pref.SValue)
			}
		}
	}
}

func enabledStatus(value int) string {
	if value == 1 {
		return "✅ Enabled"
	}
	return "❌ Disabled"
}
