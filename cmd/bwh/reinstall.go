package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/strahe/bwh/pkg/client"
	"github.com/urfave/cli/v3"
)

var reinstallCmd = &cli.Command{
	Name:  "reinstall",
	Usage: "reinstall the VPS operating system (WARNING: destroys all data)",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "os",
			Usage: "operating system template to install",
		},
		&cli.BoolFlag{
			Name:  "list",
			Usage: "list available operating system templates",
		},
		forceFlag(),
		yesFlag(),
		dryRunFlag(),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		osTemplate := cmd.String("os")
		listOnly := cmd.Bool("list")

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		return runReinstall(ctx, bwhClient, resolvedName, osTemplate, listOnly, cmd.Bool("dry-run"), skipConfirmOrForce(cmd), confirmReinstall)
	},
}

type reinstallAPI interface {
	GetAvailableOS(context.Context) (*client.AvailableOSResponse, error)
	ReinstallOS(context.Context, string) error
}

type reinstallConfirmationFunc func(instanceName, currentOS, targetOS string) bool

func runReinstall(ctx context.Context, api reinstallAPI, resolvedName, osTemplate string, listOnly, dryRun, skipConfirm bool, confirm reinstallConfirmationFunc) error {
	osInfo, err := api.GetAvailableOS(ctx)
	if err != nil {
		return fmt.Errorf("failed to get available OS templates: %w", err)
	}

	if listOnly {
		displayAvailableOS(osInfo, resolvedName)
		return nil
	}

	if osTemplate == "" {
		fmt.Printf("No OS template specified. Use --os flag with one of the following templates:\n\n")
		displayAvailableOS(osInfo, resolvedName)
		fmt.Printf("\nExample: bwh reinstall --os ubuntu-24.04-x86_64\n")
		return nil
	}

	if !isValidOSTemplate(osTemplate, osInfo.Templates) {
		fmt.Printf("❌ Invalid OS template: %s\n\n", osTemplate)
		fmt.Printf("Available templates:\n")
		for _, template := range osInfo.Templates {
			fmt.Printf("  %s\n", template)
		}
		return fmt.Errorf("invalid OS template")
	}

	fmt.Printf("Instance: %s\n", resolvedName)
	fmt.Printf("Current OS: %s\n", osInfo.Installed)
	fmt.Printf("Target OS:  %s\n", osTemplate)
	fmt.Printf("\n")

	if dryRun {
		printDryRun("reinstallOS", resolvedName, fmt.Sprintf("os: %s", osTemplate))
		return nil
	}

	if !skipConfirm {
		if !confirm(resolvedName, osInfo.Installed, osTemplate) {
			printOperationCancelled()
			return nil
		}
	}

	fmt.Printf("🔄 Starting OS reinstall for instance: %s\n", resolvedName)
	fmt.Printf("⏳ This may take several minutes...\n")

	if err := api.ReinstallOS(ctx, osTemplate); err != nil {
		return fmt.Errorf("failed to reinstall OS: %w", err)
	}

	fmt.Printf("✅ OS reinstall initiated successfully\n")
	fmt.Printf("📋 Your VPS is being reinstalled with %s\n", osTemplate)
	fmt.Printf("⚠️  Note: The process may take 5-15 minutes to complete\n")

	return nil
}

func displayAvailableOS(osInfo *client.AvailableOSResponse, instanceName string) {
	fmt.Printf("Instance: %s\n", instanceName)
	fmt.Printf("Current OS: %s\n", osInfo.Installed)
	fmt.Printf("\nAvailable OS Templates:\n")

	// Group templates by OS family for better display
	grouped := groupOSTemplates(osInfo.Templates)

	for family, templates := range grouped {
		fmt.Printf("\n%s:\n", strings.ToUpper(family[:1])+family[1:])
		for _, template := range templates {
			if template == osInfo.Installed {
				fmt.Printf("  %s (currently installed)\n", template)
			} else {
				fmt.Printf("  %s\n", template)
			}
		}
	}

	fmt.Printf("\nTotal: %d templates available\n", len(osInfo.Templates))
}

func groupOSTemplates(templates []string) map[string][]string {
	grouped := make(map[string][]string)

	for _, template := range templates {
		parts := strings.Split(template, "-")
		if len(parts) > 0 {
			family := parts[0]
			if grouped[family] == nil {
				grouped[family] = []string{}
			}
			grouped[family] = append(grouped[family], template)
		}
	}

	// Sort templates within each family
	for family := range grouped {
		sort.Strings(grouped[family])
	}

	return grouped
}

func isValidOSTemplate(template string, availableTemplates []string) bool {
	for _, available := range availableTemplates {
		if template == available {
			return true
		}
	}
	return false
}

func confirmReinstall(instanceName, currentOS, targetOS string) bool {
	fmt.Printf("🚨 DANGER: OS REINSTALL WILL DESTROY ALL DATA!\n")
	fmt.Printf("🚨 This action is IRREVERSIBLE!\n")
	fmt.Printf("\n")
	fmt.Printf("VPS Instance: %s\n", instanceName)
	fmt.Printf("Current OS: %s → Target OS: %s\n", currentOS, targetOS)
	fmt.Printf("\n")
	fmt.Printf("⚠️  ALL FILES, DATABASES, CONFIGURATIONS WILL BE LOST!\n")
	fmt.Printf("⚠️  MAKE SURE YOU HAVE BACKUPS!\n")
	fmt.Printf("\n")
	fmt.Printf("To confirm this dangerous operation, type the target OS exactly: %s\n", targetOS)
	fmt.Printf("Type here: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(response)
	return response == targetOS
}
