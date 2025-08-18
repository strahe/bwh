package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

// Version information (set by build)
var (
	Version    = "dev"
	BuildTime  = "unknown"
	CommitHash = "unknown"
)

func main() {
	cmd := &cli.Command{
		Name:    "bwh",
		Usage:   "manage your BWH instances",
		Version: Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "path to config file",
				Aliases: []string{"c"},
			},
			&cli.StringFlag{
				Name:    "instance",
				Usage:   "BWH instance to use",
				Aliases: []string{"i"},
			},
		},
		Commands: []*cli.Command{
			nodeCmd,
			infoCmd,
			rateLimitCmd,
			connectCmd,
			sshCmd,
			startCmd,
			stopCmd,
			restartCmd,
			killCmd,
			hostnameCmd,
			setPTRCmd,
			reinstallCmd,
			usageCmd,
			auditCmd,
			resetPasswordCmd,
			snapshotCmd,
			backupCmd,
			mcpCmd,
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
