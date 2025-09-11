package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/guptarohit/asciigraph"
	"github.com/strahe/bwh/pkg/client"
	"github.com/urfave/cli/v3"
)

var usageCmd = &cli.Command{
	Name:  "usage",
	Usage: "display detailed VPS usage statistics",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "compact",
			Usage: "display usage in compact format",
		},
		&cli.StringFlag{
			Name:  "period",
			Usage: "time period to display: 1d, 7d, 1m, all",
			Value: "1d",
		},
		&cli.BoolFlag{
			Name:  "summary",
			Usage: "show summary statistics only",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		compact := cmd.Bool("compact")
		period := cmd.String("period")
		summaryOnly := cmd.Bool("summary")

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Getting usage statistics for instance: %s\n", resolvedName)

		// Get usage statistics
		usageStats, err := bwhClient.GetRawUsageStats(ctx)
		if err != nil {
			return fmt.Errorf("failed to get usage statistics: %w", err)
		}

		if len(usageStats.Data) == 0 {
			fmt.Printf("No usage data available for instance: %s\n", resolvedName)
			return nil
		}

		// Get bandwidth information for total traffic display
		serviceInfo, err := bwhClient.GetServiceInfo(ctx)
		if err != nil {
			return fmt.Errorf("failed to get service info: %w", err)
		}

		// Sort data by timestamp (oldest first for proper trend display)
		sort.Slice(usageStats.Data, func(i, j int) bool {
			return usageStats.Data[i].Timestamp < usageStats.Data[j].Timestamp
		})

		// Filter data by time period
		displayData := filterDataByPeriod(usageStats.Data, period)

		// Display data
		if summaryOnly {
			displayUsageSummary(usageStats, resolvedName, len(displayData), period, serviceInfo)
		} else if compact {
			displayCompactUsage(usageStats, resolvedName, displayData, period, serviceInfo)
		} else {
			displayDetailedUsageCharts(resolvedName, displayData, serviceInfo)
		}

		return nil
	},
}

