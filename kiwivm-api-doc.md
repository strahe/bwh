# KiwiVM REST API Reference

This is an unofficial project reference for KiwiVM REST endpoints used by `bwh` and for endpoints that may be added later. It describes the upstream KiwiVM API surface, not only the methods currently implemented by the Go client.

## Overview

| Item | Value |
| --- | --- |
| Base URL | `https://api.64clouds.com/v1` |
| Request methods | `GET` or `POST` |
| POST content type | `application/x-www-form-urlencoded` |
| Response format | JSON |
| Authentication | Per-request `veid` and `api_key` parameters |

KiwiVM accepts all endpoint parameters through either `GET` query parameters or `POST` form data. In this project, read-only client methods should use `GET`; state-changing methods should use `POST` form data so credentials and write parameters are not placed in the URL.

## Authentication

Every endpoint requires these common parameters.

| Parameter | Required | Description |
| --- | --- | --- |
| `veid` | Yes | VPS ID. |
| `api_key` | Yes | API key for the VPS. |

GET format:

```http
GET https://api.64clouds.com/v1/{endpoint}?veid={VEID}&api_key={API_KEY}
```

POST format:

```bash
curl -sS -X POST "https://api.64clouds.com/v1/{endpoint}" \
  -d "veid={VEID}" \
  -d "api_key={API_KEY}"
```

GET with endpoint parameters:

```http
GET https://api.64clouds.com/v1/setPTR?veid={VEID}&api_key={API_KEY}&ip=203.0.113.10&ptr=host.example.com
```

## Common Response Format

Every endpoint returns an `error` field. A value of `0` means success. Non-zero values indicate an API-level failure.

| Field | Type | Description |
| --- | --- | --- |
| `error` | integer | `0` on success, non-zero on failure. |
| `message` | string | Error details when available. |
| `additionalErrorInfo` | string | Optional extra error context. |
| `additionalLockingInfo` | object | Optional lock status details when the VPS is busy. |

Success response:

```json
{
  "error": 0
}
```

Failure response:

```json
{
  "error": 1,
  "message": "Error details"
}
```

## Examples

Get service information:

```bash
curl -sS "https://api.64clouds.com/v1/getServiceInfo?veid={VEID}&api_key={API_KEY}"
```

Restart a VPS with POST:

```bash
curl -sS -X POST "https://api.64clouds.com/v1/restart" \
  -d "veid={VEID}" \
  -d "api_key={API_KEY}"
```

Create a snapshot:

```bash
curl -sS -X POST "https://api.64clouds.com/v1/snapshot/create" \
  -d "veid={VEID}" \
  -d "api_key={API_KEY}" \
  -d "description=Automatic_Snapshot"
```

Set a PTR record:

```bash
curl -sS -X POST "https://api.64clouds.com/v1/setPTR" \
  -d "veid={VEID}" \
  -d "api_key={API_KEY}" \
  -d "ip=203.0.113.10" \
  -d "ptr=host.example.com"
```

## High-Risk Operations

These endpoints can affect data, security, networking, service availability, or abuse-policy state. CLI flows must keep explicit confirmation for them.

| Endpoint | Risk |
| --- | --- |
| `stop` | Stops the VPS. |
| `restart` | Reboots the VPS. |
| `kill` | Force-stops the VPS and can lose unsaved data. |
| `reinstallOS` | Reinstalls the OS and destroys current data. |
| `resetRootPassword` | Replaces the root password. |
| `setHostname` | Changes the VPS hostname. |
| `setPTR` | Changes reverse DNS for an IP address. |
| `iso/mount` | Changes boot media. |
| `iso/unmount` | Changes boot media back to primary storage. |
| `basicShell/exec` | Runs a root shell command synchronously on the VPS. |
| `shellScript/exec` | Runs a root shell script asynchronously on the VPS. |
| `snapshot/create` | Creates a snapshot task that can restart and temporarily lock the VPS. |
| `snapshot/delete` | Deletes a snapshot. |
| `snapshot/restore` | Restores a snapshot and overwrites current VPS data. |
| `snapshot/toggleSticky` | Changes snapshot retention behavior. |
| `backup/copyToSnapshot` | Creates a restorable snapshot from a backup. |
| `ipv6/add` | Allocates a new IPv6 /64 subnet. |
| `ipv6/delete` | Releases an IPv6 /64 subnet. |
| `privateIp/assign` | Assigns a private IPv4 address. |
| `privateIp/delete` | Removes a private IPv4 address. |
| `migrate/start` | Migrates the VPS and replaces all IPv4 addresses. |
| `cloneFromExternalServer` | Clones a remote server into the VPS. |
| `unsuspend` | Clears an abuse case and unsuspends the VPS when allowed. |
| `resolvePolicyViolation` | Marks an active policy violation as resolved. |
| `kiwivm/setNotificationPreferences` | Changes notification settings. |

