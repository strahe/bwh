package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/strahe/bwh/internal/version"
)

const (
	defaultBaseURL = "https://api.64clouds.com/v1"
)

// Client represents a BandwagonHost API client
type Client struct {
	apiKey     string
	veid       string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new BandwagonHost client
func NewClient(apiKey, veid string) *Client {
	return &Client{
		apiKey:  apiKey,
		veid:    veid,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetBaseURL sets a custom base URL for the API client
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// GetServiceInfo gets information about the server
func (c *Client) GetServiceInfo(ctx context.Context) (*ServiceInfo, error) {
	var serviceInfo ServiceInfo
	if err := c.doRequest(ctx, "getServiceInfo", nil, &serviceInfo); err != nil {
		return nil, err
	}

	return wrapErrorWithBase(&serviceInfo, serviceInfo.BaseResponse)
}

// GetLiveServiceInfo gets real-time information about the server including detailed VPS status
// This call may take up to 15 seconds to complete as it queries the actual VPS status
func (c *Client) GetLiveServiceInfo(ctx context.Context) (*LiveServiceInfo, error) {
	var liveServiceInfo LiveServiceInfo
	if err := c.doRequest(ctx, "getLiveServiceInfo", nil, &liveServiceInfo); err != nil {
		return nil, err
	}

	return wrapErrorWithBase(&liveServiceInfo, liveServiceInfo.BaseResponse)
}

// CreateSnapshot creates a snapshot with optional description
func (c *Client) CreateSnapshot(ctx context.Context, description string) (*CreateSnapshotResponse, error) {
	params := map[string]string{}
	if description != "" {
		params["description"] = description
	}

	var resp CreateSnapshotResponse
	if err := c.doRequest(ctx, "snapshot/create", params, &resp); err != nil {
		return nil, err
	}

	return wrapErrorWithBase(&resp, resp.BaseResponse)
}

// ListSnapshots gets the list of snapshots
func (c *Client) ListSnapshots(ctx context.Context) (*SnapshotListResponse, error) {
	var resp SnapshotListResponse
	if err := c.doRequest(ctx, "snapshot/list", nil, &resp); err != nil {
		return nil, err
	}

	return wrapErrorWithBase(&resp, resp.BaseResponse)
}

// DeleteSnapshot deletes a snapshot by fileName
func (c *Client) DeleteSnapshot(ctx context.Context, fileName string) error {
	var resp BaseResponse
	if err := c.doRequest(ctx, "snapshot/delete", map[string]string{"snapshot": fileName}, &resp); err != nil {
		return err
	}

	return wrapOnlyErrorFromBase(resp)
}

// RestoreSnapshot restores a snapshot by fileName (overwrites all data on VPS)
func (c *Client) RestoreSnapshot(ctx context.Context, fileName string) error {
	var resp BaseResponse
	if err := c.doRequest(ctx, "snapshot/restore", map[string]string{"snapshot": fileName}, &resp); err != nil {
		return err
	}

	return wrapOnlyErrorFromBase(resp)
}

// ToggleSnapshotSticky sets or removes sticky attribute for a snapshot
func (c *Client) ToggleSnapshotSticky(ctx context.Context, fileName string, sticky bool) error {
	stickyStr := "0"
	if sticky {
		stickyStr = "1"
	}

	var resp BaseResponse
	if err := c.doRequest(ctx, "snapshot/toggleSticky", map[string]string{
		"snapshot": fileName,
		"sticky":   stickyStr,
	}, &resp); err != nil {
		return err
	}

	return wrapOnlyErrorFromBase(resp)
}

// ExportSnapshot generates a token for transferring snapshot to another instance
func (c *Client) ExportSnapshot(ctx context.Context, fileName string) (*SnapshotExportResponse, error) {
	var resp SnapshotExportResponse
	if err := c.doRequest(ctx, "snapshot/export", map[string]string{"snapshot": fileName}, &resp); err != nil {
		return nil, err
	}

	return wrapErrorWithBase(&resp, resp.BaseResponse)
}

// ImportSnapshot imports a snapshot from another instance using VEID and token
func (c *Client) ImportSnapshot(ctx context.Context, sourceVeid, sourceToken string) error {
	var resp BaseResponse
	if err := c.doRequest(ctx, "snapshot/import", map[string]string{
		"sourceVeid":  sourceVeid,
		"sourceToken": sourceToken,
	}, &resp); err != nil {
		return err
	}

	return wrapOnlyErrorFromBase(resp)
}

// Restart restarts the VPS
func (c *Client) Restart(ctx context.Context) error {
	var resp BaseResponse
	if err := c.doRequest(ctx, "restart", nil, &resp); err != nil {
		return err
	}

	return wrapOnlyErrorFromBase(resp)
}

// Start starts the VPS
func (c *Client) Start(ctx context.Context) error {
	var resp BaseResponse
	if err := c.doRequest(ctx, "start", nil, &resp); err != nil {
		return err
	}

	return wrapOnlyErrorFromBase(resp)
}

// Stop stops the VPS
func (c *Client) Stop(ctx context.Context) error {
	var resp BaseResponse
	if err := c.doRequest(ctx, "stop", nil, &resp); err != nil {
		return err
	}

	return wrapOnlyErrorFromBase(resp)
}

// Kill forcefully stops a VPS that is stuck and cannot be stopped by normal means
// Please use this feature with great care as any unsaved data will be lost.
func (c *Client) Kill(ctx context.Context) error {
	var resp BaseResponse
	if err := c.doRequest(ctx, "kill", nil, &resp); err != nil {
		return err
	}

	return wrapOnlyErrorFromBase(resp)
}

// GetAvailableOS gets the list of available operating systems for reinstallation
func (c *Client) GetAvailableOS(ctx context.Context) (*AvailableOSResponse, error) {
	var resp AvailableOSResponse
	if err := c.doRequest(ctx, "getAvailableOS", nil, &resp); err != nil {
		return nil, err
	}

	return wrapErrorWithBase(&resp, resp.BaseResponse)
}

// ReinstallOS reinstalls the operating system
// WARNING: This will destroy all data on the VPS!
func (c *Client) ReinstallOS(ctx context.Context, osTemplate string) error {
	var resp BaseResponse
	if err := c.doRequest(ctx, "reinstallOS", map[string]string{"os": osTemplate}, &resp); err != nil {
		return err
	}

	return wrapOnlyErrorFromBase(resp)
}

// GetRawUsageStats gets detailed usage statistics
func (c *Client) GetRawUsageStats(ctx context.Context) (*UsageStatsResponse, error) {
	var resp UsageStatsResponse
	if err := c.doRequest(ctx, "getRawUsageStats", nil, &resp); err != nil {
		return nil, err
	}

	return wrapErrorWithBase(&resp, resp.BaseResponse)
}

// GetAuditLog gets audit log entries for the VPS
func (c *Client) GetAuditLog(ctx context.Context) (*AuditLogResponse, error) {
	var resp AuditLogResponse
	if err := c.doRequest(ctx, "getAuditLog", nil, &resp); err != nil {
		return nil, err
	}

	return wrapErrorWithBase(&resp, resp.BaseResponse)
}

// ResetRootPassword resets the root password and returns the new password
func (c *Client) ResetRootPassword(ctx context.Context) (*ResetRootPasswordResponse, error) {
	var resp ResetRootPasswordResponse
	if err := c.doRequest(ctx, "resetRootPassword", nil, &resp); err != nil {
		return nil, err
	}

	return wrapErrorWithBase(&resp, resp.BaseResponse)
}

// doRequest performs a generic API request
func (c *Client) doRequest(ctx context.Context, endpoint string, params map[string]string, result any) error {
	u, err := url.Parse(c.baseURL + "/" + endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Set("veid", c.veid)
	q.Set("api_key", c.apiKey)

	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", version.GetUserAgent())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status: %d %s", resp.StatusCode, resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// ListBackups lists all available backups
func (c *Client) ListBackups(ctx context.Context) (*BackupListResponse, error) {
	var resp BackupListResponse
	if err := c.doRequest(ctx, "backup/list", nil, &resp); err != nil {
		return nil, err
	}

	return wrapErrorWithBase(&resp, resp.BaseResponse)
}

// CopyBackupToSnapshot copies a backup to a restorable snapshot
func (c *Client) CopyBackupToSnapshot(ctx context.Context, backupToken string) error {
	var resp BaseResponse
	if err := c.doRequest(ctx, "backup/copyToSnapshot", map[string]string{
		"backupToken": backupToken,
	}, &resp); err != nil {
		return err
	}

	return wrapOnlyErrorFromBase(resp)
}

// SetHostname sets a new hostname for the VPS
func (c *Client) SetHostname(ctx context.Context, newHostname string) error {
	var resp BaseResponse
	if err := c.doRequest(ctx, "setHostname", map[string]string{
		"newHostname": newHostname,
	}, &resp); err != nil {
		return err
	}

	return wrapOnlyErrorFromBase(resp)
}

// GetRateLimitStatus gets current API rate limit status
func (c *Client) GetRateLimitStatus(ctx context.Context) (*RateLimitStatus, error) {
	var resp RateLimitStatus
	if err := c.doRequest(ctx, "getRateLimitStatus", nil, &resp); err != nil {
		return nil, err
	}

	return wrapErrorWithBase(&resp, resp.BaseResponse)
}

// GetSshKeys gets SSH keys from both Hypervisor Vault and Billing Portal
func (c *Client) GetSshKeys(ctx context.Context) (*SshKeysResponse, error) {
	var resp SshKeysResponse
	if err := c.doRequest(ctx, "getSshKeys", nil, &resp); err != nil {
		return nil, err
	}

	return wrapErrorWithBase(&resp, resp.BaseResponse)
}

// UpdateSshKeys updates per-VM SSH keys in Hypervisor Vault (replaces all existing keys)
func (c *Client) UpdateSshKeys(ctx context.Context, sshKeys []string) error {
	params := map[string]string{}

	// Join SSH keys with newlines as the API expects
	if len(sshKeys) > 0 {
		params["ssh_keys"] = fmt.Sprintf("%s\n", strings.Join(sshKeys, "\n"))
	} else {
		params["ssh_keys"] = ""
	}

	var resp BaseResponse
	if err := c.doRequest(ctx, "updateSshKeys", params, &resp); err != nil {
		return err
	}

	return wrapOnlyErrorFromBase(resp)
}

// SetPTR sets new PTR (rDNS) record for IP address
func (c *Client) SetPTR(ctx context.Context, ip, ptr string) error {
	var resp BaseResponse
	if err := c.doRequest(ctx, "setPTR", map[string]string{
		"ip":  ip,
		"ptr": ptr,
	}, &resp); err != nil {
		return err
	}

	return wrapOnlyErrorFromBase(resp)
}

// MountISO sets ISO image to boot from
// VM must be completely shut down and restarted after this API call
func (c *Client) MountISO(ctx context.Context, iso string) error {
	var resp BaseResponse
	if err := c.doRequest(ctx, "iso/mount", map[string]string{
		"iso": iso,
	}, &resp); err != nil {
		return err
	}

	return wrapOnlyErrorFromBase(resp)
}

// UnmountISO removes ISO image and configures VM to boot from primary storage
// VM must be completely shut down and restarted after this API call
func (c *Client) UnmountISO(ctx context.Context) error {
	var resp BaseResponse
	if err := c.doRequest(ctx, "iso/unmount", nil, &resp); err != nil {
		return err
	}

	return wrapOnlyErrorFromBase(resp)
}

// GetMigrateLocations returns all possible migration locations and metadata
func (c *Client) GetMigrateLocations(ctx context.Context) (*MigrateLocationsResponse, error) {
	var resp MigrateLocationsResponse
	if err := c.doRequest(ctx, "migrate/getLocations", nil, &resp); err != nil {
		return nil, err
	}

	return wrapErrorWithBase(&resp, resp.BaseResponse)
}

// StartMigration starts VPS migration to a new location with a default 15-minute timeout
// NOTE: Starting migration will replace all IPv4 addresses of the VPS
func (c *Client) StartMigration(ctx context.Context, locationID string) (*MigrateStartResponse, error) {
	return c.StartMigrationWithTimeout(ctx, locationID, 15*time.Minute)
}

// StartMigrationWithTimeout starts VPS migration to a new location with a custom timeout
// NOTE: Starting migration will replace all IPv4 addresses of the VPS
func (c *Client) StartMigrationWithTimeout(ctx context.Context, locationID string, timeout time.Duration) (*MigrateStartResponse, error) {
	params := map[string]string{
		"location": locationID,
	}
	var resp MigrateStartResponse
	if err := c.doRequestWithTimeout(ctx, "migrate/start", params, &resp, timeout); err != nil {
		return nil, err
	}

	return wrapErrorWithBase(&resp, resp.BaseResponse)
}

// doRequestWithTimeout performs a generic API request using a custom timeout for long-running operations
func (c *Client) doRequestWithTimeout(ctx context.Context, endpoint string, params map[string]string, result any, timeout time.Duration) error {
	u, err := url.Parse(c.baseURL + "/" + endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Set("veid", c.veid)
	q.Set("api_key", c.apiKey)

	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	// Apply context deadline as well as client timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctxWithTimeout, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", version.GetUserAgent())

	customClient := &http.Client{Timeout: timeout}
	resp, err := customClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status: %d %s", resp.StatusCode, resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// wrapError wraps a response with error checking - returns (result, error)
func wrapError[T any](resp T, errorCode int, message string) (T, error) {
	if errorCode != 0 {
		var zero T
		return zero, &BWHError{
			Code:    errorCode,
			Message: message,
		}
	}
	return resp, nil
}

// wrapErrorWithBase wraps a response with error checking using BaseResponse - returns (result, error)
func wrapErrorWithBase[T any](resp T, base BaseResponse) (T, error) {
	if base.Error != 0 {
		var zero T
		return zero, &BWHError{
			Code:                  base.Error,
			Message:               base.Message,
			AdditionalErrorInfo:   base.AdditionalErrorInfo,
			AdditionalLockingInfo: base.AdditionalLockingInfo,
		}
	}
	return resp, nil
}

// wrapOnlyError wraps a response with error checking - returns only error
func wrapOnlyError(errorCode int, message string) error {
	if errorCode != 0 {
		return &BWHError{
			Code:    errorCode,
			Message: message,
		}
	}
	return nil
}

// wrapOnlyErrorFromBase wraps a response with error checking using BaseResponse - returns only error
func wrapOnlyErrorFromBase(base BaseResponse) error {
	if base.Error != 0 {
		return &BWHError{
			Code:                  base.Error,
			Message:               base.Message,
			AdditionalErrorInfo:   base.AdditionalErrorInfo,
			AdditionalLockingInfo: base.AdditionalLockingInfo,
		}
	}
	return nil
}

// AddIPv6 assigns a new IPv6 /64 subnet to the VPS
func (c *Client) AddIPv6(ctx context.Context) (*IPv6AddResponse, error) {
	var resp IPv6AddResponse
	if err := c.doRequest(ctx, "ipv6/add", nil, &resp); err != nil {
		return nil, err
	}

	return wrapErrorWithBase(&resp, resp.BaseResponse)
}

// DeleteIPv6 releases a specified IPv6 /64 subnet from the VPS
func (c *Client) DeleteIPv6(ctx context.Context, subnet string) error {
	params := map[string]string{
		"ip": subnet,
	}

	var resp BaseResponse
	if err := c.doRequest(ctx, "ipv6/delete", params, &resp); err != nil {
		return err
	}

	return wrapOnlyErrorFromBase(resp)
}
