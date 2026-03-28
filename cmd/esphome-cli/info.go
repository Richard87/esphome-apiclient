package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

func runInfoCmd(ctx context.Context, cmd *cli.Command) error {
	client, err := getClient(ctx, cmd)
	if err != nil {
		return err
	}
	defer client.Close()

	info, err := client.DeviceInfo()
	if err != nil {
		return fmt.Errorf("failed to get device info: %w", err)
	}
	fmt.Printf("Device Info:\n")
	fmt.Printf("  Name:            %s\n", info.Name)
	fmt.Printf("  MAC Address:     %s\n", info.MacAddress)
	fmt.Printf("  ESPHome Version: %s\n", info.EsphomeVersion)
	fmt.Printf("  Model:           %s\n", info.Model)
	fmt.Printf("  Project Name:    %s\n", info.ProjectName)
	fmt.Printf("  Project Version: %s\n", info.ProjectVersion)
	fmt.Printf("  Compilation Time:%s\n", info.CompilationTime)
	major, minor := client.APIVersion()
	fmt.Printf("  API Version:     %d.%d\n", major, minor)
	return nil
}
