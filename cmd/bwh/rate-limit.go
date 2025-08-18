package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

var rateLimitCmd = &cli.Command{
	Name:    "rate-limit",
	Usage:   "check API rate limit status",
	Aliases: []string{"rl"},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		status, err := bwhClient.GetRateLimitStatus(ctx)
		if err != nil {
			return fmt.Errorf("failed to get rate limit status: %w", err)
		}

		fmt.Printf("API Quota Status for %s:\n", resolvedName)
		fmt.Printf("  15-minute window: %d calls remaining\n", status.RemainingPoints15Min)
		fmt.Printf("  24-hour window:   %d calls remaining\n", status.RemainingPoints24H)

		if status.RemainingPoints15Min < 10 {
			fmt.Printf("\n⚠️  Warning: Low quota in 15-minute window\n")
		}
		if status.RemainingPoints24H < 100 {
			fmt.Printf("\n⚠️  Warning: Low quota in 24-hour window\n")
		}

		return nil
	},
}
