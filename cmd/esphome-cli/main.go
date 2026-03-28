package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/richard87/esphome-apiclient/cmd/esphome-cli/command"
	"github.com/urfave/cli/v3"
)

var cmd = &cli.Command{
	Name:                  "esphome-cli",
	Usage:                 "ESPHome API client CLI tool",
	EnableShellCompletion: true,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "yaml",
			Aliases: []string{"y"},
			Usage:   "Path to ESPHome device YAML config file",
		},
		&cli.StringFlag{
			Name:    "address",
			Aliases: []string{"a"},
			Usage:   "Device address (host:port). Overrides YAML-derived address.",
		},
		&cli.StringFlag{
			Name:    "key",
			Aliases: []string{"k"},
			Usage:   "Base64 Noise encryption key. Overrides YAML config.",
		},
		&cli.StringFlag{
			Name:    "name",
			Aliases: []string{"n"},
			Usage:   "Expected device name (for noise validation)",
		},
		&cli.DurationFlag{
			Name:    "timeout",
			Aliases: []string{"t"},
			Value:   5 * time.Second,
			Usage:   "Connection timeout",
		},
	},
	Commands: []*cli.Command{
		{
			Name:  "scan",
			Usage: "Scan for ESPHome devices on local network",
			Action: func(ctx context.Context, c *cli.Command) error {
				timeout := c.Root().Duration("timeout")
				if timeout == 0 {
					timeout = 5 * time.Second
				}

				return command.RunScan(ctx, timeout)
			},
		},
		{
			Name:  "info",
			Usage: "Show device info",
			Action: func(ctx context.Context, c *cli.Command) error {
				client, err := getClient(ctx, c)
				if err != nil {
					return err
				}
				defer client.Close()
				return command.RunInfo(ctx, client)
			},
		},
		{
			Name:  "entities",
			Usage: "List entities",
			Action: func(ctx context.Context, c *cli.Command) error {
				client, err := getClient(ctx, c)
				if err != nil {
					return err
				}
				defer client.Close()
				return command.RunEntities(ctx, client)
			},
		},
		{
			Name:  "sensors",
			Usage: "Stream sensor values",
			Action: func(ctx context.Context, c *cli.Command) error {
				client, err := getClient(ctx, c)
				if err != nil {
					return err
				}
				defer client.Close()
				return command.RunSensors(ctx, client)
			},
		},
		{
			Name:  "logs",
			Usage: "Stream logs",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "level",
					Value: "DEBUG",
					Usage: "Log level (NONE, ERROR, WARN, INFO, CONFIG, DEBUG, VERBOSE, VERY_VERBOSE)",
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				client, err := getClient(ctx, c)
				if err != nil {
					return err
				}
				defer client.Close()
				return command.RunLogs(ctx, client, c.String("level"))
			},
		},
		{
			Name:  "switch",
			Usage: "Control a switch",
			Flags: []cli.Flag{
				&cli.UintFlag{
					Name:  "switch-key",
					Usage: "Switch entity key",
				},
				&cli.StringFlag{
					Name:  "switch-state",
					Usage: "Switch state: on or off",
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				client, err := getClient(ctx, c)
				if err != nil {
					return err
				}
				defer client.Close()
				return command.RunSwitch(ctx, client, uint32(c.Uint("switch-key")), c.String("switch-state"))
			},
		},
		{
			Name:  "bluetooth",
			Usage: "Stream Bluetooth LE advertisements",
			Action: func(ctx context.Context, c *cli.Command) error {
				client, err := getClient(ctx, c)
				if err != nil {
					return err
				}
				defer client.Close()
				return command.RunBluetooth(ctx, client)
			},
		},
	},
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := cmd.Run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