func displayDetailedUsageCharts(instanceName string, data []client.UsageDataPoint, serviceInfo *client.ServiceInfo) {
	fmt.Printf("\n")
	fmt.Printf("┌─────────────────────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│                    Usage Trends: %-35s │\n", instanceName)
	fmt.Printf("└─────────────────────────────────────────────────────────────────────────┘\n")

	timeSpan := ""
	if len(data) > 1 {
		// Calculate the actual time coverage: from previous data point to last data point
		// Each data point represents a 5-minute period, so previous point is 5 minutes earlier
		startTime := time.Unix(data[0].Timestamp-300, 0) // Previous data point (5 min = 300 sec)
		endTime := time.Unix(data[len(data)-1].Timestamp, 0)
		duration := endTime.Sub(startTime)
		timeSpan = fmt.Sprintf(" (last %s)", formatDuration(duration))
	}

	fmt.Printf("\n📊 USAGE TRENDS - %d data points%s\n", len(data), timeSpan)

	if len(data) < 2 {
		fmt.Printf("Not enough data points to display charts.\n")
		return
	}

	timeRange := getTimeRange(data)
	duration := time.Unix(data[len(data)-1].Timestamp, 0).Sub(time.Unix(data[0].Timestamp, 0))

	// CPU Usage Chart
	fmt.Printf("\n🔥 CPU Usage (%%) %s\n", timeRange)
	cpuData := make([]float64, len(data))
	for i, point := range data {
		cpuData[i] = float64(point.CPUUsage)
	}

	cpuGraph := asciigraph.Plot(cpuData,
		asciigraph.Height(8),
		asciigraph.Width(70),
		asciigraph.Caption("CPU Usage Over Time"))
	fmt.Printf("%s\n", cpuGraph)
	fmt.Printf("\nRange: %.0f%% - %.0f%% | Average: %.1f%%\n",
		min(cpuData), max(cpuData), avg(cpuData))

	fmt.Print("\n" + strings.Repeat("─", 70) + "\n")

	// Combined Disk I/O Chart
	fmt.Printf("\n💾 Disk I/O Activity (KB) %s\n", timeRange)
	diskReadData := make([]float64, len(data))
	diskWriteData := make([]float64, len(data))

	for i, point := range data {
		diskReadData[i] = float64(point.DiskReadBytes) / 1024
		diskWriteData[i] = float64(point.DiskWriteBytes) / 1024
	}

	avgReadPerSec := sum(diskReadData) / duration.Seconds()
	avgWritePerSec := sum(diskWriteData) / duration.Seconds()

	diskGraph := asciigraph.PlotMany([][]float64{diskReadData, diskWriteData},
		asciigraph.Height(8),
		asciigraph.Width(70),
		asciigraph.SeriesColors(asciigraph.Green, asciigraph.Yellow),
		asciigraph.Caption("Disk I/O (Read: Green, Write: Yellow)"))
	fmt.Printf("%s\n", diskGraph)
	fmt.Printf("Read Total: %s | Average: %s\n",
		formatBytes(int64(sum(diskReadData)*1024)), formatDiskRate(avgReadPerSec, "KB"))
	fmt.Printf("Write Total: %s | Average: %s\n",
		formatBytes(int64(sum(diskWriteData)*1024)), formatDiskRate(avgWritePerSec, "KB"))

	fmt.Print("\n" + strings.Repeat("─", 70) + "\n")

	// Combined Network Traffic Chart
	fmt.Printf("\n🌐 Network Traffic (MB) %s\n", timeRange)
	netInData := make([]float64, len(data))
	netOutData := make([]float64, len(data))

	for i, point := range data {
		netInData[i] = float64(point.NetworkInBytes) / 1024 / 1024
		netOutData[i] = float64(point.NetworkOutBytes) / 1024 / 1024
	}

	avgInPerHour := sum(netInData) / duration.Hours()
	avgOutPerHour := sum(netOutData) / duration.Hours()

	netInGraph := asciigraph.PlotMany([][]float64{netInData, netOutData},
		asciigraph.Height(8),
		asciigraph.Width(70),
		asciigraph.SeriesColors(asciigraph.Blue, asciigraph.Red),
		asciigraph.Caption("Network Traffic (In: Blue, Out: Red)"))
	fmt.Printf("%s\n", netInGraph)
	fmt.Printf("Incoming Total: %s | Average: %s\n",
		formatBytes(int64(sum(netInData)*1024*1024)), formatRate(avgInPerHour, "MB"))
	fmt.Printf("Outgoing Total: %s | Average: %s\n",
		formatBytes(int64(sum(netOutData)*1024*1024)), formatRate(avgOutPerHour, "MB"))

	// Display total bandwidth information
	if serviceInfo != nil {
		displayBandwidthSummary(serviceInfo)
	}
}

// displayBandwidthSummary displays monthly bandwidth information in a compact format for usage charts
func displayBandwidthSummary(serviceInfo *client.ServiceInfo) {
	// Apply bandwidth multiplier for expensive locations
	actualMonthlyLimit := serviceInfo.PlanMonthlyData * int64(serviceInfo.MonthlyDataMultiplier)
	actualDataUsed := serviceInfo.DataCounter * int64(serviceInfo.MonthlyDataMultiplier)

	fmt.Printf("\n📊 Monthly Bandwidth\n")
	fmt.Printf("Used: %s / %s", formatBytes(actualDataUsed), formatBytes(actualMonthlyLimit))

	if actualMonthlyLimit > 0 {
		usagePercent := float64(actualDataUsed) / float64(actualMonthlyLimit) * 100
		fmt.Printf(" (%.1f%%)", usagePercent)
	}

	remaining := actualMonthlyLimit - actualDataUsed
	if remaining > 0 {
		fmt.Printf(" | Remaining: %s", formatBytes(remaining))
	} else {
		fmt.Printf(" | ⚠️  Over limit by: %s", formatBytes(-remaining))
	}

	if serviceInfo.MonthlyDataMultiplier > 1 {
		fmt.Printf(" [%dx multiplier]", serviceInfo.MonthlyDataMultiplier)
	}

	if serviceInfo.DataNextReset > 0 {
		resetTime := time.Unix(serviceInfo.DataNextReset, 0).Local()
		fmt.Printf("\nNext reset: %s", resetTime.Format("2006-01-02 15:04"))
	}
}