## Endpoint Index

| Category | Endpoints |
| --- | --- |
| Power management | `start`, `stop`, `restart`, `kill` |
| Service information | `getServiceInfo`, `getLiveServiceInfo` |
| OS and SSH | `getAvailableOS`, `reinstallOS`, `updateSshKeys`, `getSshKeys`, `resetRootPassword` |
| Usage and audit | `getUsageGraphs`, `getRawUsageStats`, `getAuditLog`, `getRateLimitStatus` |
| Hostname, DNS, ISO | `setHostname`, `setPTR`, `iso/mount`, `iso/unmount` |
| Shell | `basicShell/cd`, `basicShell/exec`, `shellScript/exec` |
| Snapshots and backups | `snapshot/create`, `snapshot/list`, `snapshot/delete`, `snapshot/restore`, `snapshot/toggleSticky`, `snapshot/export`, `snapshot/import`, `backup/list`, `backup/copyToSnapshot` |
| Network | `ipv6/add`, `ipv6/delete`, `privateIp/getAvailableIps`, `privateIp/assign`, `privateIp/delete` |
| Migration | `migrate/getLocations`, `migrate/start`, `cloneFromExternalServer` |
| Suspension and policy | `getSuspensionDetails`, `getPolicyViolations`, `unsuspend`, `resolvePolicyViolation` |
| KiwiVM settings | `kiwivm/getNotificationPreferences`, `kiwivm/setNotificationPreferences` |

## Endpoint Details

Unless noted otherwise, every endpoint supports `GET` and `POST`, requires the common authentication parameters, and returns the common `error` field.

### `start`

- Path: `/start`
- Parameters: common only.
- Returns: common response.
- Notes: Starts the VPS.

### `stop`

- Path: `/stop`
- Parameters: common only.
- Returns: common response.
- Notes: Stops the VPS.

### `restart`

- Path: `/restart`
- Parameters: common only.
- Returns: common response.
- Notes: Reboots the VPS.

### `kill`

- Path: `/kill`
- Parameters: common only.
- Returns: common response.
- Notes: Force-stops a VPS that cannot be stopped normally. Unsaved data can be lost.

### `getServiceInfo`

- Path: `/getServiceInfo`
- Parameters: common only.
- Returns: base VPS service information.

Known response fields:

