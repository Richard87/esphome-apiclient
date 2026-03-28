package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	esphome "github.com/richard87/esphome-apiclient"
	"github.com/richard87/esphome-apiclient/cmd/esphome-cli/command"
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
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := cmd.Run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func getClient(ctx context.Context, cmd *cli.Command) (*esphome.Client, error) {
	yamlPath := cmd.Root().String("yaml")
	address := cmd.Root().String("address")
	encKey := cmd.Root().String("key")
	deviceName := cmd.Root().String("name")
	timeout := cmd.Root().Duration("timeout")

	// If YAML is provided, parse it
	if yamlPath != "" {
		cfg, err := command.ParseYAML(yamlPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
		if deviceName == "" {
			deviceName = cfg.ESPHome.Name
		}
		if encKey == "" {
			encKey = cfg.API.Encryption.Key
		}
		// Derive address from device name if not specified
		if address == "" && deviceName != "" {
			address = deviceName + ".local:6053"
		}
	}

	if address == "" {
		return nil, fmt.Errorf("--address or --yaml is required")
	}

	// Build client options
	var opts []esphome.Option
	opts = append(opts, esphome.WithClientInfo("esphome-cli"))
	if encKey != "" {
		opts = append(opts, esphome.WithEncryptionKey(encKey))
	}
	if deviceName != "" {
		opts = append(opts, esphome.WithExpectedName(deviceName))
	}
	logger := log.New(os.Stderr, "[esphome-cli] ", log.LstdFlags)
	opts = append(opts, esphome.WithLogger(logger))
	opts = append(opts, esphome.WithReconnect(5*time.Second))

	logger.Printf("Connecting to %s...", address)
	client, err := esphome.DialWithContext(ctx, address, timeout, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	return client, nil
}
