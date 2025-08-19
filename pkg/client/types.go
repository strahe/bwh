package client

import (
	"encoding/json"
	"strconv"
	"strings"
)

// BaseResponse represents the base response structure for API calls
type BaseResponse struct {
	Error                 int                    `json:"error"`
	Message               string                 `json:"message,omitempty"`
	AdditionalErrorInfo   string                 `json:"additionalErrorInfo,omitempty"`
	AdditionalLockingInfo *AdditionalLockingInfo `json:"additionalLockingInfo,omitempty"`
}

// ServiceInfo represents the complete BWH VPS service information
type ServiceInfo struct {
	BaseResponse

	// Basic VPS Information
	VMType   string `json:"vm_type"`  // Hypervisor type (ovz or kvm)
	Hostname string `json:"hostname"` // Hostname of the VPS
	Plan     string `json:"plan"`     // Name of plan
	OS       string `json:"os"`       // Operating system
	Email    string `json:"email"`    // Primary e-mail address of the account

	// Node and Location Information
	NodeAlias         string `json:"node_alias"`          // Internal nickname of the physical node
	NodeLocationID    string `json:"node_location_id"`    // Location identifier
	NodeLocation      string `json:"node_location"`       // Physical location (country, state)
	NodeDatacenter    string `json:"node_datacenter"`     // Datacenter details
	LocationIPv6Ready bool   `json:"location_ipv6_ready"` // Whether IPv6 is supported at the current location

	// Resource Allocation (all in bytes)
	PlanDisk int64 `json:"plan_disk"` // Disk quota (bytes)
	PlanRAM  int64 `json:"plan_ram"`  // RAM (bytes)
	PlanSwap int64 `json:"plan_swap"` // SWAP (bytes)

	// Data Transfer Information
	PlanMonthlyData       int64 `json:"plan_monthly_data"`       // Allowed monthly data transfer (bytes). Multiply by monthly_data_multiplier
	DataCounter           int64 `json:"data_counter"`            // Data transfer used in current billing month. Multiply by monthly_data_multiplier
	MonthlyDataMultiplier int   `json:"monthly_data_multiplier"` // Bandwidth accounting coefficient for expensive locations
	DataNextReset         int64 `json:"data_next_reset"`         // Date and time of transfer counter reset (UNIX timestamp)

	// Network Configuration
	IPAddresses           []string `json:"ip_addresses"`             // IPv4 addresses and IPv6 /64 subnets assigned to VPS
	IPv6SitTunnelEndpoint string   `json:"ipv6_sit_tunnel_endpoint"` // IPv6 SIT tunnel endpoint
	PrivateIPAddresses    []string `json:"private_ip_addresses"`     // Private IPv4 addresses assigned to VPS
	IPNullroutes          []string `json:"ip_nullroutes"`            // Information on IP address nullrouting during (D)DoS attacks
	PlanMaxIPv6s          int      `json:"plan_max_ipv6s"`           // Maximum number of IPv6 /64 subnets allowed by plan

	// ISO Images
	ISO1          string   `json:"iso1"`           // Mounted image #1
	ISO2          string   `json:"iso2"`           // Mounted image #2 (currently unsupported)
	AvailableISOs []string `json:"available_isos"` // Array of ISO images available for use

	// Network Features
	PlanPrivateNetworkAvailable     bool              `json:"plan_private_network_available"`     // Whether Private Network features are available on this plan
	LocationPrivateNetworkAvailable bool              `json:"location_private_network_available"` // Whether Private Network features are available at this location
	RDNSAPIAvailable                bool              `json:"rdns_api_available"`                 // Whether rDNS records can be set via API
	PTR                             map[string]string `json:"ptr"`                                // rDNS records (ip=>value)
	FreeIPReplacementInterval       int               `json:"free_ip_replacement_interval"`       // Free IP replacement interval (deprecated, returns -100)

	// Security and Status
	Suspended        bool `json:"suspended"`          // Whether VPS is suspended
	PolicyViolation  bool `json:"policy_violation"`   // Whether there is an active policy violation that needs attention
	SuspensionCount  int  `json:"suspension_count"`   // Number of times service was suspended in current calendar year
	TotalAbusePoints int  `json:"total_abuse_points"` // Total abuse points accumulated in current calendar year
	MaxAbusePoints   int  `json:"max_abuse_points"`   // Maximum abuse points allowed by plan in a calendar year
}

// CreateSnapshotResponse represents the response from creating a snapshot
type CreateSnapshotResponse struct {
	BaseResponse
	NotificationEmail string `json:"notificationEmail"`
}

// FlexibleInt is a type that can unmarshal both string and int from JSON
type FlexibleInt struct {
	Value int64
}

