# bwh

[![CI](https://github.com/strahe/bwh/workflows/CI/badge.svg)](https://github.com/strahe/bwh/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/strahe/bwh)](https://goreportcard.com/report/github.com/strahe/bwh)
[![Release](https://img.shields.io/github/v/release/strahe/bwh)](https://github.com/strahe/bwh/releases)
[![GoDoc](https://godoc.org/github.com/strahe/bwh?status.svg)](https://godoc.org/github.com/strahe/bwh)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> **English | [中文](README.zh.md)**

## Overview

Go SDK, CLI, and MCP server for managing BandwagonHost (KiwiVM) VPS, with command-line and programmatic access to most features.

## Installation

Prerequisites: Go 1.24+; KiwiVM VEID and API key.

```bash
# CLI or MCP
go install github.com/strahe/bwh/cmd/bwh@latest

# Go SDK
go get github.com/strahe/bwh
```

## Quick Start

```bash
# Configure your VPS
bwh node add production --api-key <API_KEY> --veid <VEID>

# Basic operations
bwh info                              # View server details
bwh start/stop/restart                # Power management
bwh usage --period 7d                 # Check usage statistics
bwh snapshot create "backup-name"     # Create snapshots
bwh iso images                        # List available ISO images
bwh iso mount ubuntu-20.04.iso        # Mount ISO for rescue/install
bwh ipv6 add                          # Assign new IPv6 /64 subnet
bwh ipv6 list                         # List IPv6 subnets
bwh pi info                           # Show private IPv4 info (alias of `private-ip info`)
bwh connect                           # SSH connection

# Explore more commands: bwh --help
```

### Multi-Instance Support

```bash
# Add multiple VPS instances
bwh node add prod --api-key <KEY> --veid <VEID>
bwh node add dev --api-key <KEY> --veid <VEID>

# Target specific instance or set default
bwh --instance prod info
bwh node set-default prod

# View all options: bwh node --help
```

## Go SDK

```go
import (
    "context"
    "log"
    "github.com/strahe/bwh/pkg/client"
)

// Initialize client
c := client.NewClient("your-api-key", "your-veid")
ctx := context.Background()

// Get server information
info, err := c.GetServiceInfo(ctx)
if err != nil {
    log.Fatal(err)
}

// Power management
err = c.Start(ctx)                    // Start VPS
err = c.Stop(ctx)                     // Stop VPS
err = c.Restart(ctx)                  // Restart VPS

// Monitoring
usage, err := c.GetRawUsageStats(ctx) // Usage statistics
live, err := c.GetLiveServiceInfo(ctx) // Real-time status

// Backup management
snapshot, err := c.CreateSnapshot(ctx, "backup-name")
backups, err := c.ListBackups(ctx)
```

### Available Methods

**Server Management**: `GetServiceInfo`, `GetLiveServiceInfo`, `Start`, `Stop`, `Restart`, `Kill`, `SetHostname`, `ReinstallOS`, `ResetRootPassword`, `MountISO`, `UnmountISO`

**Monitoring**: `GetRawUsageStats`, `GetBasicServiceInfo`, audit log access

**Backup & Recovery**: `CreateSnapshot`, `RestoreSnapshot`, `DeleteSnapshot`, backup management

**Migration**: `GetMigrateLocations`, `StartMigration` (use `StartMigrationWithTimeout` for custom timeouts)

**Network**: SSH key management, IP/reverse DNS configuration, IPv6 subnet management, private IPv4 management

*Complete API reference*: View [pkg/client documentation](./pkg/client) or run `go doc github.com/strahe/bwh/pkg/client` for all available methods.

## MCP Server Integration

BWH includes a built-in MCP (Model Context Protocol) server that enables secure AI integration with your VPS management workflows.

### Start MCP Server

```bash
bwh mcp serve
```

### Configuration

The BWH MCP server integrates seamlessly with various AI tools and editors. Add the appropriate configuration to your MCP client:

#### Claude Desktop

Add to your `claude_desktop_config.json` file:

```json
{
  "mcpServers": {
    "bwh": {
      "command": "bwh",
      "args": ["mcp", "serve"]
    }
  }
}
```



#### Claude Code

```bash
claude mcp add bwh -- bwh mcp serve
```



#### Cursor

Add to your Cursor MCP configuration:

```json
{
  "mcpServers": {
    "bwh": {
      "command": "bwh",
      "args": ["mcp", "serve"]
    }
  }
}
```



#### Continue (VS Code Extension)

Add to your Continue configuration:

```json
{
  "mcpServers": {
    "bwh": {
      "transport": {
        "type": "stdio",
        "command": "bwh",
        "args": ["mcp", "serve"]
      }
    }
  }
}
```



### Configuration Notes

- **Custom Config**: Use `--config /path/to/config.yaml` to specify a config file
- **Multiple Instances**: The server automatically uses your default instance from configuration
- **Integration**: Add to existing MCP config files without replacing other servers

### Available MCP Tools (Read-only)

- **vps_info_get**: Get VPS information (`instance?`, `compact?`, `live?`)
- **vps_usage_get**: Get usage statistics (`instance?`, `period?`, `days?`, `group_by?`)
- **snapshot_list**: List snapshots (`instance?`, `sticky_only?`, `name_contains?`, `sort_by?`, `order?`, `limit?`)
- **backup_list**: List backups (`instance?`, `os_contains?`, `since?`, `until?`, `sort_by?`, `order?`, `limit?`)
- **vps_audit_get**: Get audit logs (`instance?`, `since?`, `until?`, `limit?`, `ip_contains?`, `type?`)
- **iso_list**: List available and mounted ISO images (`instance?`)

All MCP tools are safe, read-only operations that won't modify your VPS configuration or data.

## Available Commands

```
node            Manage BWH VPS nodes configuration
info            Display comprehensive VPS information
rate-limit      Check API rate limit status
connect         SSH into VPS (passwordless, using local SSH keys)
ssh             Manage SSH keys
start/stop      Start/stop the VPS
restart         Restart the VPS
kill            Forcefully stop a stuck VPS (WARNING: potential data loss)
hostname        Set hostname for the VPS
set-ptr         Set PTR (rDNS) record for IP address
iso             Manage ISO images for VPS boot
reinstall       Reinstall VPS operating system (WARNING: destroys all data)
usage           Display detailed VPS usage statistics
audit           Display audit log entries
reset-password  Reset the root password
snapshot        Manage VPS snapshots
backup          Manage VPS backups
migrate         Migrate VPS to another location (supports --wait/--timeout)
ipv6            Manage IPv6 subnets (add, delete, list)
private-ip (pi) Manage Private IPv4 addresses (info, available, assign, delete)
mcp             Run MCP server for read-only BWH management
```

Use `bwh <command> --help` to view detailed options and usage examples for each command.

## Build

```bash
make build  # or: go build -o bwh ./cmd/bwh
```

Run tests: `make test`

## License

MIT. See `LICENSE`.
