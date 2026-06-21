package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/strahe/bwh/pkg/client"
	"github.com/urfave/cli/v3"
)

var startCmd = &cli.Command{
	Name:  "start",
	Usage: "start the VPS",
	Flags: []cli.Flag{
		dryRunFlag(),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}
		return runVPSAction(ctx, bwhClient, resolvedName, "start", cmd.Bool("dry-run"), true, promptConfirmation)
	},
}

var stopCmd = &cli.Command{
	Name:  "stop",
	Usage: "stop the VPS",
	Flags: writeFlags(),
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}
		return runVPSAction(ctx, bwhClient, resolvedName, "stop", cmd.Bool("dry-run"), skipConfirm(cmd), promptConfirmation)
	},
}

var restartCmd = &cli.Command{
	Name:  "restart",
	Usage: "restart the VPS",
	Flags: writeFlags(),
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}
		return runVPSAction(ctx, bwhClient, resolvedName, "restart", cmd.Bool("dry-run"), skipConfirm(cmd), promptConfirmation)
	},
}

var killCmd = &cli.Command{
	Name:  "kill",
	Usage: "forcefully stop a stuck VPS (WARNING: data loss)",
	Flags: []cli.Flag{
		forceFlag(),
		yesFlag(),
		dryRunFlag(),
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}
		return runVPSAction(ctx, bwhClient, resolvedName, "kill", cmd.Bool("dry-run"), skipConfirmOrForce(cmd), confirmKill)
	},
}

var hostnameCmd = &cli.Command{
	Name:      "hostname",
	Usage:     "set hostname for the VPS",
	ArgsUsage: "<new_hostname>",
	Flags:     writeFlags(),
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

		return runSetHostname(ctx, bwhClient, resolvedName, newHostname, cmd.Bool("dry-run"), skipConfirm(cmd), promptConfirmation)
	},
}

var setPTRCmd = &cli.Command{
	Name:      "set-ptr",
	Aliases:   []string{"setPTR"},
	Usage:     "set new PTR (rDNS) record for IP address",
	ArgsUsage: "<ip> <ptr>",
	Flags:     writeFlags(),
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

		return runSetPTR(ctx, bwhClient, resolvedName, ip, ptr, cmd.Bool("dry-run"), skipConfirm(cmd), promptConfirmation)
	},
}

type powerAPI interface {
	GetLiveServiceInfo(context.Context) (*client.LiveServiceInfo, error)
	Start(context.Context) error
	Stop(context.Context) error
	Restart(context.Context) error
	Kill(context.Context) error
}

func runVPSAction(ctx context.Context, api powerAPI, resolvedName, action string, dryRun, skipConfirm bool, confirm confirmationFunc) error {
	status := ""
	if info, err := api.GetLiveServiceInfo(ctx); err == nil {
		status = strings.ToLower(strings.TrimSpace(info.VeStatus))
		if status != "" {
			fmt.Printf("Current VPS status for instance %s: %s\n", resolvedName, info.VeStatus)
		}
	} else {
		fmt.Printf("Warning: failed to read current VPS status: %v\n", err)
	}

	if action == "start" && status == "running" {
		fmt.Printf("✅ VPS is already running (no change needed)\n")
		return nil
	}
	if (action == "stop" || action == "kill") && status == "stopped" {
		fmt.Printf("✅ VPS is already stopped (no change needed)\n")
		return nil
	}

	if dryRun {
		printDryRun(action, resolvedName)
		return nil
	}

	prompt := fmt.Sprintf("%s VPS '%s'?", strings.ToUpper(action[:1])+action[1:], resolvedName)
	if action == "kill" {
		prompt = fmt.Sprintf("Forcefully kill VPS '%s'?", resolvedName)
	}
	confirmed, err := confirmWrite(prompt, skipConfirm, confirm)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	fmt.Printf("Executing %s for instance: %s\n", action, resolvedName)

	switch action {
	case "start":
		err = api.Start(ctx)
	case "stop":
		err = api.Stop(ctx)
	case "restart":
		err = api.Restart(ctx)
	case "kill":
		err = api.Kill(ctx)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	if err != nil {
		return fmt.Errorf("failed to %s VPS: %w", action, err)
	}

	fmt.Printf("✅ VPS %s completed successfully\n", action)
	return nil
}

type hostnameAPI interface {
	GetServiceInfo(context.Context) (*client.ServiceInfo, error)
	SetHostname(context.Context, string) error
}

func runSetHostname(ctx context.Context, api hostnameAPI, resolvedName, newHostname string, dryRun, skipConfirm bool, confirm confirmationFunc) error {
	info, err := api.GetServiceInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get service info: %w", err)
	}
	if info.Hostname == newHostname {
		fmt.Printf("✅ Hostname is already '%s' (no change needed)\n", newHostname)
		return nil
	}
	if dryRun {
		printDryRun("setHostname", resolvedName, fmt.Sprintf("hostname: %s -> %s", info.Hostname, newHostname))
		return nil
	}
	confirmed, err := confirmWrite(fmt.Sprintf("Set hostname to '%s' for VPS '%s'?", newHostname, resolvedName), skipConfirm, confirm)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	fmt.Printf("Setting hostname to '%s' for instance: %s\n", newHostname, resolvedName)
	if err := api.SetHostname(ctx, newHostname); err != nil {
		return fmt.Errorf("failed to set hostname: %w", err)
	}
	fmt.Printf("✅ Hostname set to '%s' successfully\n", newHostname)
	return nil
}

type ptrAPI interface {
	GetServiceInfo(context.Context) (*client.ServiceInfo, error)
	SetPTR(context.Context, string, string) error
}

func runSetPTR(ctx context.Context, api ptrAPI, resolvedName, ip, ptr string, dryRun, skipConfirm bool, confirm confirmationFunc) error {
	info, err := api.GetServiceInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get service info: %w", err)
	}
	if !info.RDNSAPIAvailable {
		return fmt.Errorf("rDNS API is not available for instance %s", resolvedName)
	}
	if !containsString(info.IPAddresses, ip) {
		return fmt.Errorf("IP address %s is not assigned to instance %s", ip, resolvedName)
	}
	currentPTR := ""
	if info.PTR != nil {
		currentPTR = info.PTR[ip]
	}
	if currentPTR == ptr {
		fmt.Printf("✅ PTR record for IP '%s' is already '%s' (no change needed)\n", ip, ptr)
		return nil
	}
	if dryRun {
		printDryRun("setPTR", resolvedName, fmt.Sprintf("ip: %s", ip), fmt.Sprintf("ptr: %s -> %s", currentPTR, ptr))
		return nil
	}
	confirmed, err := confirmWrite(fmt.Sprintf("Set PTR record for IP '%s' to '%s' on VPS '%s'?", ip, ptr, resolvedName), skipConfirm, confirm)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	fmt.Printf("Setting PTR record for IP '%s' to '%s' for instance: %s\n", ip, ptr, resolvedName)
	if err := api.SetPTR(ctx, ip, ptr); err != nil {
		return fmt.Errorf("failed to set PTR record: %w", err)
	}
	fmt.Printf("✅ PTR record set for IP '%s' to '%s' successfully\n", ip, ptr)
	return nil
}

func confirmKill(prompt string) (bool, error) {
	fmt.Printf("⚠️  WARNING: %s\n", prompt)
	fmt.Printf("⚠️  ANY UNSAVED DATA WILL BE LOST!\n")
	return promptExactConfirmation("Type 'kill' to confirm: ", "kill")
}
