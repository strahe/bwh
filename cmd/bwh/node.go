package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/strahe/bwh/internal/config"
	"github.com/urfave/cli/v3"
)

var nodeCmd = &cli.Command{
	Name:  "node",
	Usage: "manage BWH VPS nodes",
	Commands: []*cli.Command{
		nodeAddCmd,
		nodeRemoveCmd,
		nodeListCmd,
		nodeSetDefaultCmd,
		nodeShowCmd,
		nodeValidateCmd,
	},
}

var nodeAddCmd = &cli.Command{
	Name:      "add",
	Usage:     "add a new BWH VPS node",
	ArgsUsage: "<name>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "api-key",
			Usage:    "BWH API key",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "veid",
			Usage:    "BWH VeID",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "description",
			Usage: "node description",
		},
		&cli.StringFlag{
			Name:  "endpoint",
			Usage: "custom API endpoint URL",
		},
		&cli.StringSliceFlag{
			Name:  "tags",
			Usage: "node tags (can be specified multiple times)",
		},
		&cli.BoolFlag{
			Name:  "default",
			Usage: "set this node as default",
		},
		&cli.BoolFlag{
			Name:  "validate",
			Usage: "validate the node connection after adding",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		manager, err := createConfigManager(cmd)
		if err != nil {
			return err
		}

		args := cmd.Args().Slice()
		if len(args) == 0 {
			return fmt.Errorf("node name is required")
		}
		name := args[0]
		instance := &config.Instance{
			APIKey:      cmd.String("api-key"),
			VeID:        cmd.String("veid"),
			Description: cmd.String("description"),
			Endpoint:    cmd.String("endpoint"),
			Tags:        cmd.StringSlice("tags"),
		}

		if err := manager.AddInstance(name, instance, cmd.Bool("default")); err != nil {
			return fmt.Errorf("failed to add node: %w", err)
		}

		fmt.Printf("Node '%s' added successfully\n", name)

		if cmd.Bool("validate") {
			fmt.Printf("Validating node '%s'...\n", name)
			if err := manager.ValidateInstance(name); err != nil {
				fmt.Printf("Warning: node validation failed: %v\n", err)
				return nil
			}
			fmt.Printf("Node '%s' validated successfully\n", name)
		}

		return nil
	},
}

var nodeRemoveCmd = &cli.Command{
	Name:      "remove",
	Usage:     "remove a BWH VPS node",
	ArgsUsage: "<name>",
	Aliases:   []string{"rm"},
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "skip confirmation prompt",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		manager, err := createConfigManager(cmd)
		if err != nil {
			return err
		}

		name := cmd.Args().First()

		// Check if node exists
		if _, err := manager.GetInstance(name); err != nil {
			return err
		}

		// Confirmation prompt unless skipped
		if !cmd.Bool("yes") {
			confirmed, err := promptConfirmation(fmt.Sprintf("Are you sure you want to remove node '%s'?", name))
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Println("Operation cancelled")
				return nil
			}
		}

		if err := manager.RemoveInstance(name); err != nil {
			return fmt.Errorf("failed to remove node: %w", err)
		}

		fmt.Printf("Node '%s' removed successfully\n", name)
		return nil
	},
}

var nodeListCmd = &cli.Command{
	Name:    "list",
	Usage:   "list all configured BWH VPS nodes",
	Aliases: []string{"ls"},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "format",
			Usage: "output format (table, json, yaml)",
			Value: "table",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		manager, err := createConfigManager(cmd)
		if err != nil {
			return err
		}

		instances := manager.ListInstances()
		if len(instances) == 0 {
			fmt.Println("No nodes configured. Use 'bwh node add' to add one.")
			return nil
		}

		defaultInstance := manager.GetDefaultInstance()

		switch cmd.String("format") {
		case "table":
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tDESCRIPTION\tVEID\tTAGS\tDEFAULT") //nolint:errcheck
			for name, instance := range instances {
				isDefault := ""
				if name == defaultInstance {
					isDefault = "*"
				}
				tags := strings.Join(instance.Tags, ",")
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", //nolint:errcheck
					name, instance.Description, instance.VeID, tags, isDefault)
			}
			w.Flush() //nolint:errcheck
		case "json":
			// For JSON output, we need to mask sensitive data
			maskedInstances := make(map[string]interface{})
			for name, instance := range instances {
				maskedInstances[name] = map[string]interface{}{
					"description": instance.Description,
					"veid":        instance.VeID,
					"endpoint":    instance.Endpoint,
					"tags":        instance.Tags,
					"api_key":     maskAPIKey(instance.APIKey),
				}
			}
			output := map[string]interface{}{
				"default_node": defaultInstance,
				"nodes":        maskedInstances,
			}
			if err := printJSON(output); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported format: %s", cmd.String("format"))
		}

		return nil
	},
}

var nodeSetDefaultCmd = &cli.Command{
	Name:      "set-default",
	Usage:     "set the default BWH VPS node",
	ArgsUsage: "<name>",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		manager, err := createConfigManager(cmd)
		if err != nil {
			return err
		}

		name := cmd.Args().First()
		if err := manager.SetDefault(name); err != nil {
			return fmt.Errorf("failed to set default node: %w", err)
		}

		fmt.Printf("Default node set to '%s'\n", name)
		return nil
	},
}

var nodeShowCmd = &cli.Command{
	Name:      "show",
	Usage:     "show configuration for a specific VPS node",
	ArgsUsage: "[name]",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		manager, err := createConfigManager(cmd)
		if err != nil {
			return err
		}

		var instanceName string
		args := cmd.Args().Slice()
		if len(args) > 0 {
			instanceName = args[0]
		} else {
			// Show default node if no name specified
			instanceName = manager.GetDefaultInstance()
			if instanceName == "" {
				return fmt.Errorf("no default node set and no node name provided")
			}
		}

		instance, err := manager.GetInstance(instanceName)
		if err != nil {
			return err
		}

		fmt.Printf("Node: %s\n", instanceName)
		fmt.Printf("Description: %s\n", instance.Description)
		fmt.Printf("VeID: %s\n", instance.VeID)
		fmt.Printf("API Key: %s\n", maskAPIKey(instance.APIKey))
		if instance.Endpoint != "" {
			fmt.Printf("Endpoint: %s\n", instance.Endpoint)
		}
		if len(instance.Tags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(instance.Tags, ", "))
		}
		if instanceName == manager.GetDefaultInstance() {
			fmt.Printf("Default: Yes\n")
		}

		return nil
	},
}

var nodeValidateCmd = &cli.Command{
	Name:      "validate",
	Usage:     "validate BWH VPS node connection",
	ArgsUsage: "[name]",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		manager, err := createConfigManager(cmd)
		if err != nil {
			return err
		}

		var instanceName string
		args := cmd.Args().Slice()
		if len(args) > 0 {
			instanceName = args[0]
		} else {
			// Validate default node if no name specified
			_, instanceName, err = manager.ResolveInstance("")
			if err != nil {
				return err
			}
		}

		fmt.Printf("Validating node '%s'...\n", instanceName)
		if err := manager.ValidateInstance(instanceName); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		fmt.Printf("Node '%s' validated successfully\n", instanceName)
		return nil
	},
}

// maskAPIKey masks the API key for display purposes
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return strings.Repeat("*", len(apiKey))
	}
	return apiKey[:4] + strings.Repeat("*", len(apiKey)-8) + apiKey[len(apiKey)-4:]
}
