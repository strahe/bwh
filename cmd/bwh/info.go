package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/strahe/bwh/pkg/client"
	"github.com/urfave/cli/v3"
)

var infoCmd = &cli.Command{
	Name:  "info",
	Usage: "display comprehensive information about the BWH instance",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "compact",
			Usage: "display information in compact format",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Getting info for instance: %s\n", resolvedName)
		fmt.Printf("⏳ This may take up to 15 seconds...\n")

		// Get live service info (contains all data)
		liveInfo, err := bwhClient.GetLiveServiceInfo(ctx)
		if err != nil {
			return fmt.Errorf("failed to get service info: %w", err)
		}

		// Display information
		if cmd.Bool("compact") {
			displayCompactInfo(liveInfo, resolvedName)
		} else {
			displayDetailedInfo(liveInfo, resolvedName)
		}

		return nil
	},
}

// displayDetailedInfo displays comprehensive BWH instance information
func displayDetailedInfo(info *client.LiveServiceInfo, instanceName string) {
	// Header with instance name
	fmt.Printf("\n")
	fmt.Printf("┌─────────────────────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│                          BWH Instance: %-32s │\n", instanceName)
	fmt.Printf("└─────────────────────────────────────────────────────────────────────────┘\n")

	// Basic Information
	fmt.Printf("\n📋 BASIC INFORMATION\n")
	fmt.Printf("   Hostname         : %s", info.Hostname)
	if info.LiveHostname != "" && info.LiveHostname != info.Hostname {
		fmt.Printf(" (live: %s)", info.LiveHostname)
	}
	fmt.Printf("\n")
	fmt.Printf("   VM Type          : %s", info.VMType)
	if info.VeStatus != "" {
		statusIcon := getStatusIcon(info.VeStatus)
		fmt.Printf(" | Status: %s %s", statusIcon, info.VeStatus)
	}
	fmt.Printf("\n")
	fmt.Printf("   Plan             : %s\n", info.Plan)
	fmt.Printf("   Operating System : %s\n", info.OS)
	fmt.Printf("   Email            : %s\n", info.Email)
	if info.SSHPort > 0 {
		fmt.Printf("   SSH Port         : %d\n", info.SSHPort)
	}
	if info.VeMac1 != "" {
		fmt.Printf("   MAC Address      : %s\n", info.VeMac1)
	}

	// Location & Infrastructure
	fmt.Printf("\n🌍 LOCATION & INFRASTRUCTURE\n")
	fmt.Printf("   Node Alias       : %s\n", info.NodeAlias)
	fmt.Printf("   Location         : %s (ID: %s)\n", info.NodeLocation, info.NodeLocationID)
	fmt.Printf("   Datacenter       : %s\n", info.NodeDatacenter)
	fmt.Printf("   IPv6 Ready       : %s (location: %v)\n", formatBool(info.LocationIPv6Ready), info.LocationIPv6Ready)

	// Resource Allocation & Usage
	fmt.Printf("\n💾 RESOURCE ALLOCATION & USAGE\n")

	// Memory
	fmt.Printf("   Memory (RAM)     : %s total", formatBytes(info.PlanRAM))
	if info.MemAvailableKB.Value > 0 && info.PlanRAM > 0 {
		availableRAM := info.MemAvailableKB.Value * 1024
		usedRAM := info.PlanRAM - availableRAM
		usagePercent := float64(usedRAM) / float64(info.PlanRAM) * 100
		fmt.Printf(" | %s used (%.1f%%) | %s available",
			formatBytes(usedRAM), usagePercent, formatBytes(availableRAM))
	} else if info.MemAvailableKB.Value > 0 {
		availableRAM := info.MemAvailableKB.Value * 1024
		usedRAM := info.PlanRAM - availableRAM
		fmt.Printf(" | %s used | %s available",
			formatBytes(usedRAM), formatBytes(availableRAM))
	}
	fmt.Printf("\n")

	// Disk
	fmt.Printf("   Disk Space       : %s total", formatBytes(info.PlanDisk))
	if info.VeUsedDiskSpaceB.Value > 0 && info.PlanDisk > 0 {
		usagePercent := float64(info.VeUsedDiskSpaceB.Value) / float64(info.PlanDisk) * 100
		availableDisk := info.PlanDisk - info.VeUsedDiskSpaceB.Value
		fmt.Printf(" | %s used (%.1f%%) | %s available",
			formatBytes(info.VeUsedDiskSpaceB.Value), usagePercent, formatBytes(availableDisk))
	} else if info.VeUsedDiskSpaceB.Value > 0 {
		availableDisk := info.PlanDisk - info.VeUsedDiskSpaceB.Value
		fmt.Printf(" | %s used | %s available",
			formatBytes(info.VeUsedDiskSpaceB.Value), formatBytes(availableDisk))
	}
	if info.VeDiskQuotaGB.Value > 0 {
		fmt.Printf(" | %d GB quota", info.VeDiskQuotaGB.Value)
	}
	fmt.Printf("\n")

	// Swap
	if info.SwapTotalKB.Value > 0 {
		// Use actual swap data when available
		swapTotal := info.SwapTotalKB.Value * 1024
		swapAvailable := info.SwapAvailableKB.Value * 1024
		swapUsed := swapTotal - swapAvailable
		fmt.Printf("   Swap             : %s total", formatBytes(swapTotal))
		if swapTotal > 0 {
			swapUsagePercent := float64(swapUsed) / float64(swapTotal) * 100
			fmt.Printf(" | %s used (%.1f%%) | %s available",
				formatBytes(swapUsed), swapUsagePercent, formatBytes(swapAvailable))
		} else {
			fmt.Printf(" | %s used | %s available",
				formatBytes(swapUsed), formatBytes(swapAvailable))
		}
	} else {
		// Fallback to plan data
		fmt.Printf("   Swap             : %s total", formatBytes(info.PlanSwap))
	}
	fmt.Printf("\n")

	// System Performance
	if info.LoadAverage != "" || info.IsCPUThrottled.Value == 1 || info.IsDiskThrottled.Value == 1 {
		fmt.Printf("\n⚡ SYSTEM PERFORMANCE\n")
		if info.LoadAverage != "" {
			fmt.Printf("   Load Average     : %s\n", info.LoadAverage)
		}
		if info.IsCPUThrottled.Value == 1 {
			fmt.Printf("   CPU Status       : 🔴 THROTTLED (resets automatically every 2 hours)\n")
		} else {
			fmt.Printf("   CPU Status       : ✅ Normal (throttled: %d)\n", info.IsCPUThrottled.Value)
		}
		if info.VMType == "kvm" {
			if info.IsDiskThrottled.Value == 1 {
				fmt.Printf("   Disk I/O Status  : 🔴 THROTTLED (resets automatically in 15-180 minutes)\n")
			} else {
				fmt.Printf("   Disk I/O Status  : ✅ Normal (throttled: %d)\n", info.IsDiskThrottled.Value)
			}
		}
	}

	// Data Transfer
	displayBandwidthInfo(&info.ServiceInfo)

	// Network Configuration
	fmt.Printf("\n🌐 NETWORK CONFIGURATION\n")
	if len(info.IPAddresses) > 0 {
		fmt.Printf("   Public IPs       : %s\n", strings.Join(info.IPAddresses, ", "))
	} else {
		fmt.Printf("   Public IPs       : None\n")
	}

	if len(info.PrivateIPAddresses) > 0 {
		fmt.Printf("   Private IPs      : %s\n", strings.Join(info.PrivateIPAddresses, ", "))
	}

	if len(info.IPNullroutes) > 0 {
		fmt.Printf("   ⚠️  DDoS Protection  : %d IP(s) currently null-routed\n", len(info.IPNullroutes))
		fmt.Printf("   Null-routed IPs  : %s\n", strings.Join(info.IPNullroutes, ", "))
	}

	if info.IPv6SitTunnelEndpoint != "" {
		fmt.Printf("   IPv6 SIT Tunnel  : %s\n", info.IPv6SitTunnelEndpoint)
	}

	fmt.Printf("   Max IPv6 Subnets : %d /64 subnet(s)\n", info.PlanMaxIPv6s)

	// Network Features
	fmt.Printf("\n🔧 NETWORK FEATURES\n")
	fmt.Printf("   Private Network  : %s (plan: %v, location: %v)\n",
		formatNetworkFeature(info.PlanPrivateNetworkAvailable, info.LocationPrivateNetworkAvailable),
		info.PlanPrivateNetworkAvailable, info.LocationPrivateNetworkAvailable)
	fmt.Printf("   RDNS API         : %s (available: %v)\n", formatBool(info.RDNSAPIAvailable), info.RDNSAPIAvailable)
	if info.FreeIPReplacementInterval > 0 {
		fmt.Printf("   IP Replacement   : Every %d hours\n", info.FreeIPReplacementInterval)
	} else {
		fmt.Printf("   IP Replacement   : ❌ Not available (deprecated feature)\n")
	}

	// PTR Records
	if len(info.PTR) > 0 {
		fmt.Printf("\n🔍 PTR RECORDS\n")
		for ip, ptr := range info.PTR {
			fmt.Printf("   %-15s : %s\n", ip, ptr)
		}
	}

	// ISO Images
	displayISOInfo(&info.ServiceInfo)

	// Security & Account Status
	displaySecurityInfo(&info.ServiceInfo)

	// OpenVZ specific information
	if info.VMType == "ovz" && (len(info.VzStatus) > 0 || len(info.VzQuota) > 0) {
		fmt.Printf("\n🔧 OPENVZ DETAILS\n")
		if len(info.VzStatus) > 0 {
			fmt.Printf("   System Status:\n")
			for key, value := range info.VzStatus {
				fmt.Printf("     %-16s : %v\n", key, value)
			}
		}
		if len(info.VzQuota) > 0 {
			fmt.Printf("   Quota Information:\n")
			for key, value := range info.VzQuota {
				fmt.Printf("     %-16s : %v\n", key, value)
			}
		}
	}

	fmt.Printf("\n")
}

