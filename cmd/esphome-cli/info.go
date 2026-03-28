package main

import (
	"context"
	"fmt"

	esphome "github.com/richard87/esphome-apiclient"
)

func runInfo(ctx context.Context, client *esphome.Client) error {
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
