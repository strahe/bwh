package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/urfave/cli/v3"
)

// connectCmd connects to the resolved instance via SSH using key-based auth
var connectCmd = &cli.Command{
	Name:    "connect",
	Usage:   "SSH into the resolved instance (passwordless, using local SSH keys)",
	Aliases: []string{"c"},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "user",
			Aliases: []string{"u"},
			Usage:   "SSH username",
			Value:   "root",
		},
		&cli.IntFlag{
			Name:    "port",
			Aliases: []string{"p"},
			Usage:   "SSH port (overrides detected port)",
		},
		&cli.StringFlag{
			Name:    "identity",
			Aliases: []string{"i"},
			Usage:   "Path to identity file (passed to ssh -i)",
		},
		&cli.BoolFlag{
			Name:  "ipv6",
			Usage: "Prefer IPv6 address when selecting target IP",
		},
		&cli.StringFlag{
			Name:  "cmd",
			Usage: "Remote command to execute instead of opening an interactive shell",
		},
		&cli.BoolFlag{
			Name:  "no-host-check",
			Usage: "Disable StrictHostKeyChecking and do not record host keys",
		},
		&cli.StringSliceFlag{
			Name:  "ssh-args",
			Usage: "Additional raw arguments to pass to the ssh binary",
		},
		&cli.BoolFlag{
			Name:    "print",
			Aliases: []string{"dry-run"},
			Usage:   "Print the ssh command without executing it",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		// Ensure ssh binary exists
		if _, err := exec.LookPath("ssh"); err != nil {
			return fmt.Errorf("ssh binary not found in PATH: %w", err)
		}

		bwhClient, resolvedName, err := createBWHClient(cmd)
		if err != nil {
			return err
		}

		fmt.Printf("Resolving connection target for instance: %s\n", resolvedName)

		liveInfo, err := bwhClient.GetLiveServiceInfo(ctx)
		if err != nil {
			return fmt.Errorf("failed to get live service info: %w", err)
		}

		preferIPv6 := cmd.Bool("ipv6")
		ipAddr, err := selectTargetIP(liveInfo.IPAddresses, preferIPv6)
		if err != nil {
			return err
		}

		sshUser := cmd.String("user")
		sshPort := cmd.Int("port")
		if sshPort == 0 {
			if liveInfo.SSHPort > 0 {
				sshPort = liveInfo.SSHPort
			} else {
				sshPort = 22
			}
		}

		sshArgs := buildSSHArgs(cmd, sshUser, ipAddr, sshPort)

		if cmd.Bool("print") {
			fmt.Printf("ssh %s\n", strings.Join(sshArgs, " "))
			return nil
		}

		sshCmd := exec.CommandContext(ctx, "ssh", sshArgs...)
		sshCmd.Stdin = os.Stdin
		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr

		return sshCmd.Run()
	},
}

func selectTargetIP(allIPs []string, preferIPv6 bool) (string, error) {
	if len(allIPs) == 0 {
		return "", errors.New("no IP addresses found for the instance")
	}

	var ipv4s []string
	var ipv6s []string
	for _, addr := range allIPs {
		ip := parseIPFromAddress(addr)
		if ip == "" {
			continue
		}
		if strings.Contains(ip, ":") {
			ipv6s = append(ipv6s, ip)
		} else {
			ipv4s = append(ipv4s, ip)
		}
	}

	if preferIPv6 {
		if len(ipv6s) > 0 {
			return ipv6s[0], nil
		}
		if len(ipv4s) > 0 {
			return ipv4s[0], nil
		}
	} else {
		if len(ipv4s) > 0 {
			return ipv4s[0], nil
		}
		if len(ipv6s) > 0 {
			return ipv6s[0], nil
		}
	}

	return "", errors.New("no usable IP address found")
}

// parseIPFromAddress extracts a usable IP from values that may include IPv6 subnets
// or other decorations. The API can return IPv6 /64 subnets; we still prefer the
// base address for connection purposes.
func parseIPFromAddress(addr string) string {
	trimmed := strings.TrimSpace(addr)
	// If it looks like IPv6 with subnet, split by '/'
	if strings.Contains(trimmed, "/") {
		parts := strings.Split(trimmed, "/")
		trimmed = parts[0]
	}
	// Validate IP format
	ip := net.ParseIP(trimmed)
	if ip == nil {
		return ""
	}
	return trimmed
}

func buildSSHArgs(cmd *cli.Command, user string, host string, port int) []string {
	args := []string{"-p", fmt.Sprintf("%d", port)}

	if identity := cmd.String("identity"); identity != "" {
		args = append(args, "-i", identity)
	}

	if cmd.Bool("no-host-check") {
		args = append(args,
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
		)
	}

	// Ensure password auth is not attempted if keys are missing (explicitly honor passwordless intent)
	args = append(args, "-o", "PasswordAuthentication=no")

	// Extra raw args
	if extra := cmd.StringSlice("ssh-args"); len(extra) > 0 {
		args = append(args, extra...)
	}

	// Destination
	var destination string
	if strings.Contains(host, ":") {
		// IPv6 needs brackets
		destination = fmt.Sprintf("%s@[%s]", user, host)
	} else {
		destination = fmt.Sprintf("%s@%s", user, host)
	}
	args = append(args, destination)

	// Optional remote command
	if remoteCmd := cmd.String("cmd"); remoteCmd != "" {
		args = append(args, remoteCmd)
	}

	return args
}