| Field | Description |
| --- | --- |
| `vm_type` | Hypervisor type, such as `ovz` or `kvm`. |
| `hostname` | VPS hostname. |
| `node_alias` | Internal physical node name. |
| `node_location_id` | Location identifier. |
| `node_location` | Physical location. |
| `node_datacenter` | Datacenter description. |
| `location_ipv6_ready` | Whether the current location supports IPv6. |
| `plan` | Plan name. |
| `plan_disk` | Disk quota in bytes. |
| `plan_ram` | RAM in bytes. |
| `plan_swap` | Swap in bytes. |
| `os` | Installed operating system. |
| `email` | Primary account email address. |
| `plan_monthly_data` | Monthly transfer allowance before applying `monthly_data_multiplier`. |
| `data_counter` | Current billing-month transfer usage before applying `monthly_data_multiplier`. |
| `monthly_data_multiplier` | Transfer accounting multiplier for the location. |
| `data_next_reset` | Transfer counter reset time as a Unix timestamp. |
| `ip_addresses` | Assigned IPv4 addresses and IPv6 /64 subnets. |
| `ipv6_sit_tunnel_endpoint` | IPv6 SIT tunnel endpoint when present. |
| `private_ip_addresses` | Assigned private IPv4 addresses. |
| `ip_nullroutes` | Nullroute details keyed by IP address. |
| `iso1` | Mounted ISO image 1. |
| `iso2` | Mounted ISO image 2; currently unsupported upstream. |
| `available_isos` | ISO images available for mounting. |
| `plan_max_ipv6s` | Maximum IPv6 /64 subnets allowed by the plan. |
| `rdns_api_available` | Whether rDNS records can be changed through the API. |
| `plan_private_network_available` | Whether private networking is available on the plan. |
| `location_private_network_available` | Whether private networking is available at the location. |
| `ptr` | rDNS records keyed by IP address. |
| `free_ip_replacement_interval` | Deprecated IP replacement interval value. |
| `suspended` | Whether the VPS is suspended. |
| `policy_violation` | Whether an active policy violation needs attention. |
| `suspension_count` | Suspension count for the current calendar year. |
| `total_abuse_points` | Abuse points accumulated in the current calendar year. |
| `max_abuse_points` | Abuse point limit for the plan and calendar year. |

`ip_nullroutes` can be an empty array, `null`, or an object keyed by IP. Non-empty entries can include `nullroute_timestamp`, `nullroute_duration_s`, and `log`.

### `getLiveServiceInfo`

- Path: `/getLiveServiceInfo`
- Parameters: common only.
- Returns: all `getServiceInfo` fields plus live status fields.
- Notes: This call can take up to 15 seconds.

Common live fields:

| Field | Description |
| --- | --- |
| `is_cpu_throttled` | `0` when CPU is not throttled, `1` when CPU throttling is active. |
| `ssh_port` | SSH port. For KVM this may only be returned while the VPS is running. |

OpenVZ-specific fields:

| Field | Description |
| --- | --- |
| `vz_status` | Beancounters, load, process, file, socket, memory, and related runtime data. |
| `vz_quota` | Disk and inode quota data. |

KVM-specific fields:

| Field | Description |
| --- | --- |
| `ve_status` | VM state such as `Starting`, `Running`, or `Stopped`. |
| `ve_mac1` | MAC address of the primary network interface. |
| `ve_used_disk_space_b` | Occupied mapped disk space in bytes. |
| `ve_disk_quota_gb` | Disk image size in GB. |
| `is_disk_throttled` | `0` when disk I/O is not throttled, `1` when throttling is active. |
| `live_hostname` | Hostname reported from inside the VPS. |
| `load_average` | Raw load average string. |
| `mem_available_kb` | Available RAM in KB. |
| `swap_total_kb` | Total swap in KB. |
| `swap_available_kb` | Available swap in KB. |
| `screendump_png_base64` | Base64-encoded PNG screenshot of the VGA console. |

### `getAvailableOS`

- Path: `/getAvailableOS`
- Parameters: common only.
- Returns: available OS templates.

| Field | Description |
| --- | --- |
| `installed` | Currently installed operating system. |
| `templates` | Installable OS templates. |

### `reinstallOS`

- Path: `/reinstallOS`
- Parameters: `os` - required OS template from `getAvailableOS`.
- Returns: reinstall task details.
- Notes: Destroys current VPS data.

| Field | Description |
| --- | --- |
| `rootPassword` | New root password. |
| `sshPort` | SSH port. |
| `sshKeys` | SSH keys written to `/root/.ssh/authorized_keys`. |
| `sshKeysBrief` | Shortened SSH key display values. |
| `notificationEmail` | Email address notified when the task completes. |

### `updateSshKeys`

- Path: `/updateSshKeys`
- Parameters: `ssh_keys` - required per-VM SSH keys.
- Returns: common response.
- Notes: These keys are stored in Hypervisor Vault and override account-level keys during future `reinstallOS` calls.