func displayCompactUsage(stats *client.UsageStatsResponse, instanceName string, data []client.UsageDataPoint, period string, serviceInfo *client.ServiceInfo) {
	fmt.Printf("\nUsage Summary: %s (%s)\n", instanceName, stats.VMType)
	fmt.Printf("├─ %d data points (period: %s)", len(data), period)

	if len(data) > 1 {
		// Calculate the actual time coverage: from previous data point to last data point
		startTime := time.Unix(data[0].Timestamp-300, 0) // Previous data point (5 min = 300 sec)
		endTime := time.Unix(data[len(data)-1].Timestamp, 0)
		timeRange := endTime.Sub(startTime)
		fmt.Printf(" over %s\n", formatDuration(timeRange))
	} else {
		fmt.Printf("\n")
	}

	// CPU usage stats (no chart)
	cpuData := make([]float64, len(data))
	for i, point := range data {
		cpuData[i] = float64(point.CPUUsage)
	}

	if len(cpuData) > 0 {
		fmt.Printf("├─ CPU Usage: %.0f%% - %.0f%% (avg: %.1f%%)\n",
			min(cpuData), max(cpuData), avg(cpuData))
	}

	// Disk I/O stats (no chart)
	totalDiskRead, totalDiskWrite := float64(0), float64(0)
	for _, point := range data {
		totalDiskRead += float64(point.DiskReadBytes) / 1024
		totalDiskWrite += float64(point.DiskWriteBytes) / 1024
	}

	if len(data) > 0 {
		fmt.Printf("├─ Disk I/O: %s read + %s write\n",
			formatBytes(int64(totalDiskRead*1024)),
			formatBytes(int64(totalDiskWrite*1024)))
	}

	// Network traffic stats (no chart)
	totalNetIn, totalNetOut := float64(0), float64(0)
	for _, point := range data {
		totalNetIn += float64(point.NetworkInBytes) / 1024 / 1024
		totalNetOut += float64(point.NetworkOutBytes) / 1024 / 1024
	}

	if len(data) > 0 {
		fmt.Printf("└─ Network: %s in + %s out\n",
			formatBytes(int64(totalNetIn*1024*1024)),
			formatBytes(int64(totalNetOut*1024*1024)))
	}

	// Display total bandwidth information
	if serviceInfo != nil {
		displayCompactBandwidthInfo(serviceInfo)
	}

	fmt.Printf("\n")
}

// displayCompactBandwidthInfo displays monthly bandwidth information in compact format
func displayCompactBandwidthInfo(serviceInfo *client.ServiceInfo) {
	// Apply bandwidth multiplier for expensive locations
	actualMonthlyLimit := serviceInfo.PlanMonthlyData * int64(serviceInfo.MonthlyDataMultiplier)
	actualDataUsed := serviceInfo.DataCounter * int64(serviceInfo.MonthlyDataMultiplier)

	bandwidthLine := fmt.Sprintf("├─ Monthly Data: %s / %s",
		formatBytes(actualDataUsed), formatBytes(actualMonthlyLimit))

	if actualMonthlyLimit > 0 {
		usagePercent := float64(actualDataUsed) / float64(actualMonthlyLimit) * 100
		bandwidthLine += fmt.Sprintf(" (%.1f%%)", usagePercent)
	}

	if serviceInfo.MonthlyDataMultiplier > 1 {
		bandwidthLine += fmt.Sprintf(" [%dx]", serviceInfo.MonthlyDataMultiplier)
	}

	fmt.Printf("%s\n", bandwidthLine)
}

