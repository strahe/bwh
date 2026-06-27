package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/urfave/cli/v3"
)

type confirmationFunc func(string) (bool, error)

func yesFlag() cli.Flag {
	return &cli.BoolFlag{
		Name:    "yes",
		Aliases: []string{"y"},
		Usage:   "skip confirmation prompt",
	}
}

func dryRunFlag() cli.Flag {
	return &cli.BoolFlag{
		Name:  "dry-run",
		Usage: "validate and show the write action without calling the write API",
	}
}

func writeFlags(extra ...cli.Flag) []cli.Flag {
	flags := make([]cli.Flag, 0, len(extra)+2)
	flags = append(flags, extra...)
	flags = append(flags, yesFlag(), dryRunFlag())
	return flags
}

func forceFlag() cli.Flag {
	return &cli.BoolFlag{
		Name:  "force",
		Usage: "skip confirmation prompt for this dangerous operation",
	}
}

func skipConfirm(cmd *cli.Command) bool {
	return cmd.Bool("yes")
}

func skipConfirmOrForce(cmd *cli.Command) bool {
	return cmd.Bool("yes") || cmd.Bool("force")
}

func confirmWrite(prompt string, skip bool, confirm confirmationFunc) (bool, error) {
	if skip {
		return true, nil
	}
	confirmed, err := confirm(prompt)
	if err != nil {
		return false, err
	}
	if !confirmed {
		printOperationCancelled()
		return false, nil
	}
	return true, nil
}

func promptExactConfirmation(prompt, expected string) (bool, error) {
	fmt.Print(prompt)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			fmt.Printf("\n")
			return false, fmt.Errorf("operation cancelled (EOF)")
		}
		return false, fmt.Errorf("failed to read user input: %w", err)
	}

	return strings.TrimSpace(response) == expected, nil
}

func printOperationCancelled() {
	fmt.Println("Operation cancelled")
}

func printDryRun(endpoint, instanceName string, details ...string) {
	fmt.Printf("DRY RUN: would call %s for instance %s\n", endpoint, instanceName)
	for _, detail := range details {
		if strings.TrimSpace(detail) != "" {
			fmt.Printf("   %s\n", detail)
		}
	}
}

func maskSensitive(value string) string {
	return maskSecret(value)
}

func maskSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func maskSSHKey(key string) string {
	fields := strings.Fields(key)
	if len(fields) == 0 {
		return ""
	}
	if len(fields) == 1 {
		return maskSensitive(fields[0])
	}
	masked := fields[0] + " " + maskSensitive(fields[1])
	if len(fields) > 2 {
		masked += " " + fields[len(fields)-1]
	}
	return masked
}

func sameStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	normalizedA := make([]string, len(a))
	normalizedB := make([]string, len(b))
	for i := range a {
		normalizedA[i] = strings.TrimSpace(a[i])
		normalizedB[i] = strings.TrimSpace(b[i])
	}
	slices.Sort(normalizedA)
	slices.Sort(normalizedB)
	return slices.Equal(normalizedA, normalizedB)
}

func trimIPv6Subnet(subnet string) string {
	return strings.TrimSuffix(strings.TrimSpace(subnet), "/64")
}

func isIPv6Address(value string) bool {
	return strings.Contains(value, ":")
}
