package main

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"
)

var privateIPCmd = &cli.Command{
	Name:    "private-ip",
	Usage:   "manage Private IPv4 addresses",
	Aliases: []string{"pi"},
	Commands: []*cli.Command{
		privateIPInfoCmd,
		privateIPListAvailableCmd,
		privateIPAssignCmd,
		privateIPDeleteCmd,
	},
}

var privateIPInfoCmd = &cli.Command{
	Name:  "info",
	Usage: "show private IPv4 information for the VPS",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Getting private IPv4 info for instance: %s\n", resolvedName)

		serviceInfo, err := bwhClient.GetServiceInfo(ctx)
		if err != nil {
			return fmt.Errorf("failed to get service info: %w", err)
		}

		fmt.Printf("\n🔒 PRIVATE IPv4 STATUS\n")
		fmt.Printf("   Plan Support    : %s\n", yesNo(serviceInfo.PlanPrivateNetworkAvailable))
		fmt.Printf("   Location Support: %s\n", yesNo(serviceInfo.LocationPrivateNetworkAvailable))

		ips := serviceInfo.PrivateIPAddresses
		fmt.Printf("\n📋 ASSIGNED PRIVATE IPv4 ADDRESSES (%d)\n", len(ips))
		if len(ips) == 0 {
			fmt.Printf("   No private IPv4 addresses assigned\n")
			fmt.Printf("   💡 Use 'bwh private-ip assign' to assign a private IP\n")
		} else {
			for i, ip := range ips {
				fmt.Printf("   %d. %s\n", i+1, ip)
			}
		}
		return nil
	},
}

var privateIPListAvailableCmd = &cli.Command{
	Name:  "available",
	Usage: "list available private IPv4 addresses that can be assigned",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"a"},
			Usage:   "list all available IPs without aggregation",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Getting available private IPv4 addresses for instance: %s\n", resolvedName)

		resp, err := bwhClient.GetAvailablePrivateIPs(ctx)
		if err != nil {
			return fmt.Errorf("failed to get available private IPs: %w", err)
		}

		if len(resp.AvailableIPs) == 0 {
			fmt.Printf("No available private IPv4 addresses.\n")
			return nil
		}

		if cmd.Bool("all") {
			fmt.Printf("\n📋 AVAILABLE PRIVATE IPv4 ADDRESSES (%d)\n", len(resp.AvailableIPs))
			for i, ip := range resp.AvailableIPs {
				fmt.Printf("   %d. %s\n", i+1, ip)
			}
			return nil
		}

		ranges, total := aggregateIPv4Ranges(resp.AvailableIPs)
		fmt.Printf("\n📋 AVAILABLE PRIVATE IPv4 RANGES (%d ranges, %d IPs)\n", len(ranges), total)
		for i, r := range ranges {
			fmt.Printf("   %d. %s\n", i+1, r)
		}
		return nil
	},
}

var privateIPAssignCmd = &cli.Command{
	Name:      "assign",
	Usage:     "assign a private IPv4 address (random if not specified)",
	ArgsUsage: "[ip]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "skip confirmation prompt",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		var ip string
		if cmd.Args().Len() > 0 {
			ip = cmd.Args().First()
			if parsed := net.ParseIP(ip); parsed == nil || parsed.To4() == nil {
				return fmt.Errorf("invalid IPv4 address: %s", ip)
			}
		}

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		if !cmd.Bool("yes") {
			if ip == "" {
				fmt.Printf("This will assign a random private IPv4 address to instance: %s\n", resolvedName)
			} else {
				fmt.Printf("This will assign private IPv4 address %s to instance: %s\n", ip, resolvedName)
			}
			confirmed, err := promptConfirmation("Proceed?")
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Printf("Operation cancelled\n")
				return nil
			}
		}

		resp, err := bwhClient.AssignPrivateIP(ctx, ip)
		if err != nil {
			return fmt.Errorf("failed to assign private IP: %w", err)
		}

		fmt.Printf("✅ Private IP assigned successfully\n")
		if len(resp.AssignedIPs) > 0 {
			fmt.Printf("\n📋 ASSIGNED PRIVATE IPv4 ADDRESSES\n")
			for i, assigned := range resp.AssignedIPs {
				fmt.Printf("   %d. %s\n", i+1, assigned)
			}
		}
		return nil
	},
}

