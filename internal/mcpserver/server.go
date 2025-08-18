package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/strahe/bwh/internal/config"
	"github.com/strahe/bwh/pkg/client"
)

// RunMCPStdioServer starts a minimal stdio-based MCP server exposing read-only tools.
// This is a placeholder wiring that we will flesh out in subsequent edits.
func RunMCPStdioServer(ctx context.Context, configPath, instanceName string) error {
	// Load config and resolve instance so we can sanity check connectivity on startup
	manager, err := config.NewManager(configPath)
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}

	// Resolve once here only for connectivity check (uses provided instanceName or config default)
	instForCheck, _, err := manager.ResolveInstance(instanceName)
	if err != nil {
		return fmt.Errorf("failed to resolve instance: %w", err)
	}

	// Prepare a client to verify we can at least talk to API when server starts
	bwhClient := client.NewClient(instForCheck.APIKey, instForCheck.VeID)
	if instForCheck.Endpoint != "" {
		bwhClient.SetBaseURL(instForCheck.Endpoint)
	}

	// Lightweight connectivity check (rate limit endpoint is cheap)
	if _, err := bwhClient.GetRateLimitStatus(ctx); err != nil {
		return fmt.Errorf("failed API connectivity: %w", err)
	}

	// Construct MCP server (stdio)
	s := server.NewMCPServer(
		"BWH / BandwagonHost (搬瓦工) MCP",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithRecovery(),
	)

	// Register read-only tools
	registerReadOnlyTools(s, manager)

	// Register simple resources
	registerResources(s, manager)

	// Run over stdio and block
	return server.ServeStdio(s)
}