func displayUsageSummary(stats *client.UsageStatsResponse, instanceName string, dataPoints int, period string, serviceInfo *client.ServiceInfo) {
	if len(stats.Data) == 0 {
		return
	}

	// Sort data chronologically for summary
	data := make([]client.UsageDataPoint, len(stats.Data))
	copy(data, stats.Data)
	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp < data[j].Timestamp
	})

	// Calculate summary statistics
	cpuData := make([]float64, len(data))
	netInTotal, netOutTotal, diskReadTotal, diskWriteTotal := float64(0), float64(0), float64(0), float64(0)

	for i, point := range data {
		cpuData[i] = float64(point.CPUUsage)
		netInTotal += float64(point.NetworkInBytes)
		netOutTotal += float64(point.NetworkOutBytes)
		diskReadTotal += float64(point.DiskReadBytes)
		diskWriteTotal += float64(point.DiskWriteBytes)
	}

	// Calculate the actual time coverage: from previous data point to last data point
	startTime := time.Unix(data[0].Timestamp-300, 0) // Previous data point (5 min = 300 sec)
	endTime := time.Unix(data[len(data)-1].Timestamp, 0)
	timeSpan := endTime.Sub(startTime)

	fmt.Printf("\n📈 Usage Summary: %s (%s)\n", instanceName, stats.VMType)
	fmt.Printf("   Data Points      : %d total (%d displayed, period: %s)\n", len(stats.Data), dataPoints, period)
	fmt.Printf("   Time Span        : %s\n", formatDuration(timeSpan))
	fmt.Printf("\n")
	fmt.Printf("   CPU Usage        : %.1f%% avg | %.0f%% - %.0f%% range\n",
		avg(cpuData), min(cpuData), max(cpuData))
	fmt.Printf("   Disk Activity    : %s read, %s write (total)\n",
		formatBytes(int64(diskReadTotal)), formatBytes(int64(diskWriteTotal)))
	fmt.Printf("   Network Traffic  : %s in, %s out (total)\n",
		formatBytes(int64(netInTotal)), formatBytes(int64(netOutTotal)))

	// Display monthly bandwidth summary
	if serviceInfo != nil {
		displaySummaryBandwidthInfo(serviceInfo)
	}

	if timeSpan.Hours() > 0 {
		netInPerHour := netInTotal / timeSpan.Hours()
		netOutPerHour := netOutTotal / timeSpan.Hours()
		fmt.Printf("   Network Rate     : %s/h in, %s/h out (average)\n",
			formatBytes(int64(netInPerHour)), formatBytes(int64(netOutPerHour)))
	}
}

// displaySummaryBandwidthInfo displays monthly bandwidth information in summary format
func displaySummaryBandwidthInfo(serviceInfo *client.ServiceInfo) {
	// Apply bandwidth multiplier for expensive locations
	actualMonthlyLimit := serviceInfo.PlanMonthlyData * int64(serviceInfo.MonthlyDataMultiplier)
	actualDataUsed := serviceInfo.DataCounter * int64(serviceInfo.MonthlyDataMultiplier)

	fmt.Printf("   Monthly Bandwidth: %s / %s",
		formatBytes(actualDataUsed), formatBytes(actualMonthlyLimit))

	if actualMonthlyLimit > 0 {
		usagePercent := float64(actualDataUsed) / float64(actualMonthlyLimit) * 100
		remaining := actualMonthlyLimit - actualDataUsed
		fmt.Printf(" (%.1f%% used, %s remaining)", usagePercent, formatBytes(remaining))
	}

	if serviceInfo.MonthlyDataMultiplier > 1 {
		fmt.Printf(" [%dx multiplier]", serviceInfo.MonthlyDataMultiplier)
	}
	fmt.Printf("\n")

	if serviceInfo.DataNextReset > 0 {
		resetTime := time.Unix(serviceInfo.DataNextReset, 0).Local()
		fmt.Printf("   Next Reset       : %s\n", resetTime.Format("2006-01-02 15:04"))
	}
}