// displayCompactInfo displays BWH instance information in compact format
func displayCompactInfo(info *client.LiveServiceInfo, instanceName string) {
	fmt.Printf("\nInstance: %s\n", instanceName)
	fmt.Printf("├─ Host: %s (%s)\n", info.Hostname, info.VMType)
	fmt.Printf("├─ Plan: %s\n", info.Plan)
	fmt.Printf("├─ OS: %s\n", info.OS)
	fmt.Printf("├─ Location: %s (%s)\n", info.NodeLocation, info.NodeAlias)

	// VPS Status for KVM
	if info.VMType == "kvm" && info.VeStatus != "" {
		statusIcon := getStatusIcon(info.VeStatus)
		fmt.Printf("├─ VPS Status: %s %s\n", statusIcon, info.VeStatus)
	}

	// Resources with usage
	fmt.Printf("├─ Resources:\n")

	// Memory
	memLine := fmt.Sprintf("│  ├─ RAM: %s", formatBytes(info.PlanRAM))
	if info.MemAvailableKB.Value > 0 && info.PlanRAM > 0 {
		availableRAM := info.MemAvailableKB.Value * 1024
		usedRAM := info.PlanRAM - availableRAM
		usagePercent := float64(usedRAM) / float64(info.PlanRAM) * 100
		memLine += fmt.Sprintf(" (%s used, %.1f%%)", formatBytes(usedRAM), usagePercent)
	} else if info.MemAvailableKB.Value > 0 {
		availableRAM := info.MemAvailableKB.Value * 1024
		usedRAM := info.PlanRAM - availableRAM
		memLine += fmt.Sprintf(" (%s used)", formatBytes(usedRAM))
	}
	fmt.Printf("%s\n", memLine)

	// Disk
	diskLine := fmt.Sprintf("│  ├─ Disk: %s", formatBytes(info.PlanDisk))
	if info.VeUsedDiskSpaceB.Value > 0 && info.PlanDisk > 0 {
		usagePercent := float64(info.VeUsedDiskSpaceB.Value) / float64(info.PlanDisk) * 100
		diskLine += fmt.Sprintf(" (%s used, %.1f%%)", formatBytes(info.VeUsedDiskSpaceB.Value), usagePercent)
	} else if info.VeUsedDiskSpaceB.Value > 0 {
		diskLine += fmt.Sprintf(" (%s used)", formatBytes(info.VeUsedDiskSpaceB.Value))
	}
	fmt.Printf("%s\n", diskLine)

	// Swap
	var swapLine string
	if info.SwapTotalKB.Value > 0 {
		// Use actual swap data when available
		swapTotal := info.SwapTotalKB.Value * 1024
		swapUsed := (info.SwapTotalKB.Value - info.SwapAvailableKB.Value) * 1024
		swapLine = fmt.Sprintf("│  └─ Swap: %s", formatBytes(swapTotal))
		if swapTotal > 0 {
			swapUsagePercent := float64(swapUsed) / float64(swapTotal) * 100
			swapLine += fmt.Sprintf(" (%s used, %.1f%%)", formatBytes(swapUsed), swapUsagePercent)
		} else {
			swapLine += fmt.Sprintf(" (%s used)", formatBytes(swapUsed))
		}
	} else {
		// Fallback to plan data
		swapLine = fmt.Sprintf("│  └─ Swap: %s", formatBytes(info.PlanSwap))
	}
	fmt.Printf("%s\n", swapLine)

	// Network
	if len(info.IPAddresses) > 0 {
		fmt.Printf("├─ IPs: %s\n", strings.Join(info.IPAddresses, ", "))
	}

	// Performance issues
	throttleStatus := []string{}
	if info.IsCPUThrottled.Value == 1 {
		throttleStatus = append(throttleStatus, "CPU")
	}
	if info.IsDiskThrottled.Value == 1 {
		throttleStatus = append(throttleStatus, "Disk I/O")
	}
	if len(throttleStatus) > 0 {
		fmt.Printf("├─ Throttled: 🔴 %s\n", strings.Join(throttleStatus, ", "))
	}

	// Load average for KVM
	if info.LoadAverage != "" {
		fmt.Printf("├─ Load Average: %s\n", info.LoadAverage)
	}

	// Bandwidth
	actualMonthlyLimit := info.PlanMonthlyData * int64(info.MonthlyDataMultiplier)
	actualDataUsed := info.DataCounter * int64(info.MonthlyDataMultiplier)

	if actualMonthlyLimit > 0 {
		usagePercent := float64(actualDataUsed) / float64(actualMonthlyLimit) * 100
		fmt.Printf("├─ Bandwidth: %s / %s (%.1f%%)\n",
			formatBytes(actualDataUsed),
			formatBytes(actualMonthlyLimit),
			usagePercent)
	} else {
		fmt.Printf("├─ Bandwidth: %s / %s\n",
			formatBytes(actualDataUsed),
			formatBytes(actualMonthlyLimit))
	}
	if info.MonthlyDataMultiplier > 1 {
		fmt.Printf("├─ Multiplier: %dx (expensive location)\n", info.MonthlyDataMultiplier)
	}

	// Status
	status := "Active"
	if info.Suspended {
		status = "SUSPENDED"
	}
	fmt.Printf("└─ Status: %s", status)

	if info.PolicyViolation {
		fmt.Printf(" | Policy Violation: YES")
	}
	if info.TotalAbusePoints > 0 {
		fmt.Printf(" | Abuse: %d/%d", info.TotalAbusePoints, info.MaxAbusePoints)
	}
	fmt.Printf("\n\n")
}

