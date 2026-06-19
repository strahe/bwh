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

var notificationsCmd = &cli.Command{
	Name:  "notifications",
	Usage: "inspect KiwiVM notification preferences",
	Commands: []*cli.Command{
		notificationsListCmd,
		notificationsSetCmd,
	},
}

var notificationsWriteFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:    "yes",
		Aliases: []string{"y"},
		Usage:   "skip confirmation prompt",
	},
	&cli.BoolFlag{
		Name:  "dry-run",
		Usage: "validate and show the write action without calling the write API",
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

var notificationsSetCmd = &cli.Command{
	Name:      "set",
	Usage:     "set a KiwiVM notification preference",
	ArgsUsage: "<preference_id> <on|off>",
	Flags:     notificationsWriteFlags,
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() != 2 {
			return fmt.Errorf("notifications set requires exactly two arguments: <preference_id> <on|off>")
		}
		preferenceID := strings.TrimSpace(cmd.Args().Get(0))
		if preferenceID == "" {
			return fmt.Errorf("notification preference id cannot be empty")
		}
		enabled, err := parseNotificationState(cmd.Args().Get(1))
		if err != nil {
			return err
		}

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		return runNotificationSet(ctx, bwhClient, resolvedName, preferenceID, enabled, cmd.Bool("dry-run"), cmd.Bool("yes"), promptConfirmation)
	},
}

type notificationAPI interface {
	GetNotificationPreferences(context.Context) (*client.NotificationPreferencesResponse, error)
	SetNotificationPreferences(context.Context, map[string]bool) (*client.SetNotificationPreferencesResponse, error)
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

func parseNotificationState(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "on", "true", "1", "enable", "enabled":
		return true, nil
	case "off", "false", "0", "disable", "disabled":
		return false, nil
	default:
		return false, fmt.Errorf("notification state must be one of: on, off, true, false, 1, 0, enable, disable, enabled, disabled")
	}
}

func runNotificationSet(
	ctx context.Context,
	api notificationAPI,
	resolvedName string,
	preferenceID string,
	enabled bool,
	dryRun bool,
	skipConfirm bool,
	confirm confirmationFunc,
) error {
	fmt.Printf("Checking notification preference '%s' for instance: %s\n", preferenceID, resolvedName)
	resp, err := api.GetNotificationPreferences(ctx)
	if err != nil {
		return fmt.Errorf("failed to get notification preferences: %w", err)
	}

	category, pref, ok := findNotificationPreference(resp.EmailPreferences, preferenceID)
	if !ok {
		return fmt.Errorf("notification preference '%s' not found", preferenceID)
	}

	currentEnabled := pref.IsEnabled == 1
	fmt.Printf("\nTarget notification preference:\n")
	fmt.Printf("   ID         : %s\n", preferenceID)
	fmt.Printf("   Category   : %s\n", category)
	fmt.Printf("   Current    : %s\n", enabledStatus(pref.IsEnabled))
	fmt.Printf("   Target     : %s\n", enabledStatus(boolToInt(enabled)))
	if pref.FriendlyDescription != "" {
		fmt.Printf("   Description: %s\n", pref.FriendlyDescription)
	}

	if currentEnabled == enabled {
		fmt.Printf("\nNo change needed; preference already matches target state.\n")
		return nil
	}
	if dryRun {
		fmt.Printf("\nDRY RUN: would update notification preference '%s' on instance %s\n", preferenceID, resolvedName)
		return nil
	}
	if !skipConfirm {
		confirmed, err := confirm(fmt.Sprintf("Update notification preference '%s'?", preferenceID))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Printf("Operation cancelled\n")
			return nil
		}
	}

	updateResp, err := api.SetNotificationPreferences(ctx, map[string]bool{preferenceID: enabled})
	if err != nil {
		return fmt.Errorf("failed to update notification preference: %w", err)
	}
	fmt.Printf("✅ Notification preference '%s' updated\n", preferenceID)
	if len(updateResp.UpdatedEmailPreferences) > 0 {
		fmt.Printf("Updated preferences: %d\n", len(updateResp.UpdatedEmailPreferences))
	}
	return nil
}

func findNotificationPreference(
	preferences map[string]map[string]client.NotificationPreference,
	preferenceID string,
) (string, client.NotificationPreference, bool) {
	for category, prefs := range preferences {
		if pref, ok := prefs[preferenceID]; ok {
			return category, pref, true
		}
	}
	return "", client.NotificationPreference{}, false
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
