package main

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/strahe/bwh/pkg/client"
	"github.com/urfave/cli/v3"
)

var ipv6Cmd = &cli.Command{
	Name:  "ipv6",
	Usage: "manage IPv6 subnets",
	Commands: []*cli.Command{
		ipv6AddCmd,
		ipv6DeleteCmd,
		ipv6ListCmd,
	},
}

var ipv6AddCmd = &cli.Command{
	Name:  "add",
	Usage: "assign a new IPv6 /64 subnet",
	Flags: writeFlags(),
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		serviceInfo, err := bwhClient.GetServiceInfo(ctx)
		if err != nil {
			return fmt.Errorf("failed to get service info: %w", err)
		}
		if !serviceInfo.LocationIPv6Ready {
			return fmt.Errorf("IPv6 is not available at this location (%s)", serviceInfo.NodeLocation)
		}
		currentIPv6 := countIPv6Subnets(serviceInfo.IPAddresses)
		if serviceInfo.PlanMaxIPv6s > 0 && currentIPv6 >= serviceInfo.PlanMaxIPv6s {
			return fmt.Errorf("IPv6 subnet limit reached: %d/%d", currentIPv6, serviceInfo.PlanMaxIPv6s)
		}
		if cmd.Bool("dry-run") {
			printDryRun("ipv6/add", resolvedName, fmt.Sprintf("assigned IPv6 subnets: %d/%d", currentIPv6, serviceInfo.PlanMaxIPv6s))
			return nil
		}

		if !cmd.Bool("yes") {
			fmt.Printf("Adding IPv6 /64 subnet to instance: %s\n", resolvedName)
			fmt.Printf("\n💡 This will assign a new IPv6 /64 subnet to your VPS.\n")
			fmt.Printf("⚠️  A full VM restart (stop + start) will be required after assignment\n")
			fmt.Printf("   to automatically activate IPv6 networking inside the VM.\n")
			confirmed, err := promptConfirmation("Continue with IPv6 subnet assignment?")
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Printf("Operation cancelled\n")
				return nil
			}
		}

		fmt.Printf("Adding IPv6 subnet to instance: %s\n", resolvedName)

		resp, err := bwhClient.AddIPv6(ctx)
		if err != nil {
			return fmt.Errorf("failed to add IPv6 subnet: %w", err)
		}

		fmt.Printf("✅ IPv6 subnet added successfully\n")
		fmt.Printf("📋 ASSIGNED SUBNET\n")
		fmt.Printf("   IPv6 Subnet  : %s/64\n", resp.AssignedSubnet)
		fmt.Printf("\n⚠️  IMPORTANT: VM restart required for automatic IPv6 activation\n")
		fmt.Printf("   1. Stop the VM: 'bwh stop' (status must show 'Stopped')\n")
		fmt.Printf("   2. Start the VM: 'bwh start'\n")
		fmt.Printf("   This will automatically activate IPv6 networking inside the VM.\n")

		return nil
	},
}

var ipv6DeleteCmd = &cli.Command{
	Name:      "delete",
	Usage:     "release an IPv6 /64 subnet",
	ArgsUsage: "<subnet>",
	Flags:     writeFlags(),
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() != 1 {
			return fmt.Errorf("IPv6 subnet is required")
		}
		subnet := cmd.Args().First()

		// Validate IPv6 subnet format
		if !isValidIPv6Subnet(subnet) {
			return fmt.Errorf("invalid IPv6 subnet format: %s (expected format: 2001:db8:1234:5678::)", subnet)
		}

		// Normalize subnet format (remove /64 suffix if present, we'll add it back for display)
		normalizedSubnet := strings.TrimSuffix(subnet, "/64")

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		serviceInfo, err := bwhClient.GetServiceInfo(ctx)
		if err != nil {
			return fmt.Errorf("failed to get service info: %w", err)
		}
		if !hasIPv6Subnet(serviceInfo.IPAddresses, normalizedSubnet) {
			return fmt.Errorf("IPv6 subnet %s/64 is not assigned to instance %s", normalizedSubnet, resolvedName)
		}
		if cmd.Bool("dry-run") {
			printDryRun("ipv6/delete", resolvedName, fmt.Sprintf("subnet: %s/64", normalizedSubnet))
			return nil
		}

		if !cmd.Bool("yes") {
			fmt.Printf("⚠️  WARNING: This will release the IPv6 subnet and it cannot be undone.\n")
			fmt.Printf("The subnet will no longer be available to your VPS.\n")
			fmt.Printf("\nSubnet to delete: %s/64\n", normalizedSubnet)
			confirmed, err := promptConfirmation("Continue with IPv6 subnet deletion?")
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Printf("Operation cancelled\n")
				return nil
			}
		}

		fmt.Printf("Deleting IPv6 subnet '%s' from instance: %s\n", normalizedSubnet, resolvedName)

		if err := bwhClient.DeleteIPv6(ctx, normalizedSubnet); err != nil {
			return fmt.Errorf("failed to delete IPv6 subnet: %w", err)
		}

		fmt.Printf("✅ IPv6 subnet '%s' deleted successfully\n", normalizedSubnet)

		return nil
	},
}