### `getSshKeys`

- Path: `/getSshKeys`
- Parameters: common only.
- Returns: per-VM, account-level, and preferred SSH keys.

| Field | Description |
| --- | --- |
| `ssh_keys_veid` | Per-VM SSH keys stored in Hypervisor Vault. |
| `ssh_keys_user` | Account-level SSH keys from the billing portal. |
| `ssh_keys_preferred` | Keys that `reinstallOS` will use; per-VM keys take priority. |
| `shortened_ssh_keys_veid` | Shortened per-VM keys. |
| `shortened_ssh_keys_user` | Shortened account-level keys. |
| `shortened_ssh_keys_preferred` | Shortened preferred keys. |

### `resetRootPassword`

- Path: `/resetRootPassword`
- Parameters: common only.
- Returns: `password` - the new root password.

### `getUsageGraphs`

- Path: `/getUsageGraphs`
- Parameters: common only.
- Returns: legacy usage graph data.
- Notes: Obsolete upstream. Use `getRawUsageStats`.

### `getRawUsageStats`

- Path: `/getRawUsageStats`
- Parameters: common only.
- Returns: detailed usage statistics from KiwiVM.

Known response fields in this project:

| Field | Description |
| --- | --- |
| `vm_type` | VM type, such as `kvm` or `ovz`. |
| `data` | Usage data points. |

Known data point fields:

| Field | Description |
| --- | --- |
| `timestamp` | Unix timestamp. |
| `cpu_usage` | CPU usage percentage. |
| `network_in_bytes` | Incoming network bytes. |
| `network_out_bytes` | Outgoing network bytes. |
| `disk_read_bytes` | Disk read bytes. |
| `disk_write_bytes` | Disk write bytes. |

### `getAuditLog`

- Path: `/getAuditLog`
- Parameters: common only.
- Returns: `log_entries` - audit log entries from KiwiVM.

Known entry fields:

| Field | Description |
| --- | --- |
| `timestamp` | Event time as a Unix timestamp. |
| `requestor_ipv4` | Requestor IPv4 address encoded as an integer. |
| `type` | Event type code. |
| `summary` | Human-readable event summary. |

### `setHostname`

- Path: `/setHostname`
- Parameters: `newHostname` - required hostname.
- Returns: common response.

### `setPTR`

- Path: `/setPTR`
- Parameters: `ip` - required IP address; `ptr` - required PTR/rDNS value.
- Returns: common response.

### `iso/mount`

- Path: `/iso/mount`
- Parameters: `iso` - required ISO image name.
- Returns: common response.
- Notes: The VM must be fully shut down. Restart after changing the mounted ISO.

### `iso/unmount`

- Path: `/iso/unmount`
- Parameters: common only.
- Returns: common response.
- Notes: Removes the ISO boot image and configures the VM to boot from primary storage. The VM must be fully shut down. Restart after changing the boot media.

### `basicShell/cd`

- Path: `/basicShell/cd`
- Parameters: `currentDir` - required current directory; `newDir` - required target directory.
- Returns: `pwd` - directory after the change.
- Notes: Supports building a Basic shell style interaction.

### `basicShell/exec`

- Path: `/basicShell/exec`
- Parameters: `command` - required shell command.
- Returns: `error` - command exit status; `message` - command output.
- Notes: Runs synchronously inside the VPS.

### `shellScript/exec`

- Path: `/shellScript/exec`
- Parameters: `script` - required shell script.
- Returns: `log` - output log file name.
- Notes: Runs asynchronously inside the VPS.

### `snapshot/create`

- Path: `/snapshot/create`
- Parameters: `description` - optional snapshot description.
- Returns: `notificationEmail` - email address notified when the task completes.

### `snapshot/list`

- Path: `/snapshot/list`
- Parameters: common only.
- Returns: `snapshots` - snapshot list.

Known snapshot fields:

| Field | Description |
| --- | --- |
| `fileName` | Snapshot file name used by snapshot operations. |
| `os` | Snapshot operating system. |
| `description` | Snapshot description. |
| `size` | Compressed snapshot size in bytes. |
| `uncompressed` | Uncompressed snapshot size in bytes. |
| `md5` | Snapshot MD5 hash. |
| `sticky` | Whether the snapshot is protected from automatic purge. |
| `purgesIn` | Seconds until automatic purge when not sticky. |
| `downloadLink` | HTTP download link. |
| `downloadLinkSSL` | HTTPS download link. |

### `snapshot/delete`

- Path: `/snapshot/delete`
- Parameters: `snapshot` - required `fileName` from `snapshot/list`.
- Returns: common response.

### `snapshot/restore`

- Path: `/snapshot/restore`
- Parameters: `snapshot` - required `fileName` from `snapshot/list`.
- Returns: common response.
- Notes: Restores the snapshot and overwrites current VPS data.

### `snapshot/toggleSticky`

- Path: `/snapshot/toggleSticky`
- Parameters: `snapshot` - required `fileName`; `sticky` - required `1` to set sticky or `0` to remove sticky.
- Returns: common response.
- Notes: Sticky snapshots are not automatically purged.

### `snapshot/export`

- Path: `/snapshot/export`
- Parameters: `snapshot` - required `fileName` from `snapshot/list`.
- Returns: `token` - transfer token for `snapshot/import`.

### `snapshot/import`

- Path: `/snapshot/import`
- Parameters: `sourceVeid` - required source VPS ID; `sourceToken` - required token from `snapshot/export`.
- Returns: common response.
- Notes: Imports a snapshot from another instance.

### `backup/list`

- Path: `/backup/list`
- Parameters: common only.
- Returns: `backups` - automatic backup list.

Known backup fields:

| Field | Description |
| --- | --- |
| `backupToken` | Backup token. Some responses expose this as the backup map key. |
| `size` | Backup size in bytes. |
| `os` | Backup operating system. |
| `md5` | Backup MD5 hash. |
| `timestamp` | Backup creation time as a Unix timestamp. |

### `backup/copyToSnapshot`

- Path: `/backup/copyToSnapshot`
- Parameters: `backupToken` - required token from `backup/list`.
- Returns: common response.
- Notes: Copies an automatic backup into a restorable snapshot.

### `ipv6/add`

- Path: `/ipv6/add`
- Parameters: common only.
- Returns: `assigned_subnet` - newly assigned IPv6 /64 subnet.

### `ipv6/delete`

- Path: `/ipv6/delete`
- Parameters: `ip` - required IPv6 /64 subnet.
- Returns: common response.
- Notes: Releases the specified IPv6 /64 subnet.

### `migrate/getLocations`

- Path: `/migrate/getLocations`
- Parameters: common only.
- Returns: migration target metadata.

| Field | Description |
| --- | --- |
| `currentLocation` | Current location ID. |
| `locations` | Location IDs available for migration. |
| `descriptions` | Friendly location descriptions keyed by location ID. |
| `dataTransferMultipliers` | Transfer allowance multipliers keyed by location ID. |

### `migrate/start`

- Path: `/migrate/start`
- Parameters: `location` - required target location ID from `migrate/getLocations`.
- Returns: migration task details.
- Notes: Replaces all IPv4 addresses on the VPS.

| Field | Description |
| --- | --- |
| `notificationEmail` | Email address notified when the task completes. |
| `newIps` | New IP addresses assigned to the VPS. |

### `cloneFromExternalServer`

- Path: `/cloneFromExternalServer`
- Parameters: `externalServerIP` - required source IP; `externalServerSSHport` - required SSH port; `externalServerRootPassword` - required root password.
- Returns: common response.
- Notes: OpenVZ only. Clones a remote server or VPS into this instance.

### `getSuspensionDetails`

- Path: `/getSuspensionDetails`
- Parameters: common only.
- Returns: suspension status and abuse evidence.

