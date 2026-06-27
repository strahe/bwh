package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/strahe/bwh/pkg/client"
	"github.com/urfave/cli/v3"
)

var abuseCmd = &cli.Command{
	Name:  "abuse",
	Usage: "display and resolve abuse suspensions and policy violations",
	Commands: []*cli.Command{
		abuseSuspensionsCmd,
		abusePolicyCmd,
		abuseUnsuspendCmd,
		abuseResolvePolicyCmd,
	},
}

var abuseSuspensionsCmd = &cli.Command{
	Name:  "suspensions",
	Usage: "show service suspension details",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Getting suspension details for instance: %s\n", resolvedName)
		resp, err := bwhClient.GetSuspensionDetails(ctx)
		if err != nil {
			return fmt.Errorf("failed to get suspension details: %w", err)
		}

		displaySuspensionDetails(resp)
		return nil
	},
}

var abusePolicyCmd = &cli.Command{
	Name:  "policy",
	Usage: "show active policy violations",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Getting policy violations for instance: %s\n", resolvedName)
		resp, err := bwhClient.GetPolicyViolations(ctx)
		if err != nil {
			return fmt.Errorf("failed to get policy violations: %w", err)
		}

		displayPolicyViolations(resp)
		return nil
	},
}

var abuseUnsuspendCmd = &cli.Command{
	Name:      "unsuspend",
	Usage:     "clear a soft abuse issue and unsuspend the VPS",
	ArgsUsage: "<record_id>",
	Flags:     writeFlags(),
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() != 1 {
			return fmt.Errorf("record_id is required")
		}
		recordID, err := parseRecordID(cmd.Args().First())
		if err != nil {
			return err
		}

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		return runAbuseUnsuspend(ctx, bwhClient, resolvedName, recordID, cmd.Bool("dry-run"), skipConfirm(cmd), promptConfirmation)
	},
}

var abuseResolvePolicyCmd = &cli.Command{
	Name:      "resolve-policy",
	Usage:     "mark a soft policy violation as resolved",
	ArgsUsage: "<record_id>",
	Flags:     writeFlags(),
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() != 1 {
			return fmt.Errorf("record_id is required")
		}
		recordID, err := parseRecordID(cmd.Args().First())
		if err != nil {
			return err
		}

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		return runAbuseResolvePolicy(ctx, bwhClient, resolvedName, recordID, cmd.Bool("dry-run"), skipConfirm(cmd), promptConfirmation)
	},
}

type abuseAPI interface {
	GetSuspensionDetails(context.Context) (*client.SuspensionDetailsResponse, error)
	GetPolicyViolations(context.Context) (*client.PolicyViolationsResponse, error)
	Unsuspend(context.Context, int) error
	ResolvePolicyViolation(context.Context, int) error
}

func displaySuspensionDetails(resp *client.SuspensionDetailsResponse) {
	fmt.Printf("\n🚫 SUSPENSION DETAILS\n")
	fmt.Printf("   Suspensions (YTD): %d\n", resp.SuspensionCount)
	fmt.Printf("   Abuse Points     : %d / %d\n", resp.TotalAbusePoints, resp.MaxAbusePoints)

	if len(resp.Suspensions) == 0 {
		fmt.Printf("\nNo active suspension issues found.\n")
		return
	}

	records := append([]client.SuspensionRecord{}, resp.Suspensions...)
	sort.Slice(records, func(i, j int) bool {
		return records[i].RecordID < records[j].RecordID
	})

	fmt.Printf("\nOutstanding Issues (%d):\n", len(records))
	for i, record := range records {
		fmt.Printf("\n[%d] Case #%d\n", i+1, record.RecordID)
		fmt.Printf("    Flag        : %s\n", record.Flag)
		fmt.Printf("    Soft Resolve: %s\n", yesNo(record.IsSoft == 1))
		fmt.Printf("    Abuse Points: %d\n", record.AbusePoints)
		if record.EvidenceRecordID != 0 {
			fmt.Printf("    Evidence ID : %d\n", record.EvidenceRecordID)
			if text := resp.Evidence[fmt.Sprintf("%d", record.EvidenceRecordID)]; text != "" {
				fmt.Printf("    Evidence    : %s\n", summarizeText(text, 120))
			}
		}
	}
}

func displayPolicyViolations(resp *client.PolicyViolationsResponse) {
	fmt.Printf("\n⚠️  POLICY VIOLATIONS\n")
	fmt.Printf("   Abuse Points: %d / %d\n", resp.TotalAbusePoints, resp.MaxAbusePoints)

	if len(resp.PolicyViolations) == 0 {
		fmt.Printf("\nNo active policy violations found.\n")
		return
	}

	records := append([]client.PolicyViolationRecord{}, resp.PolicyViolations...)
	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp > records[j].Timestamp
	})

	fmt.Printf("\nActive Violations (%d):\n", len(records))
	for i, record := range records {
		fmt.Printf("\n[%d] Case #%d\n", i+1, record.RecordID)
		fmt.Printf("    Flag        : %s\n", record.Flag)
		fmt.Printf("    Soft Resolve: %s\n", yesNo(record.IsSoft == 1))
		fmt.Printf("    Abuse Points: %d\n", record.AbusePoints)
		if record.Timestamp > 0 {
			fmt.Printf("    Created     : %s\n", time.Unix(record.Timestamp, 0).Format("2006-01-02 15:04:05"))
		}
		if record.SuspendAt > 0 {
			fmt.Printf("    Suspend At  : %s\n", time.Unix(record.SuspendAt, 0).Format("2006-01-02 15:04:05"))
		}
		if record.EvidenceData != "" {
			fmt.Printf("    Evidence    : %s\n", summarizeText(record.EvidenceData, 120))
		}
	}
}