var privateIPDeleteCmd = &cli.Command{
	Name:      "delete",
	Usage:     "delete a private IPv4 address",
	ArgsUsage: "<ip>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "skip confirmation prompt",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() != 1 {
			return fmt.Errorf("private IPv4 address is required")
		}
		ip := cmd.Args().First()
		if parsed := net.ParseIP(ip); parsed == nil || parsed.To4() == nil {
			return fmt.Errorf("invalid IPv4 address: %s", ip)
		}

		if !cmd.Bool("yes") {
			fmt.Printf("⚠️  This will delete private IPv4 address %s from the instance.\n", ip)
			confirmed, err := promptConfirmation("Proceed with deletion?")
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Printf("Operation cancelled\n")
				return nil
			}
		}

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Deleting private IPv4 address '%s' from instance: %s\n", ip, resolvedName)

		if err := bwhClient.DeletePrivateIP(ctx, ip); err != nil {
			return fmt.Errorf("failed to delete private IP: %w", err)
		}

		fmt.Printf("✅ Private IPv4 address '%s' deleted successfully\n", ip)
		return nil
	},
}

// display helper for possible reuse; currently not used besides inline prints
func displayPrivateIPs(title string, ips []string, instanceName string) {
	fmt.Printf("\n📋 %s (%d)\n", title, len(ips))
	for i, ip := range ips {
		fmt.Printf("   %d. %s\n", i+1, ip)
	}
}

// yesNo converts a boolean to user-friendly Yes/No string
func yesNo(b bool) string {
	if b {
		return "✅ Yes"
	}
	return "❌ No"
}

// aggregateIPv4Ranges groups contiguous IPv4 addresses into concise ranges.
// If start and end share the same first three octets, prints as A.B.C.start-endD (e.g., 10.59.12.26-254).
// Otherwise prints as startIP-endIP. Singletons are printed as the single IP.
func aggregateIPv4Ranges(ips []string) ([]string, int) {
	nums := make([]uint32, 0, len(ips))
	seen := make(map[uint32]struct{}, len(ips))
	for _, s := range ips {
		if n, ok := ipv4ToUint32(s); ok {
			if _, exists := seen[n]; !exists {
				seen[n] = struct{}{}
				nums = append(nums, n)
			}
		}
	}
	if len(nums) == 0 {
		return []string{}, 0
	}

	sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })

	var (
		ranges []string
		start  = nums[0]
		prev   = nums[0]
		total  = len(nums)
	)

	flush := func(a, b uint32) {
		if a == b {
			ranges = append(ranges, uint32ToIPv4(a))
			return
		}
		as, bs := uint32ToIPv4(a), uint32ToIPv4(b)
		a1, a2, a3, a4 := splitIPv4(as)
		b1, b2, b3, b4 := splitIPv4(bs)
		if a1 == b1 && a2 == b2 && a3 == b3 {
			ranges = append(ranges, fmt.Sprintf("%d.%d.%d.%d-%d", a1, a2, a3, a4, b4))
		} else {
			ranges = append(ranges, fmt.Sprintf("%s-%s", as, bs))
		}
	}

	for i := 1; i < len(nums); i++ {
		n := nums[i]
		if n == prev || n == prev+1 {
			prev = n
			continue
		}
		flush(start, prev)
		start, prev = n, n
	}
	flush(start, prev)
	return ranges, total
}

func ipv4ToUint32(s string) (uint32, bool) {
	ip := net.ParseIP(s)
	if ip == nil {
		return 0, false
	}
	v4 := ip.To4()
	if v4 == nil {
		return 0, false
	}
	return uint32(v4[0])<<24 | uint32(v4[1])<<16 | uint32(v4[2])<<8 | uint32(v4[3]), true
}

func uint32ToIPv4(n uint32) string {
	a := byte(n >> 24)
	b := byte(n >> 16)
	c := byte(n >> 8)
	d := byte(n)
	return fmt.Sprintf("%d.%d.%d.%d", a, b, c, d)
}

func splitIPv4(s string) (int, int, int, int) {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return 0, 0, 0, 0
	}
	p1, _ := strconv.Atoi(parts[0])
	p2, _ := strconv.Atoi(parts[1])
	p3, _ := strconv.Atoi(parts[2])
	p4, _ := strconv.Atoi(parts[3])
	return p1, p2, p3, p4
}