// Helper functions
func getStatusIcon(status string) string {
	switch strings.ToLower(status) {
	case "running":
		return "🟢"
	case "stopped":
		return "🔴"
	case "starting":
		return "🟡"
	default:
		return "❓"
	}
}

func displayBandwidthInfo(info *client.ServiceInfo) {
	fmt.Printf("\n📊 DATA TRANSFER\n")

	// Apply bandwidth multiplier for expensive locations
	actualMonthlyLimit := info.PlanMonthlyData * int64(info.MonthlyDataMultiplier)
	actualDataUsed := info.DataCounter * int64(info.MonthlyDataMultiplier)

	fmt.Printf("   Monthly Limit    : %s\n", formatBytes(actualMonthlyLimit))
	if actualMonthlyLimit > 0 {
		usagePercent := float64(actualDataUsed) / float64(actualMonthlyLimit) * 100
		fmt.Printf("   Used This Month  : %s (%.1f%%)\n", formatBytes(actualDataUsed), usagePercent)

	} else {
		fmt.Printf("   Used This Month  : %s\n", formatBytes(actualDataUsed))
	}
	fmt.Printf("   Remaining        : %s\n", formatBytes(actualMonthlyLimit-actualDataUsed))
	if info.MonthlyDataMultiplier > 1 {
		fmt.Printf("   Data Multiplier  : %dx (expensive bandwidth location)\n", info.MonthlyDataMultiplier)
		fmt.Printf("   Base Limit       : %s\n", formatBytes(info.PlanMonthlyData))
		fmt.Printf("   Base Used        : %s\n", formatBytes(info.DataCounter))
	} else {
		fmt.Printf("   Data Multiplier  : %dx\n", info.MonthlyDataMultiplier)
	}

	if info.DataNextReset > 0 {
		resetTime := time.Unix(info.DataNextReset, 0).Local()
		fmt.Printf("   Next Reset       : %s\n", resetTime.Format("2006-01-02 15:04:05"))
	}
}

