package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "esphome-cli",
		Usage: "ESPHome API client CLI tool",
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
				Name:   "scan",
				Usage:  "Scan for ESPHome devices on local network",
				Action: runScanCmd,
			},
			{
				Name:   "info",
				Usage:  "Show device info",
				Action: runInfoCmd,
			},
			{
				Name:   "entities",
				Usage:  "List entities",
				Action: runEntitiesCmd,
			},
			{
				Name:   "sensors",
				Usage:  "Stream sensor values",
				Action: runSensorsCmd,
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
				Action: runLogsCmd,
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
				Action: runSwitchCmd,
			},
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := cmd.Run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