// Helper functions for data analysis
func min(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	minVal := data[0]
	for _, v := range data {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

func max(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	maxVal := data[0]
	for _, v := range data {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

func avg(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	return sum(data) / float64(len(data))
}

func sum(data []float64) float64 {
	total := float64(0)
	for _, v := range data {
		total += v
	}
	return total
}

// formatRate formats rate values with appropriate units (for network traffic)
func formatRate(value float64, unit string) string {
	switch unit {
	case "KB":
		if value >= 1024*1024 {
			return fmt.Sprintf("%.1f GB/h", value/(1024*1024))
		} else if value >= 1024 {
			return fmt.Sprintf("%.1f MB/h", value/1024)
		}
		return fmt.Sprintf("%.1f KB/h", value)
	case "MB":
		if value >= 1024 {
			return fmt.Sprintf("%.1f GB/h", value/1024)
		}
		return fmt.Sprintf("%.1f MB/h", value)
	default:
		return fmt.Sprintf("%.1f %s/h", value, unit)
	}
}

// formatDiskRate formats disk I/O rate values with appropriate units (per second)
func formatDiskRate(value float64, unit string) string {
	switch unit {
	case "KB":
		if value >= 1024*1024 {
			return fmt.Sprintf("%.1f GB/s", value/(1024*1024))
		} else if value >= 1024 {
			return fmt.Sprintf("%.1f MB/s", value/1024)
		}
		return fmt.Sprintf("%.1f KB/s", value)
	case "MB":
		if value >= 1024 {
			return fmt.Sprintf("%.1f GB/s", value/1024)
		}
		return fmt.Sprintf("%.1f MB/s", value)
	default:
		return fmt.Sprintf("%.1f %s/s", value, unit)
	}
}

// filterDataByPeriod filters data points based on the specified time period
func filterDataByPeriod(data []client.UsageDataPoint, period string) []client.UsageDataPoint {
	if len(data) == 0 {
		return data
	}

	now := time.Now()
	var cutoffTime time.Time

	switch period {
	case "1d":
		cutoffTime = now.Add(-24 * time.Hour)
	case "7d":
		cutoffTime = now.Add(-7 * 24 * time.Hour)
	case "1m":
		cutoffTime = now.Add(-30 * 24 * time.Hour)
	case "all":
		return data
	default:
		// Default to 1 day
		cutoffTime = now.Add(-24 * time.Hour)
	}

	var filtered []client.UsageDataPoint
	for _, point := range data {
		pointTime := time.Unix(point.Timestamp, 0)
		if pointTime.After(cutoffTime) {
			filtered = append(filtered, point)
		}
	}

	return filtered
}

// getTimeRange formats the time range for chart titles
func getTimeRange(data []client.UsageDataPoint) string {
	if len(data) < 2 {
		return ""
	}

	// Use previous data point as start time to show actual coverage
	startTime := time.Unix(data[0].Timestamp-300, 0).Local() // Previous data point (5 min = 300 sec)
	endTime := time.Unix(data[len(data)-1].Timestamp, 0).Local()

	// Format based on time span
	timeSpan := endTime.Sub(startTime)
	var format string
	if timeSpan.Hours() > 48 {
		format = "01-02 15:04"
	} else {
		format = "01-02 15:04"
	}

	return fmt.Sprintf("(%s - %s)", startTime.Format(format), endTime.Format(format))
}

// formatDuration formats duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Hour {
		return d.Truncate(time.Minute).String()
	}

	hours := int(d.Hours())
	if hours < 24 {
		minutes := int(d.Minutes()) % 60
		if minutes == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}

	days := hours / 24
	remainingHours := hours % 24

	if days < 7 {
		if remainingHours == 0 {
			return fmt.Sprintf("%d days", days)
		}
		return fmt.Sprintf("%d days %dh", days, remainingHours)
	}

	weeks := days / 7
	remainingDays := days % 7

	if weeks < 4 {
		if remainingDays == 0 {
			return fmt.Sprintf("%d weeks", weeks)
		}
		return fmt.Sprintf("%d weeks %d days", weeks, remainingDays)
	}

	months := weeks / 4 // Approximate months
	remainingWeeks := weeks % 4

	if remainingWeeks == 0 {
		return fmt.Sprintf("%d months", months)
	}
	return fmt.Sprintf("%d months %d weeks", months, remainingWeeks)
}