func summarizeText(s string, maxLen int) string {
	if maxLen <= 0 {
		return "..."
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func parseRecordID(raw string) (int, error) {
	recordID, err := strconv.Atoi(raw)
	if err != nil || recordID <= 0 {
		return 0, fmt.Errorf("record_id must be a positive integer")
	}
	return recordID, nil
}

func runAbuseUnsuspend(ctx context.Context, api abuseAPI, resolvedName string, recordID int, dryRun, skipConfirm bool, confirm confirmationFunc) error {
	fmt.Printf("Checking suspension case #%d for instance: %s\n", recordID, resolvedName)
	resp, err := api.GetSuspensionDetails(ctx)
	if err != nil {
		return fmt.Errorf("failed to get suspension details: %w", err)
	}

	record, ok := findSuspensionRecord(resp.Suspensions, recordID)
	if !ok {
		return fmt.Errorf("suspension case #%d not found", recordID)
	}
	printSuspensionActionSummary(record)
	if record.IsSoft != 1 {
		return fmt.Errorf("suspension case #%d cannot be resolved through API; contact support", recordID)
	}
	if dryRun {
		printDryRun("unsuspend", resolvedName, fmt.Sprintf("case: #%d", recordID))
		return nil
	}
	confirmed, err := confirmWrite(fmt.Sprintf("Unsuspend VPS by clearing case #%d?", recordID), skipConfirm, confirm)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	if err := api.Unsuspend(ctx, recordID); err != nil {
		return fmt.Errorf("failed to unsuspend case #%d: %w", recordID, err)
	}
	fmt.Printf("✅ Suspension case #%d cleared\n", recordID)
	return nil
}

func runAbuseResolvePolicy(ctx context.Context, api abuseAPI, resolvedName string, recordID int, dryRun, skipConfirm bool, confirm confirmationFunc) error {
	fmt.Printf("Checking policy violation case #%d for instance: %s\n", recordID, resolvedName)
	resp, err := api.GetPolicyViolations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get policy violations: %w", err)
	}

	record, ok := findPolicyViolationRecord(resp.PolicyViolations, recordID)
	if !ok {
		return fmt.Errorf("policy violation case #%d not found", recordID)
	}
	printPolicyActionSummary(record)
	if record.IsSoft != 1 {
		return fmt.Errorf("policy violation case #%d cannot be resolved through API; contact support", recordID)
	}
	if dryRun {
		printDryRun("resolvePolicyViolation", resolvedName, fmt.Sprintf("case: #%d", recordID))
		return nil
	}
	confirmed, err := confirmWrite(fmt.Sprintf("Mark policy violation case #%d as resolved?", recordID), skipConfirm, confirm)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	if err := api.ResolvePolicyViolation(ctx, recordID); err != nil {
		return fmt.Errorf("failed to resolve policy violation case #%d: %w", recordID, err)
	}
	fmt.Printf("✅ Policy violation case #%d resolved\n", recordID)
	return nil
}

func findSuspensionRecord(records []client.SuspensionRecord, recordID int) (client.SuspensionRecord, bool) {
	for _, record := range records {
		if record.RecordID == recordID {
			return record, true
		}
	}
	return client.SuspensionRecord{}, false
}

func findPolicyViolationRecord(records []client.PolicyViolationRecord, recordID int) (client.PolicyViolationRecord, bool) {
	for _, record := range records {
		if record.RecordID == recordID {
			return record, true
		}
	}
	return client.PolicyViolationRecord{}, false
}

func printSuspensionActionSummary(record client.SuspensionRecord) {
	fmt.Printf("\nTarget suspension case:\n")
	fmt.Printf("   Case ID     : %d\n", record.RecordID)
	fmt.Printf("   Flag        : %s\n", record.Flag)
	fmt.Printf("   Soft Resolve: %s\n", yesNo(record.IsSoft == 1))
	fmt.Printf("   Abuse Points: %d\n", record.AbusePoints)
}

func printPolicyActionSummary(record client.PolicyViolationRecord) {
	fmt.Printf("\nTarget policy violation:\n")
	fmt.Printf("   Case ID     : %d\n", record.RecordID)
	fmt.Printf("   Flag        : %s\n", record.Flag)
	fmt.Printf("   Soft Resolve: %s\n", yesNo(record.IsSoft == 1))
	fmt.Printf("   Abuse Points: %d\n", record.AbusePoints)
	if record.SuspendAt > 0 {
		fmt.Printf("   Suspend At  : %s\n", time.Unix(record.SuspendAt, 0).Format("2006-01-02 15:04:05"))
	}
}
