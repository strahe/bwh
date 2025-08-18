package main

import (
	"context"
	"fmt"

	"github.com/strahe/bwh/internal/mcpserver"
	"github.com/urfave/cli/v3"
)

var mcpCmd = &cli.Command{
	Name:  "mcp",
	Usage: "run MCP server for read-only BWH management",
	Commands: []*cli.Command{
		mcpServeCmd,
	},
}

var mcpServeCmd = &cli.Command{
	Name:  "serve",
	Usage: "start MCP server over stdio (read-only tools)",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		// Defer to internal mcp server package, passing through config and instance flags
		configPath := cmd.String("config")
		instanceName := cmd.String("instance")

		if err := mcpserver.RunMCPStdioServer(ctx, configPath, instanceName); err != nil {
			return fmt.Errorf("failed to start MCP server: %w", err)
		}
		return nil
	},
}