// registerReadOnlyTools wires read-only tool handlers backed by pkg/client
func registerReadOnlyTools(s *server.MCPServer, manager *config.Manager) {
	// vps_info_get
	s.AddTool(
		mcp.NewTool(
			"vps_info_get",
			mcp.WithDescription("Get live VPS information for BWH/BandwagonHost/搬瓦工/瓦工"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithString("instance", mcp.Description("Target instance name; defaults to config default")),
			mcp.WithBoolean("compact", mcp.DefaultBool(false), mcp.Description("Return concise summary instead of full payload")),
			mcp.WithBoolean("live", mcp.DefaultBool(true), mcp.Description("Use live info (true) or cached service info (false)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Resolve per-call from args or config default
			requested := req.GetString("instance", "")
			compact := req.GetBool("compact", false)
			live := req.GetBool("live", true)
			inst, resolved, err := manager.ResolveInstance(requested)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("resolve instance failed: %v", err)), nil
			}
			c := client.NewClient(inst.APIKey, inst.VeID)
			if inst.Endpoint != "" {
				c.SetBaseURL(inst.Endpoint)
			}
			if live {
				info, err := c.GetLiveServiceInfo(ctx)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("get live info failed: %v", err)), nil
				}
				if compact {
					summary := map[string]any{
						"hostname": info.Hostname,
						"vm_type":  info.VMType,
						"status":   info.VeStatus,
						"plan":     info.Plan,
						"os":       info.OS,
						"location": info.NodeLocation,
						"ips":      len(info.IPAddresses),
					}
					return mcp.NewToolResultStructuredOnly(map[string]any{
						"instance": resolved,
						"summary":  summary,
					}), nil
				}
				return mcp.NewToolResultStructuredOnly(map[string]any{
					"instance": resolved,
					"data":     info,
				}), nil
			}
			serviceInfo, err := c.GetServiceInfo(ctx)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("get service info failed: %v", err)), nil
			}
			if compact {
				summary := map[string]any{
					"hostname": serviceInfo.Hostname,
					"vm_type":  serviceInfo.VMType,
					"plan":     serviceInfo.Plan,
					"os":       serviceInfo.OS,
					"location": serviceInfo.NodeLocation,
					"ips":      len(serviceInfo.IPAddresses),
				}
				return mcp.NewToolResultStructuredOnly(map[string]any{
					"instance": resolved,
					"summary":  summary,
				}), nil
			}
			return mcp.NewToolResultStructuredOnly(map[string]any{
				"instance": resolved,
				"data":     serviceInfo,
			}), nil
		},
	)

	// vps_usage_get
	s.AddTool(
		mcp.NewTool(
			"vps_usage_get",
			mcp.WithDescription("Get usage summary for BWH/BandwagonHost/搬瓦工/瓦工 (supports period/days/group_by)"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithString("instance", mcp.Description("Target instance name; defaults to config default")),
			mcp.WithString("period", mcp.Description("Lookback window, e.g. 1d, 7d, 30d")),
			mcp.WithNumber("days", mcp.Description("Lookback days if period not provided")),
			mcp.WithString("group_by", mcp.Enum("5m", "hour", "day"), mcp.Description("Aggregation bucket: 5m|hour|day (default: day)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			requested := req.GetString("instance", "")
			periodStr := req.GetString("period", "")
			daysArg := req.GetInt("days", 0)
			groupBy := req.GetString("group_by", "day")

			inst, resolved, err := manager.ResolveInstance(requested)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("resolve instance failed: %v", err)), nil
			}
			c := client.NewClient(inst.APIKey, inst.VeID)
			if inst.Endpoint != "" {
				c.SetBaseURL(inst.Endpoint)
			}
			stats, err := c.GetRawUsageStats(ctx)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("get usage failed: %v", err)), nil
			}

			days := 0
			if daysArg > 0 {
				days = daysArg
			} else if len(periodStr) > 1 && periodStr[len(periodStr)-1] == 'd' {
				var n int
				for i := 0; i < len(periodStr)-1; i++ {
					ch := periodStr[i]
					if ch < '0' || ch > '9' {
						n = 0
						break
					}
					n = n*10 + int(ch-'0')
				}
				days = n
			}
			if days <= 0 {
				days = 1
			}

			now := time.Now().UTC()
			cutoff := now.Add(-time.Duration(days) * 24 * time.Hour).Unix()

			var bucketDur time.Duration
			switch groupBy {
			case "5m":
				bucketDur = 5 * time.Minute
			case "hour":
				bucketDur = time.Hour
			default:
				groupBy = "day"
				bucketDur = 24 * time.Hour
			}

			type agg struct {
				count          int
				cpuSum         float64
				cpuMin         float64
				cpuMax         float64
				netInTotal     int64
				netOutTotal    int64
				diskReadTotal  int64
				diskWriteTotal int64
			}

			buckets := map[int64]*agg{}
			var global agg
			global.cpuMin = 101
			global.cpuMax = -1
			var firstTs int64 = 0
			var lastTs int64 = 0

			for _, p := range stats.Data {
				if p.Timestamp < cutoff {
					continue
				}
				cpu := float64(p.CPUUsage)
				netIn := p.NetworkInBytes
				netOut := p.NetworkOutBytes
				read := p.DiskReadBytes
				write := p.DiskWriteBytes

				if firstTs == 0 || p.Timestamp < firstTs {
					firstTs = p.Timestamp
				}
				if p.Timestamp > lastTs {
					lastTs = p.Timestamp
				}

				bucketStart := time.Unix(p.Timestamp, 0).UTC().Truncate(bucketDur).Unix()
				a, ok := buckets[bucketStart]
				if !ok {
					a = &agg{cpuMin: 101, cpuMax: -1}
					buckets[bucketStart] = a
				}
				a.count++
				a.cpuSum += cpu
				if cpu < a.cpuMin {
					a.cpuMin = cpu
				}
				if cpu > a.cpuMax {
					a.cpuMax = cpu
				}
				a.netInTotal += netIn
				a.netOutTotal += netOut
				a.diskReadTotal += read
				a.diskWriteTotal += write

				global.count++
				global.cpuSum += cpu
				if cpu < global.cpuMin {
					global.cpuMin = cpu
				}
				if cpu > global.cpuMax {
					global.cpuMax = cpu
				}
				global.netInTotal += netIn
				global.netOutTotal += netOut
				global.diskReadTotal += read
				global.diskWriteTotal += write
			}

			if global.count == 0 {
				return mcp.NewToolResultStructuredOnly(map[string]any{
					"instance": resolved,
					"vm_type":  stats.VMType,
					"range":    map[string]any{"days": days, "group_by": groupBy},
					"buckets":  []any{},
				}), nil
			}

			type bucketOut struct {
				StartRFC3339   string  `json:"start_rfc3339"`
				Points         int     `json:"points"`
				CPUAvg         float64 `json:"cpu_avg"`
				CPUMin         float64 `json:"cpu_min"`
				CPUMax         float64 `json:"cpu_max"`
				NetInTotal     int64   `json:"net_in_total_bytes"`
				NetOutTotal    int64   `json:"net_out_total_bytes"`
				DiskReadTotal  int64   `json:"disk_read_total_bytes"`
				DiskWriteTotal int64   `json:"disk_write_total_bytes"`
			}

			// sort keys
			var keys []int64
			for k := range buckets {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

			var outBuckets []bucketOut
			for _, k := range keys {
				a := buckets[k]
				cpuAvg := 0.0
				if a.count > 0 {
					cpuAvg = a.cpuSum / float64(a.count)
				}
				outBuckets = append(outBuckets, bucketOut{
					StartRFC3339:   time.Unix(k, 0).UTC().Format(time.RFC3339),
					Points:         a.count,
					CPUAvg:         cpuAvg,
					CPUMin:         a.cpuMin,
					CPUMax:         a.cpuMax,
					NetInTotal:     a.netInTotal,
					NetOutTotal:    a.netOutTotal,
					DiskReadTotal:  a.diskReadTotal,
					DiskWriteTotal: a.diskWriteTotal,
				})
			}

			globalCPUAvg := global.cpuSum / float64(global.count)
			durSec := lastTs - firstTs
			if durSec < 0 {
				durSec = 0
			}
			durSec += 300
			summary := map[string]any{
				"vm_type":      stats.VMType,
				"points":       global.count,
				"time_start":   time.Unix(firstTs, 0).UTC().Format(time.RFC3339),
				"time_end":     time.Unix(lastTs, 0).UTC().Format(time.RFC3339),
				"duration_sec": durSec,
				"cpu": map[string]any{
					"avg": globalCPUAvg,
					"min": global.cpuMin,
					"max": global.cpuMax,
				},
				"network_bytes": map[string]any{
					"in_total":  global.netInTotal,
					"out_total": global.netOutTotal,
				},
				"disk_bytes": map[string]any{
					"read_total":  global.diskReadTotal,
					"write_total": global.diskWriteTotal,
				},
			}

			return mcp.NewToolResultStructuredOnly(map[string]any{
				"instance": resolved,
				"range":    map[string]any{"days": days, "group_by": groupBy},
				"summary":  summary,
				"buckets":  outBuckets,
			}), nil
		},
	)

	// snapshot_list
	s.AddTool(
		mcp.NewTool(
			"snapshot_list",
			mcp.WithDescription("List snapshots for BWH/BandwagonHost/搬瓦工/瓦工"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithString("instance", mcp.Description("Target instance name; defaults to config default")),
			mcp.WithBoolean("sticky_only", mcp.DefaultBool(false), mcp.Description("Filter to sticky snapshots only")),
			mcp.WithString("name_contains", mcp.Description("Filter by substring in fileName/description")),
			mcp.WithString("sort_by", mcp.Enum("name", "size", "sticky"), mcp.Description("Sort key: name|size|sticky (default: name)")),
			mcp.WithString("order", mcp.Enum("asc", "desc"), mcp.Description("Sort order asc|desc (default: asc)")),
			mcp.WithNumber("limit", mcp.Description("Maximum items to return")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			requested := req.GetString("instance", "")
			stickyOnly := req.GetBool("sticky_only", false)
			nameContains := strings.TrimSpace(req.GetString("name_contains", ""))
			sortBy := req.GetString("sort_by", "name")
			order := req.GetString("order", "asc")
			limit := req.GetInt("limit", 0)

			inst, resolved, err := manager.ResolveInstance(requested)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("resolve instance failed: %v", err)), nil
			}
			c := client.NewClient(inst.APIKey, inst.VeID)
			if inst.Endpoint != "" {
				c.SetBaseURL(inst.Endpoint)
			}
			list, err := c.ListSnapshots(ctx)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("list snapshots failed: %v", err)), nil
			}

			snaps := make([]client.SnapshotInfo, 0, len(list.Snapshots))
			for _, s := range list.Snapshots {
				if stickyOnly && !s.Sticky {
					continue
				}
				if nameContains != "" {
					desc := s.Description
					if strings.Contains(strings.ToLower(s.FileName), strings.ToLower(nameContains)) || (desc != "" && strings.Contains(strings.ToLower(desc), strings.ToLower(nameContains))) {
						snaps = append(snaps, s)
					} else {
						continue
					}
				} else {
					snaps = append(snaps, s)
				}
			}

			sort.Slice(snaps, func(i, j int) bool {
				switch sortBy {
				case "size":
					if order == "desc" {
						return snaps[i].Size.Value > snaps[j].Size.Value
					}
					return snaps[i].Size.Value < snaps[j].Size.Value
				case "sticky":
					if order == "desc" {
						return snaps[i].Sticky && !snaps[j].Sticky
					}
					return (!snaps[i].Sticky && snaps[j].Sticky)
				default: // name
					if order == "desc" {
						return snaps[i].FileName > snaps[j].FileName
					}
					return snaps[i].FileName < snaps[j].FileName
				}
			})
			if limit > 0 && limit < len(snaps) {
				snaps = snaps[:limit]
			}

			return mcp.NewToolResultStructuredOnly(map[string]any{
				"instance": resolved,
				"items":    snaps,
			}), nil
		},
	)

	// backup_list
	s.AddTool(
		mcp.NewTool(
			"backup_list",
			mcp.WithDescription("List backups for BWH/BandwagonHost/搬瓦工/瓦工"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithString("instance", mcp.Description("Target instance name; defaults to config default")),
			mcp.WithString("os_contains", mcp.Description("Filter backups by OS substring")),
			mcp.WithString("since", mcp.Description("RFC3339 timestamp inclusive start filter")),
			mcp.WithString("until", mcp.Description("RFC3339 timestamp inclusive end filter")),
			mcp.WithString("sort_by", mcp.Enum("time", "size"), mcp.Description("Sort key: time|size (default: time)")),
			mcp.WithString("order", mcp.Enum("asc", "desc"), mcp.Description("Sort order asc|desc (default: desc by time)")),
			mcp.WithNumber("limit", mcp.Description("Maximum items to return")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			requested := req.GetString("instance", "")
			osContains := strings.ToLower(strings.TrimSpace(req.GetString("os_contains", "")))
			sinceStr := strings.TrimSpace(req.GetString("since", ""))
			untilStr := strings.TrimSpace(req.GetString("until", ""))
			sortBy := req.GetString("sort_by", "time")
			order := req.GetString("order", "desc")
			limit := req.GetInt("limit", 0)

			inst, resolved, err := manager.ResolveInstance(requested)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("resolve instance failed: %v", err)), nil
			}
			c := client.NewClient(inst.APIKey, inst.VeID)
			if inst.Endpoint != "" {
				c.SetBaseURL(inst.Endpoint)
			}
			resp, err := c.ListBackups(ctx)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("list backups failed: %v", err)), nil
			}

			var sinceTs, untilTs int64
			if sinceStr != "" {
				if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
					sinceTs = t.Unix()
				}
			}
			if untilStr != "" {
				if t, err := time.Parse(time.RFC3339, untilStr); err == nil {
					untilTs = t.Unix()
				}
			}

			backups := make([]client.BackupInfo, 0, len(resp.Backups))
			for token, b := range resp.Backups {
				b.Token = token
				if osContains != "" && !strings.Contains(strings.ToLower(b.OS), osContains) {
					continue
				}
				if sinceTs > 0 && b.Timestamp < sinceTs {
					continue
				}
				if untilTs > 0 && b.Timestamp > untilTs {
					continue
				}
				backups = append(backups, b)
			}

			sort.Slice(backups, func(i, j int) bool {
				switch sortBy {
				case "size":
					if order == "desc" {
						return backups[i].Size > backups[j].Size
					}
					return backups[i].Size < backups[j].Size
				default: // time
					if order == "asc" {
						return backups[i].Timestamp < backups[j].Timestamp
					}
					return backups[i].Timestamp > backups[j].Timestamp
				}
			})
			if limit > 0 && limit < len(backups) {
				backups = backups[:limit]
			}

			return mcp.NewToolResultStructuredOnly(map[string]any{
				"instance": resolved,
				"items":    backups,
			}), nil
		},
	)

	// vps_audit_get
	s.AddTool(
		mcp.NewTool(
			"vps_audit_get",
			mcp.WithDescription("Get audit log entries for BWH/BandwagonHost/搬瓦工/瓦工"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			mcp.WithString("instance", mcp.Description("Target instance name; defaults to config default")),
			mcp.WithString("since", mcp.Description("RFC3339 timestamp inclusive start filter")),
			mcp.WithString("until", mcp.Description("RFC3339 timestamp inclusive end filter")),
			mcp.WithNumber("limit", mcp.Description("Maximum items to return (newest first)")),
			mcp.WithString("ip_contains", mcp.Description("Filter by requestor IPv4 string contains")),
			mcp.WithNumber("type", mcp.Description("Filter by event type integer")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			requested := req.GetString("instance", "")
			sinceStr := strings.TrimSpace(req.GetString("since", ""))
			untilStr := strings.TrimSpace(req.GetString("until", ""))
			limit := req.GetInt("limit", 0)
			ipContains := strings.TrimSpace(req.GetString("ip_contains", ""))
			typeFilter := req.GetInt("type", -1)

			inst, resolved, err := manager.ResolveInstance(requested)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("resolve instance failed: %v", err)), nil
			}
			c := client.NewClient(inst.APIKey, inst.VeID)
			if inst.Endpoint != "" {
				c.SetBaseURL(inst.Endpoint)
			}
			logResp, err := c.GetAuditLog(ctx)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("get audit log failed: %v", err)), nil
			}

			var sinceTs, untilTs int64
			if sinceStr != "" {
				if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
					sinceTs = t.Unix()
				}
			}
			if untilStr != "" {
				if t, err := time.Parse(time.RFC3339, untilStr); err == nil {
					untilTs = t.Unix()
				}
			}

			entries := logResp.LogEntries
			// newest first
			sort.Slice(entries, func(i, j int) bool { return entries[i].Timestamp > entries[j].Timestamp })

			filtered := make([]client.AuditLogEntry, 0, len(entries))
			for _, e := range entries {
				if sinceTs > 0 && e.Timestamp < sinceTs {
					continue
				}
				if untilTs > 0 && e.Timestamp > untilTs {
					continue
				}
				if typeFilter >= 0 && e.Type != typeFilter {
					continue
				}
				if ipContains != "" {
					ip := fmt.Sprintf("%d.%d.%d.%d", byte(e.RequestorIPv4>>24), byte(e.RequestorIPv4>>16), byte(e.RequestorIPv4>>8), byte(e.RequestorIPv4))
					if !strings.Contains(ip, ipContains) {
						continue
					}
				}
				filtered = append(filtered, e)
				if limit > 0 && len(filtered) >= limit {
					break
				}
			}

			return mcp.NewToolResultStructuredOnly(map[string]any{
				"instance": resolved,
				"items":    filtered,
			}), nil
		},
	)
}

// registerResources exposes a minimal set of resources to browse last-fetched data or config view
func registerResources(s *server.MCPServer, manager *config.Manager) {
	// Session/config view resource
	s.AddResource(
		mcp.NewResource(
			"bwh://session/default",
			"Session Config",
			mcp.WithResourceDescription("Default instance and available nodes (BWH/BandwagonHost/搬瓦工/瓦工) for this session"),
			mcp.WithMIMEType("application/json"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			// Build a safe view of instances without exposing API keys
			instances := manager.ListInstances()
			masked := map[string]map[string]any{}
			for name, inst := range instances {
				masked[name] = map[string]any{
					"veid":        inst.VeID,
					"endpoint":    inst.Endpoint,
					"description": inst.Description,
					"tags":        inst.Tags,
				}
			}
			payload := map[string]any{
				"default_instance": manager.GetDefaultInstance(),
				"instances":        masked,
			}
			b, err := json.Marshal(payload)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal resource: %w", err)
			}
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      "bwh://session/default",
					MIMEType: "application/json",
					Text:     string(b),
				},
			}, nil
		},
	)
}
