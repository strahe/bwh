package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/strahe/bwh/internal/config"
	"github.com/strahe/bwh/internal/version"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:                  "bwh",
		Usage:                 "manage your BWH instances",
		Version:               version.GetVersion(),
		EnableShellCompletion: true,
		ShellComplete:         shellComplete,
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
			isoCmd,
			reinstallCmd,
			usageCmd,
			auditCmd,
			resetPasswordCmd,
			snapshotCmd,
			backupCmd,
			migrateCmd,
			ipv6Cmd,
			privateIPCmd,
			mcpCmd,
			updateCmd,
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func shellComplete(ctx context.Context, cmd *cli.Command) {
	args := os.Args

	// Check if completing instance flag value
	for i, arg := range args {
		if (arg == "--instance" || arg == "-i") && i+1 < len(args) {
			configManager, err := config.NewManager(cmd.String("config"))
			if err != nil {
				return
			}
			for _, instance := range configManager.GetAvailableInstances() {
				fmt.Println(instance)
			}
			return
		}
	}
}
