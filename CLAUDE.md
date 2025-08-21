# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go CLI, library, and MCP server for managing BandwagonHost VPS instances. Uses urfave/cli/v3 framework with multi-instance support via YAML config (~/.bwh/config.yaml).

## Development Commands

**REQUIRED before commits**: `make lint` (must pass with 0 issues)

Common commands:
- `make build` or `make` - Build binary
- `make dev` - Full workflow (format + lint + test + build)
- `make check` - Quick check (format + lint + test)
- `go test ./...` - Run tests

## Architecture

### Package Structure

- `cmd/bwh/` - CLI application entry point
  - `main.go` - Main CLI application using urfave/cli/v3
  - `helpers.go` - Shared client initialization logic (ALWAYS use these helpers)
  - `control.go` - VPS control commands (start, stop, restart, kill, hostname, set-ptr)
  - `info.go` - VPS information display (detailed/compact formats)
  - `node.go` - Multi-instance configuration management
  - `snapshot.go`, `backup.go`, `usage.go`, `audit.go`, `iso.go`, `private_ip.go`, etc. - Feature-specific commands
  - `mcp.go` - MCP server command
- `pkg/client/` - Public API client library
  - `client.go` - BWH API client with comprehensive VPS management methods
  - `types.go` - API response structures with detailed field documentation
- `internal/config/` - Configuration management (multi-instance YAML config)
- `internal/mcpserver/` - MCP server implementation for AI tool integration

### Key Patterns

- **Client Initialization**: ALL CLI commands must use `createBWHClient(cmd)` or `createBWHClientWithInstance(cmd)` helpers
- **API Endpoint**: `https://api.64clouds.com/v1`
- **Instance Resolution**: CLI flag > env var > default > single instance
- **Command Organization**: Commands grouped by functionality in separate files

## MCP Server Integration

The project includes MCP server implementation in `internal/mcpserver/` for safe AI tool integration.

### Usage
```bash
bwh mcp serve  # Start MCP server over stdio
```

### Available MCP Tools
- `vps_info_get` - Get VPS information (`instance?`, `compact?`, `live?`)
- `vps_usage_get` - Get usage statistics (`instance?`, `period?`, `days?`, `group_by?`)
- `snapshot_list` - List snapshots with filtering and sorting options
- `backup_list` - List backups with time range and filtering
- `vps_audit_get` - Get audit logs with time range and filtering
- `iso_list` - List available and mounted ISO images (`instance?`)
- `instance_list` - List all configured instances with metadata (no parameters)

### Common Usage Patterns
- `instance_list()` to discover all available instances before other operations
- `vps_info_get(compact=true)` for quick status overview
- `vps_info_get(live=true)` for real-time data and current status
- `vps_usage_get(days=N, group_by=day)` for usage statistics over N days
- `iso_list()` for available and mounted ISO images information
- When `instance` parameter is omitted, uses default from config

### Implementation Notes
- All tools are read-only and safe for AI use
- Implements proper MCP protocol with error handling
- Supports resource exposure for session information

## Client Initialization Pattern

**ALWAYS** use helper functions from `helpers.go`:

```go
// Standard commands
bwhClient, resolvedName, err := createBWHClient(cmd)

// When instance config needed
bwhClient, instance, resolvedName, err := createBWHClientWithInstance(cmd)
```

**DO NOT** duplicate client initialization code in commands.

## Documentation Synchronization

**CRITICAL**: Documentation must always stay synchronized with the actual project state.

### Rules
- **README files** (`README.md`, `README.zh.md`) must accurately reflect current features and commands
- **CLAUDE.md** must be updated when architecture, commands, or key patterns change
- **Update triggers**: When adding/removing commands, changing API methods, or modifying core functionality
- **Verification**: Always check if documentation updates are needed after code changes

### When to Update Documentation
- ✅ Adding new CLI commands or subcommands
- ✅ Adding new API client methods
- ✅ Changing command names, flags, or behavior
- ✅ Modifying package structure or key patterns
- ✅ Adding new MCP tools or capabilities
- ✅ Changing development workflows or build processes

### Documentation Files to Consider
- `README.md` - English documentation (commands list, API methods, usage examples)
- `README.zh.md` - Chinese documentation (同步更新)
- `CLAUDE.md` - Development guidance (architecture, patterns, commands location)

**Failure to maintain documentation synchronization leads to user confusion and development inefficiency.**

## Code Style

- **NO obvious comments** that restate code
- **DO document** exported functions with Go doc comments
- **ADD comments** only for complex business logic

Example of good comment:
```go
// FlexibleInt handles inconsistent BWH API responses that return
// numeric values as both strings and integers
// type FlexibleInt struct {
//     Value int64
// }
```