var ipv6ListCmd = &cli.Command{
	Name:  "list",
	Usage: "list IPv6 subnets assigned to the VPS",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "compact",
			Usage: "display IPv6 information in compact format",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Getting IPv6 information for instance: %s\n", resolvedName)

		// Get service info to retrieve IPv6 information
		serviceInfo, err := bwhClient.GetServiceInfo(ctx)
		if err != nil {
			return fmt.Errorf("failed to get service info: %w", err)
		}

		// Check IPv6 availability at location
		if !serviceInfo.LocationIPv6Ready {
			fmt.Printf("\n❌ IPv6 is not available at this location (%s)\n", serviceInfo.NodeLocation)
			fmt.Printf("   IPv6 support varies by datacenter location.\n")
			return nil
		}

		// Display IPv6 information
		if cmd.Bool("compact") {
			displayIPv6InfoCompact(serviceInfo, resolvedName)
		} else {
			displayIPv6InfoDetailed(serviceInfo, resolvedName)
		}

		return nil
	},
}

func displayIPv6InfoDetailed(info *client.ServiceInfo, instanceName string) {
	fmt.Printf("\n")
	fmt.Printf("┌─────────────────────────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│                    IPv6 Information: %-35s │\n", instanceName)
	fmt.Printf("└─────────────────────────────────────────────────────────────────────────────┘\n")

	fmt.Printf("\n🌐 IPv6 STATUS\n")
	fmt.Printf("   Location Support : ✅ Available at %s\n", info.NodeLocation)
	fmt.Printf("   Plan Limit       : %d IPv6 /64 subnets maximum\n", info.PlanMaxIPv6s)

	if info.IPv6SitTunnelEndpoint != "" {
		fmt.Printf("   SIT Tunnel       : %s\n", info.IPv6SitTunnelEndpoint)
	}

	// Extract IPv6 subnets from IP addresses
	var ipv6Subnets []string
	var ipv4Addresses []string

	for _, ip := range info.IPAddresses {
		if strings.Contains(ip, ":") {
			ipv6Subnets = append(ipv6Subnets, ip)
		} else {
			ipv4Addresses = append(ipv4Addresses, ip)
		}
	}

	fmt.Printf("\n📋 ASSIGNED SUBNETS (%d/%d used)\n", len(ipv6Subnets), info.PlanMaxIPv6s)
	if len(ipv6Subnets) == 0 {
		fmt.Printf("   No IPv6 subnets assigned\n")
		fmt.Printf("   💡 Use 'bwh ipv6 add' to assign a new IPv6 /64 subnet\n")
	} else {
		for i, subnet := range ipv6Subnets {
			fmt.Printf("   %d. %s/64\n", i+1, subnet)
		}
	}

	if len(ipv4Addresses) > 0 {
		fmt.Printf("\n📍 IPv4 ADDRESSES\n")
		for i, ip := range ipv4Addresses {
			fmt.Printf("   %d. %s\n", i+1, ip)
		}
	}

	if len(info.PrivateIPAddresses) > 0 {
		fmt.Printf("\n🔒 PRIVATE IPv4 ADDRESSES\n")
		for i, ip := range info.PrivateIPAddresses {
			fmt.Printf("   %d. %s\n", i+1, ip)
		}
	}
}

func displayIPv6InfoCompact(info *client.ServiceInfo, instanceName string) {
	fmt.Printf("\nIPv6 Status: %s\n", instanceName)

	if !info.LocationIPv6Ready {
		fmt.Printf("├─ ❌ IPv6 not available at %s\n", info.NodeLocation)
		return
	}

	// Extract IPv6 subnets
	var ipv6Subnets []string
	for _, ip := range info.IPAddresses {
		if strings.Contains(ip, ":") {
			ipv6Subnets = append(ipv6Subnets, ip)
		}
	}

	fmt.Printf("├─ ✅ IPv6 available at %s\n", info.NodeLocation)
	fmt.Printf("├─ Quota: %d/%d subnets used\n", len(ipv6Subnets), info.PlanMaxIPv6s)

	if len(ipv6Subnets) == 0 {
		fmt.Printf("└─ No IPv6 subnets assigned\n")
	} else {
		for i, subnet := range ipv6Subnets {
			if i == len(ipv6Subnets)-1 {
				fmt.Printf("└─ %s/64\n", subnet)
			} else {
				fmt.Printf("├─ %s/64\n", subnet)
			}
		}
	}
}

func countIPv6Subnets(ips []string) int {
	count := 0
	for _, ip := range ips {
		if isIPv6Address(ip) {
			count++
		}
	}
	return count
}

func hasIPv6Subnet(ips []string, subnet string) bool {
	normalized := trimIPv6Subnet(subnet)
	for _, ip := range ips {
		if trimIPv6Subnet(ip) == normalized {
			return true
		}
	}
	return false
}

// isValidIPv6Subnet validates if the given string is a valid IPv6 address
func isValidIPv6Subnet(subnet string) bool {
	// Remove /64 suffix if present
	subnet = strings.TrimSuffix(subnet, "/64")

	// Parse as IPv6 address
	ip := net.ParseIP(subnet)
	if ip == nil {
		return false
	}

	// Check if it's IPv6 (not IPv4)
	return ip.To4() == nil
}