func displayISOInfo(info *client.ServiceInfo) {
	if info.ISO1 != "" || info.ISO2 != "" || len(info.AvailableISOs) > 0 {
		fmt.Printf("\n💿 ISO MANAGEMENT\n")
		if info.ISO1 != "" {
			fmt.Printf("   💿 Mounted ISO 1   : %s\n", info.ISO1)
		}
		if info.ISO2 != "" {
			fmt.Printf("   💿 Mounted ISO 2   : %s (currently unsupported)\n", info.ISO2)
		} else if info.ISO1 == "" {
			fmt.Printf("   Mounted ISOs     : None\n")
		}
		if len(info.AvailableISOs) > 0 {
			fmt.Printf("   Available ISOs   : %d images ready for mounting\n", len(info.AvailableISOs))
		}
	}
}

func displaySecurityInfo(info *client.ServiceInfo) {
	fmt.Printf("\n🔒 SECURITY & ACCOUNT STATUS\n")

	if info.Suspended {
		fmt.Printf("   Service Status   : 🚫 SUSPENDED (suspended: %v)\n", info.Suspended)
	} else {
		fmt.Printf("   Service Status   : ✅ Active (suspended: %v)\n", info.Suspended)
	}

	if info.PolicyViolation {
		fmt.Printf("   Policy Violation : ⚠️  YES - Attention required (violation: %v)\n", info.PolicyViolation)
	} else {
		fmt.Printf("   Policy Violation : ✅ No violations (violation: %v)\n", info.PolicyViolation)
	}

	fmt.Printf("   Suspensions (YTD): %d time(s) in current calendar year\n", info.SuspensionCount)
	fmt.Printf("   Abuse Points     : %d / %d (calendar year)\n", info.TotalAbusePoints, info.MaxAbusePoints)

	if info.TotalAbusePoints > 0 && info.MaxAbusePoints > 0 {
		abusePercent := float64(info.TotalAbusePoints) / float64(info.MaxAbusePoints) * 100
		if abusePercent > 80 {
			fmt.Printf("   Abuse Level      : 🔴 %.1f%% (HIGH RISK)\n", abusePercent)
		} else if abusePercent > 50 {
			fmt.Printf("   Abuse Level      : 🟡 %.1f%% (MEDIUM RISK)\n", abusePercent)
		} else {
			fmt.Printf("   Abuse Level      : 🟢 %.1f%% (LOW RISK)\n", abusePercent)
		}
	} else if info.TotalAbusePoints > 0 {
		fmt.Printf("   Abuse Level      : ⚠️  %d points (no limit data)\n", info.TotalAbusePoints)
	} else {
		fmt.Printf("   Abuse Level      : ✅ 0%% (Clean record)\n")
	}
}

// formatBool converts boolean to readable yes/no
func formatBool(b bool) string {
	if b {
		return "✅ Yes"
	}
	return "❌ No"
}

// formatNetworkFeature formats network feature availability
func formatNetworkFeature(planSupports, locationSupports bool) string {
	if planSupports && locationSupports {
		return "✅ Available"
	} else if planSupports && !locationSupports {
		return "⚠️  Plan supports, but not available in this location"
	} else if !planSupports && locationSupports {
		return "⚠️  Available in location, but not in current plan"
	}
	return "❌ Not available"
}