func (f *FlexibleInt) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as int first
	var intVal int64
	if err := json.Unmarshal(data, &intVal); err == nil {
		f.Value = intVal
		return nil
	}

	// Try as string
	var strVal string
	if err := json.Unmarshal(data, &strVal); err == nil {
		if parsed, err := strconv.ParseInt(strVal, 10, 64); err == nil {
			f.Value = parsed
			return nil
		}
	}

	f.Value = 0
	return nil
}

// LiveServiceInfo represents the complete live VPS information including ServiceInfo and real-time status
type LiveServiceInfo struct {
	ServiceInfo

	// Common fields for both OVZ and KVM
	IsCPUThrottled FlexibleInt `json:"is_cpu_throttled"` // 0 = not throttled, 1 = throttled (resets every 2 hours)
	SSHPort        int         `json:"ssh_port"`         // SSH port of the VPS

	// OpenVZ specific fields
	VzStatus map[string]any `json:"vz_status,omitempty"` // OpenVZ beancounters, load average, processes, etc.
	VzQuota  map[string]any `json:"vz_quota,omitempty"`  // OpenVZ disk size, inodes and usage info

	// KVM specific fields
	VeStatus            string      `json:"ve_status,omitempty"`             // Starting, Running or Stopped
	VeMac1              string      `json:"ve_mac1,omitempty"`               // MAC address of primary network interface
	VeUsedDiskSpaceB    FlexibleInt `json:"ve_used_disk_space_b,omitzero"`   // Occupied disk space in bytes
	VeDiskQuotaGB       FlexibleInt `json:"ve_disk_quota_gb,omitzero"`       // Actual size of disk image in GB
	IsDiskThrottled     FlexibleInt `json:"is_disk_throttled,omitzero"`      // 0 = not throttled, 1 = throttled (resets 15-180 min)
	LiveHostname        string      `json:"live_hostname,omitempty"`         // Result of "hostname" command inside VPS
	LoadAverage         string      `json:"load_average,omitempty"`          // Raw load average string
	MemAvailableKB      FlexibleInt `json:"mem_available_kb,omitzero"`       // Available RAM in KB
	SwapTotalKB         FlexibleInt `json:"swap_total_kb,omitzero"`          // Total Swap in KB
	SwapAvailableKB     FlexibleInt `json:"swap_available_kb,omitzero"`      // Available Swap in KB
	ScreendumpPngBase64 string      `json:"screendump_png_base64,omitempty"` // base64 encoded PNG screenshot of VGA console
}

// AvailableOSResponse represents the response from getAvailableOS API call
type AvailableOSResponse struct {
	BaseResponse
	Installed string   `json:"installed"` // Currently installed Operating System
	Templates []string `json:"templates"` // Array of available OS templates
}

// UsageDataPoint represents a single data point in usage statistics
type UsageDataPoint struct {
	Timestamp       int64 `json:"timestamp"`         // Unix timestamp
	CPUUsage        int   `json:"cpu_usage"`         // CPU usage percentage
	NetworkInBytes  int64 `json:"network_in_bytes"`  // Network incoming bytes
	NetworkOutBytes int64 `json:"network_out_bytes"` // Network outgoing bytes
	DiskReadBytes   int64 `json:"disk_read_bytes"`   // Disk read bytes
	DiskWriteBytes  int64 `json:"disk_write_bytes"`  // Disk write bytes
}

// UsageStatsResponse represents the response from getRawUsageStats API call
type UsageStatsResponse struct {
	BaseResponse
	Data   []UsageDataPoint `json:"data"`    // Array of usage data points
	VMType string           `json:"vm_type"` // VM type (kvm/ovz)
}

// AuditLogEntry represents a single audit log entry
type AuditLogEntry struct {
	Timestamp     int64  `json:"timestamp"`      // Unix timestamp of the event
	RequestorIPv4 uint32 `json:"requestor_ipv4"` // IPv4 address of the requestor (as integer)
	Type          int    `json:"type"`           // Event type code
	Summary       string `json:"summary"`        // Human-readable summary of the event
}

// AuditLogResponse represents the response from getAuditLog API call
type AuditLogResponse struct {
	BaseResponse
	LogEntries []AuditLogEntry `json:"log_entries"` // Array of audit log entries
}

// ResetRootPasswordResponse represents the response from resetRootPassword API call
type ResetRootPasswordResponse struct {
	BaseResponse
	Password string `json:"password"` // The new root password
}

