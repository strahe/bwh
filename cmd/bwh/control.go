package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"
)

var startCmd = &cli.Command{
	Name:  "start",
	Usage: "start the VPS",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return executeVPSAction(ctx, cmd, "start", false)
	},
}

var stopCmd = &cli.Command{
	Name:  "stop",
	Usage: "stop the VPS",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "skip confirmation prompt",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return executeVPSAction(ctx, cmd, "stop", !cmd.Bool("yes"))
	},
}

var restartCmd = &cli.Command{
	Name:  "restart",
	Usage: "restart the VPS",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "skip confirmation prompt",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return executeVPSAction(ctx, cmd, "restart", !cmd.Bool("yes"))
	},
}

var killCmd = &cli.Command{
	Name:  "kill",
	Usage: "forcefully stop a stuck VPS (WARNING: data loss)",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "force",
			Usage: "force kill without confirmation (dangerous)",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		return executeVPSAction(ctx, cmd, "kill", !cmd.Bool("force"))
	},
}

var hostnameCmd = &cli.Command{
	Name:      "hostname",
	Usage:     "set hostname for the VPS",
	ArgsUsage: "<new_hostname>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "skip confirmation prompt",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() != 1 {
			return fmt.Errorf("hostname command requires exactly one argument: <new_hostname>")
		}

		newHostname := cmd.Args().Get(0)
		if newHostname == "" {
			return fmt.Errorf("hostname cannot be empty")
		}

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		// Confirmation prompt
		if !cmd.Bool("yes") {
			if !confirmAction("set hostname", resolvedName, newHostname) {
				fmt.Println("Operation cancelled.")
				return nil
			}
		}

		fmt.Printf("Setting hostname to '%s' for instance: %s\n", newHostname, resolvedName)

		if err := bwhClient.SetHostname(ctx, newHostname); err != nil {
			return fmt.Errorf("failed to set hostname: %w", err)
		}

		fmt.Printf("✅ Hostname set to '%s' successfully\n", newHostname)
		return nil
	},
}

var setPTRCmd = &cli.Command{
	Name:      "set-ptr",
	Aliases:   []string{"setPTR"},
	Usage:     "set new PTR (rDNS) record for IP address",
	ArgsUsage: "<ip> <ptr>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "skip confirmation prompt",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() != 2 {
			return fmt.Errorf("setPTR command requires exactly two arguments: <ip> <ptr>")
		}

		ip := cmd.Args().Get(0)
		ptr := cmd.Args().Get(1)

		if ip == "" {
			return fmt.Errorf("IP address cannot be empty")
		}
		if ptr == "" {
			return fmt.Errorf("PTR record cannot be empty")
		}

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		// Confirmation prompt
		if !cmd.Bool("yes") {
			if !confirmAction("set PTR", resolvedName, ip, ptr) {
				fmt.Println("Operation cancelled.")
				return nil
			}
		}

		fmt.Printf("Setting PTR record for IP '%s' to '%s' for instance: %s\n", ip, ptr, resolvedName)

		if err := bwhClient.SetPTR(ctx, ip, ptr); err != nil {
			return fmt.Errorf("failed to set PTR record: %w", err)
		}

		fmt.Printf("✅ PTR record set for IP '%s' to '%s' successfully\n", ip, ptr)
		return nil
	},
}

func executeVPSAction(ctx context.Context, cmd *cli.Command, action string, needsConfirm bool) error {
	bwhClient, resolvedName, err := createBWHClient(cmd)
	if err != nil {
		return err
	}

	// Confirmation prompt
	if needsConfirm {
		if !confirmAction(action, resolvedName) {
			fmt.Println("Operation cancelled.")
			return nil
		}
	}

	fmt.Printf("Executing %s for instance: %s\n", action, resolvedName)

	// Execute action
	switch action {
	case "start":
		err = bwhClient.Start(ctx)
	case "stop":
		err = bwhClient.Stop(ctx)
	case "restart":
		err = bwhClient.Restart(ctx)
	case "kill":
		err = bwhClient.Kill(ctx)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	if err != nil {
		return fmt.Errorf("failed to %s VPS: %w", action, err)
	}

	fmt.Printf("✅ VPS %s completed successfully\n", action)
	return nil
}

func confirmAction(action, instanceName string, args ...string) bool {
	var prompt string
	switch action {
	case "stop":
		prompt = fmt.Sprintf("Stop VPS '%s'? [y/N]: ", instanceName)
	case "restart":
		prompt = fmt.Sprintf("Restart VPS '%s'? [y/N]: ", instanceName)
	case "kill":
		fmt.Printf("⚠️  WARNING: KILL will forcefully terminate VPS '%s'\n", instanceName)
		fmt.Printf("⚠️  ANY UNSAVED DATA WILL BE LOST!\n")
		prompt = "Type 'kill' to confirm: "
	case "reset root password":
		fmt.Printf("Reset root password for VPS '%s'?\n", instanceName)
		fmt.Printf("This will generate a new random root password.\n")
		prompt = "Continue? [y/N]: "
	case "set hostname":
		if len(args) > 0 {
			fmt.Printf("Set hostname to '%s' for VPS '%s'? [y/N]: ", args[0], instanceName)
		} else {
			fmt.Printf("Set hostname for VPS '%s'? [y/N]: ", instanceName)
		}
		prompt = ""
	case "set PTR":
		if len(args) >= 2 {
			fmt.Printf("Set PTR record for IP '%s' to '%s' for VPS '%s'? [y/N]: ", args[0], args[1], instanceName)
		} else {
			fmt.Printf("Set PTR record for VPS '%s'? [y/N]: ", instanceName)
		}
		prompt = ""
	case "mount ISO":
		if len(args) > 0 {
			fmt.Printf("Mount ISO '%s' for VPS '%s'?\n", args[0], instanceName)
		} else {
			fmt.Printf("Mount ISO for VPS '%s'?\n", instanceName)
		}
		fmt.Printf("⚠️  VPS must be completely shut down and restarted after this operation.\n")
		prompt = "Continue? [y/N]: "
	case "unmount ISO":
		fmt.Printf("Unmount ISO for VPS '%s'?\n", instanceName)
		fmt.Printf("⚠️  VPS must be completely shut down and restarted after this operation.\n")
		prompt = "Continue? [y/N]: "
	}

	if prompt != "" {
		fmt.Print(prompt)
	}
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))

	if action == "kill" {
		return response == "kill"
	}

	return response == "y" || response == "yes"
}
