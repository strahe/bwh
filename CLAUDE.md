# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go CLI, library, and MCP server for managing BandwagonHost VPS instances. Uses urfave/cli/v3 framework with multi-instance support via YAML config.

## Development Workflow

**REQUIRED before commits**: `make lint` (must pass with 0 issues)

Quick reference:
- `make build` - Build binary  
- `make check` - Lint + test + format
- `make dev` - Full workflow
- `bwh update --check` - Check for CLI updates

## Critical Patterns (ALWAYS FOLLOW)

### Helper Functions - MUST USE existing helpers in `cmd/bwh/helpers.go`:

```go
// Client setup (REQUIRED for all commands)
bwhClient, resolvedName, err := createBWHClient(cmd)
// OR when instance config needed:
bwhClient, instance, resolvedName, err := createBWHClientWithInstance(cmd)

// User confirmation (REQUIRED for y/N prompts)
confirmed, err := promptConfirmation("Continue?")
if err != nil || !confirmed { return nil }

// Token validation (use existing validators)
err := validateBackupToken(token)
```

### Pre-Implementation Checklist

Before writing new command functionality:
1. ✅ **Search first**: `grep -r "similar_functionality" cmd/bwh/`
2. ✅ **Client setup**: Using `createBWHClient*()` helpers?
3. ✅ **User confirmation**: Using `promptConfirmation()`?
4. ✅ **Progress display**: Using `internal/progress` package?
5. ✅ **Code reuse**: Checked `helpers.go` for existing functions?

### Common Anti-Patterns (AVOID)

❌ **Manual client setup** → ✅ Use `createBWHClient()` helpers  
❌ **Manual confirmation logic** → ✅ Use `promptConfirmation()`  
❌ **Duplicate progress bars** → ✅ Use `internal/progress`  
❌ **Manual input validation** → ✅ Check existing validators in `helpers.go`

## Architecture Reference

### Key Components
- `cmd/bwh/helpers.go` - **CRITICAL**: Contains required helper functions
- `internal/progress/` - Progress display for downloads
- `internal/updater/` - Self-update functionality  
- `internal/mcpserver/` - MCP server for AI integration
- `pkg/client/` - BWH API client library

### MCP Tools Summary
See `internal/mcpserver/server.go` for complete list. Key tools:
- `instance_list()` - List configured instances
- `vps_info_get(live=true)` - Real-time VPS status
- `vps_usage_get(days=N)` - Usage statistics

## Documentation Sync (REQUIRED)

**AUTOMATIC UPDATE RULE**: Every code change MUST sync docs immediately.

### Update Triggers (check ALL when making changes):
- ✅ New CLI commands/subcommands → Update READMEs + CLAUDE.md
- ✅ Modified command behavior/flags → Update READMEs  
- ✅ New API methods → Update README API section
- ✅ Architecture changes → Update CLAUDE.md
- ✅ New MCP tools → Update README + CLAUDE.md MCP section
- ✅ Helper function changes → Update CLAUDE.md patterns

### Files to Update:
- `README.md` - English user documentation
- `README.zh.md` - Chinese user documentation  
- `CLAUDE.md` - Development patterns and architecture

**Failure to sync docs = broken workflow for future development**

## Code Style

- Use existing helpers (prevent code duplication)
- `//nolint:errcheck` for expected ignored errors  
- No obvious comments that restate code
- Document exported functions with Go doc comments

**Remember**: Every time you write confirmation/client logic, check if it already exists in helpers.