| Field | Description |
| --- | --- |
| `suspension_count` | Suspension count for the current calendar year. |
| `total_abuse_points` | Abuse points accumulated in the current calendar year. |
| `max_abuse_points` | Abuse point limit for the plan and calendar year. |
| `suspensions` | Outstanding suspension issues. |
| `evidence` | Complaint text or issue details keyed by evidence record ID. |

Known suspension fields:

| Field | Description |
| --- | --- |
| `record_id` | Case ID used by `unsuspend`. |
| `flag` | Abuse type. |
| `is_soft` | `1` when the issue can be cleared through API, `0` when support contact is required. |
| `evidence_record_id` | Evidence record ID. |
| `abuse_points` | Abuse points added by the case. |

### `getPolicyViolations`

- Path: `/getPolicyViolations`
- Parameters: common only.
- Returns: active policy violations.

| Field | Description |
| --- | --- |
| `total_abuse_points` | Abuse points accumulated in the current calendar year. |
| `max_abuse_points` | Abuse point limit for the plan and calendar year. |
| `policy_violations` | Active policy violations that need attention. |

Known policy violation fields:

| Field | Description |
| --- | --- |
| `record_id` | Case ID used by `resolvePolicyViolation`. |
| `timestamp` | Creation time as a Unix timestamp. |
| `suspend_at` | Time when the service will be suspended if unresolved. |
| `flag` | Violation type. |
| `is_soft` | `1` when the issue can be resolved through API, `0` when support contact is required. |
| `abuse_points` | Abuse points added by the violation. |
| `evidence_data` | Violation details. |

### `unsuspend`

- Path: `/unsuspend`
- Parameters: `record_id` - required case ID from `getSuspensionDetails`.
- Returns: common response.
- Notes: Clears the abuse issue and unsuspends the VPS when the case is API-resolvable.

### `resolvePolicyViolation`

- Path: `/resolvePolicyViolation`
- Parameters: `record_id` - required case ID from `getPolicyViolations`.
- Returns: common response.
- Notes: Marks the policy violation as resolved to avoid service suspension.

### `getRateLimitStatus`

- Path: `/getRateLimitStatus`
- Parameters: common only.
- Returns: current API rate limit budget.

| Field | Description |
| --- | --- |
| `remaining_points_15min` | Remaining points in the current 15-minute interval. |
| `remaining_points_24h` | Remaining points in the current 24-hour interval. |

### `privateIp/getAvailableIps`

- Path: `/privateIp/getAvailableIps`
- Parameters: common only.
- Returns: `available_ips` - private IPv4 addresses that can be assigned.

### `privateIp/assign`

- Path: `/privateIp/assign`
- Parameters: `ip` - optional private IPv4 address. If omitted, KiwiVM assigns one automatically.
- Returns: `assigned_ips` - private IPv4 addresses assigned by the request.

### `privateIp/delete`

- Path: `/privateIp/delete`
- Parameters: `ip` - required private IPv4 address.
- Returns: common response.

### `kiwivm/getNotificationPreferences`

- Path: `/kiwivm/getNotificationPreferences`
- Parameters: common only.
- Returns: notification settings and their current state.

| Field | Description |
| --- | --- |
| `email_preferences` | Available notification preferences and state. |
| `notificationEmail` | Email address receiving notifications. |

Known preference fields:

| Field | Description |
| --- | --- |
| `friendly_description` | Human-readable preference description. |
| `is_enabled` | Current enabled state. |
| `changed_timestamp` | Last change time as a Unix timestamp. |
| `s_value` | Stored preference value. |

### `kiwivm/setNotificationPreferences`

- Path: `/kiwivm/setNotificationPreferences`
- Parameters: `json_notification_preferences` - required JSON object mapping preference IDs to `0` or `1`.
- Returns: submitted changes, actual changes, and preference descriptions.

| Field | Description |
| --- | --- |
| `submitted_email_preferences` | Preferences submitted in the request. |
| `updated_email_preferences` | Preferences changed by KiwiVM. |
| `friendly_descriptions` | Friendly descriptions for preference IDs. |