// SnapshotInfo represents a single snapshot
type SnapshotInfo struct {
	FileName        string      `json:"fileName"`        // File name of the snapshot
	OS              string      `json:"os"`              // Operating system of the snapshot
	Description     string      `json:"description"`     // Description of the snapshot
	Size            FlexibleInt `json:"size"`            // Size of the snapshot in bytes (compressed)
	MD5             string      `json:"md5"`             // MD5 hash of the snapshot
	Sticky          bool        `json:"sticky"`          // Whether snapshot is sticky (never purged)
	Uncompressed    FlexibleInt `json:"uncompressed"`    // Uncompressed size in bytes
	PurgesIn        FlexibleInt `json:"purgesIn"`        // Seconds until snapshot is purged
	DownloadLink    string      `json:"downloadLink"`    // HTTP download link
	DownloadLinkSSL string      `json:"downloadLinkSSL"` // HTTPS download link
}

// SnapshotListResponse represents the response from snapshot/list API call
type SnapshotListResponse struct {
	BaseResponse
	Snapshots []SnapshotInfo `json:"snapshots"` // Array of snapshots
}

// SnapshotExportResponse represents the response from snapshot/export API call
type SnapshotExportResponse struct {
	BaseResponse
	Token string `json:"token"` // Token for import
}

// BackupInfo represents a single backup entry
type BackupInfo struct {
	Token     string `json:"-"`         // Backup token (from map key)
	Size      int64  `json:"size"`      // Backup size in bytes
	OS        string `json:"os"`        // Operating system
	MD5       string `json:"md5"`       // MD5 hash
	Timestamp int64  `json:"timestamp"` // Unix timestamp
}

// BackupListResponse represents the response from backup/list API call
type BackupListResponse struct {
	BaseResponse
	Backups map[string]BackupInfo `json:"backups"` // Map of backup token to backup info
}

// RateLimitStatus represents the response from getRateLimitStatus API call
type RateLimitStatus struct {
	BaseResponse
	RemainingPoints15Min int `json:"remaining_points_15min"` // API calls remaining in 15-minute window
	RemainingPoints24H   int `json:"remaining_points_24h"`   // API calls remaining in 24-hour window
}

// SshKeysResponse represents the response from getSshKeys API call
type SshKeysResponse struct {
	BaseResponse
	SshKeysVeid               string `json:"ssh_keys_veid"`                // Per-VM keys in Hypervisor Vault (newline-separated)
	SshKeysUser               string `json:"ssh_keys_user"`                // Per-Account keys in Billing Portal (newline-separated)
	SshKeysPreferred          string `json:"ssh_keys_preferred"`           // Keys that will be used during reinstallOS (newline-separated)
	ShortenedSshKeysVeid      string `json:"shortened_ssh_keys_veid"`      // Visually shortened VM keys
	ShortenedSshKeysUser      string `json:"shortened_ssh_keys_user"`      // Visually shortened user keys
	ShortenedSshKeysPreferred string `json:"shortened_ssh_keys_preferred"` // Visually shortened preferred keys
}

// GetSshKeysVeidSlice returns VM SSH keys as a slice
func (r *SshKeysResponse) GetSshKeysVeidSlice() []string {
	return splitSshKeys(r.SshKeysVeid)
}

// GetSshKeysUserSlice returns user SSH keys as a slice
func (r *SshKeysResponse) GetSshKeysUserSlice() []string {
	return splitSshKeys(r.SshKeysUser)
}

// GetSshKeysPreferredSlice returns preferred SSH keys as a slice
func (r *SshKeysResponse) GetSshKeysPreferredSlice() []string {
	return splitSshKeys(r.SshKeysPreferred)
}

// GetShortenedSshKeysVeidSlice returns shortened VM SSH keys as a slice
func (r *SshKeysResponse) GetShortenedSshKeysVeidSlice() []string {
	return splitSshKeys(r.ShortenedSshKeysVeid)
}

// GetShortenedSshKeysUserSlice returns shortened user SSH keys as a slice
func (r *SshKeysResponse) GetShortenedSshKeysUserSlice() []string {
	return splitSshKeys(r.ShortenedSshKeysUser)
}

// GetShortenedSshKeysPreferredSlice returns shortened preferred SSH keys as a slice
func (r *SshKeysResponse) GetShortenedSshKeysPreferredSlice() []string {
	return splitSshKeys(r.ShortenedSshKeysPreferred)
}

// splitSshKeys splits a newline-separated SSH keys string into a slice
func splitSshKeys(keys string) []string {
	if keys == "" {
		return []string{}
	}

	lines := strings.Split(strings.TrimSpace(keys), "\n")
	var result []string
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// IPv6AddResponse represents the response from ipv6/add API call
type IPv6AddResponse struct {
	BaseResponse
	AssignedSubnet string `json:"assigned_subnet"` // Newly assigned IPv6 /64 subnet
}
