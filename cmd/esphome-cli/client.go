package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	esphome "github.com/richard87/esphome-apiclient"
	"github.com/urfave/cli/v3"
)

func getClient(ctx context.Context, cmd *cli.Command) (*esphome.Client, error) {
	yamlPath := cmd.Root().String("yaml")
	address := cmd.Root().String("address")
	encKey := cmd.Root().String("key")
	deviceName := cmd.Root().String("name")
	timeout := cmd.Root().Duration("timeout")

	// If YAML is provided, parse it
	if yamlPath != "" {
		cfg, err := parseYaml(yamlPath)
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
