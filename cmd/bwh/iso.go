package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

var isoCmd = &cli.Command{
	Name:  "iso",
	Usage: "manage ISO images for VPS boot",
	Commands: []*cli.Command{
		{
			Name:  "images",
			Usage: "list available ISO images and current mounted images",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				bwhClient, resolvedName, err := createBWHClient(cmd)
				if err != nil {
					return err
				}

				fmt.Printf("Getting ISO image information for instance: %s\n\n", resolvedName)

				serviceInfo, err := bwhClient.GetServiceInfo(ctx)
				if err != nil {
					return fmt.Errorf("failed to get service info: %w", err)
				}

				// Show available images
				fmt.Printf("📀 Available ISO Images:\n")
				if len(serviceInfo.AvailableISOs) == 0 {
					fmt.Printf("  No ISO images available\n")
				} else {
					for _, iso := range serviceInfo.AvailableISOs {
						fmt.Printf("  • %s\n", iso)
					}
				}

				// Show currently mounted images
				fmt.Printf("\n🔗 Currently Mounted:\n")
				if serviceInfo.ISO1 != "" {
					fmt.Printf("  • ISO1: %s\n", serviceInfo.ISO1)
				} else {
					fmt.Printf("  • ISO1: (none)\n")
				}
				if serviceInfo.ISO2 != "" {
					fmt.Printf("  • ISO2: %s\n", serviceInfo.ISO2)
				} else {
					fmt.Printf("  • ISO2: (none)\n")
				}

				fmt.Printf("\n💡 Usage:\n")
				fmt.Printf("  bwh iso mount <image_name>   # Mount an ISO image\n")
				fmt.Printf("  bwh iso unmount              # Unmount current ISO\n")
				fmt.Printf("\n⚠️  Note: VPS must be completely shut down and restarted after mount/unmount operations\n")

				return nil
			},
		},
		{
			Name:      "mount",
			Usage:     "mount ISO image to boot from (requires VPS shutdown and restart)",
			ArgsUsage: "<iso>",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "yes",
					Aliases: []string{"y"},
					Usage:   "skip confirmation prompt",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				if cmd.Args().Len() != 1 {
					return fmt.Errorf("iso mount command requires exactly one argument: <iso>")
				}

				iso := cmd.Args().Get(0)
				if iso == "" {
					return fmt.Errorf("ISO name cannot be empty")
				}

				bwhClient, resolvedName, err := createBWHClient(cmd)
				if err != nil {
					return err
				}

				// Confirmation prompt
				if !cmd.Bool("yes") {
					if !confirmAction("mount ISO", resolvedName, iso) {
						fmt.Println("Operation cancelled.")
						return nil
					}
				}

				fmt.Printf("Mounting ISO '%s' for instance: %s\n", iso, resolvedName)
				fmt.Printf("⚠️  Remember: VPS must be completely shut down and restarted after this operation\n")

				if err := bwhClient.MountISO(ctx, iso); err != nil {
					return fmt.Errorf("failed to mount ISO: %w", err)
				}

				fmt.Printf("✅ ISO '%s' mounted successfully\n", iso)
				fmt.Printf("📝 Next steps: shutdown VPS completely and restart to boot from ISO\n")
				return nil
			},
		},
		{
			Name:  "unmount",
			Usage: "unmount ISO image and boot from primary storage (requires VPS shutdown and restart)",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "yes",
					Aliases: []string{"y"},
					Usage:   "skip confirmation prompt",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				bwhClient, resolvedName, err := createBWHClient(cmd)
				if err != nil {
					return err
				}

				// Confirmation prompt
				if !cmd.Bool("yes") {
					if !confirmAction("unmount ISO", resolvedName) {
						fmt.Println("Operation cancelled.")
						return nil
					}
				}

				fmt.Printf("Unmounting ISO for instance: %s\n", resolvedName)
				fmt.Printf("⚠️  Remember: VPS must be completely shut down and restarted after this operation\n")

				if err := bwhClient.UnmountISO(ctx); err != nil {
					return fmt.Errorf("failed to unmount ISO: %w", err)
				}

				fmt.Printf("✅ ISO unmounted successfully\n")
				fmt.Printf("📝 Next steps: shutdown VPS completely and restart to boot from primary storage\n")
				return nil
			},
		},
	},
}
