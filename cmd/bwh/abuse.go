package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/strahe/bwh/pkg/client"
	"github.com/urfave/cli/v3"
)

var abuseCmd = &cli.Command{
	Name:  "abuse",
	Usage: "inspect abuse suspensions and policy violations",
	Commands: []*cli.Command{
		abuseSuspensionsCmd,
		abusePolicyCmd,
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
