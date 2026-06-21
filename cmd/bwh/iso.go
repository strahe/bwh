package main

import (
	"context"
	"fmt"

	"github.com/strahe/bwh/pkg/client"
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
			Flags:     writeFlags(),
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

				return runMountISO(ctx, bwhClient, resolvedName, iso, cmd.Bool("dry-run"), skipConfirm(cmd), promptConfirmation)
			},
		},
		{
			Name:  "unmount",
			Usage: "unmount ISO image and boot from primary storage (requires VPS shutdown and restart)",
			Flags: writeFlags(),
			Action: func(ctx context.Context, cmd *cli.Command) error {
				bwhClient, resolvedName, err := createBWHClient(cmd)
				if err != nil {
					return err
				}

				return runUnmountISO(ctx, bwhClient, resolvedName, cmd.Bool("dry-run"), skipConfirm(cmd), promptConfirmation)
			},
		},
	},
}

type isoAPI interface {
	GetServiceInfo(context.Context) (*client.ServiceInfo, error)
	MountISO(context.Context, string) error
	UnmountISO(context.Context) error
}

func runMountISO(ctx context.Context, api isoAPI, resolvedName, iso string, dryRun, skipConfirm bool, confirm confirmationFunc) error {
	info, err := api.GetServiceInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get service info: %w", err)
	}
	if !containsString(info.AvailableISOs, iso) {
		return fmt.Errorf("ISO image %q is not available for instance %s", iso, resolvedName)
	}
	if info.ISO1 == iso {
		fmt.Printf("✅ ISO '%s' is already mounted (no change needed)\n", iso)
		return nil
	}
	if dryRun {
		printDryRun("iso/mount", resolvedName, fmt.Sprintf("iso: %s", iso))
		return nil
	}
	confirmed, err := confirmWrite(fmt.Sprintf("Mount ISO '%s' for VPS '%s'?", iso, resolvedName), skipConfirm, confirm)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	fmt.Printf("Mounting ISO '%s' for instance: %s\n", iso, resolvedName)
	fmt.Printf("⚠️  Remember: VPS must be completely shut down and restarted after this operation\n")
	if err := api.MountISO(ctx, iso); err != nil {
		return fmt.Errorf("failed to mount ISO: %w", err)
	}
	fmt.Printf("✅ ISO '%s' mounted successfully\n", iso)
	fmt.Printf("📝 Next steps: shutdown VPS completely and restart to boot from ISO\n")
	return nil
}

func runUnmountISO(ctx context.Context, api isoAPI, resolvedName string, dryRun, skipConfirm bool, confirm confirmationFunc) error {
	info, err := api.GetServiceInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get service info: %w", err)
	}
	if info.ISO1 == "" && info.ISO2 == "" {
		fmt.Printf("✅ No ISO is currently mounted (no change needed)\n")
		return nil
	}
	if dryRun {
		printDryRun("iso/unmount", resolvedName, fmt.Sprintf("mounted ISO1: %s", info.ISO1))
		return nil
	}
	confirmed, err := confirmWrite(fmt.Sprintf("Unmount ISO for VPS '%s'?", resolvedName), skipConfirm, confirm)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	fmt.Printf("Unmounting ISO for instance: %s\n", resolvedName)
	fmt.Printf("⚠️  Remember: VPS must be completely shut down and restarted after this operation\n")
	if err := api.UnmountISO(ctx); err != nil {
		return fmt.Errorf("failed to unmount ISO: %w", err)
	}
	fmt.Printf("✅ ISO unmounted successfully\n")
	fmt.Printf("📝 Next steps: shutdown VPS completely and restart to boot from primary storage\n")
	return nil
}
