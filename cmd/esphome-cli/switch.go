package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/urfave/cli/v3"
)

func runSwitchCmd(ctx context.Context, cmd *cli.Command) error {
	client, err := getClient(ctx, cmd)
	if err != nil {
		return err
	}
	defer client.Close()

	key := uint32(cmd.Uint("switch-key"))
	stateStr := cmd.String("switch-state")

	if key == 0 {
		// List available switches
		if _, err := client.ListEntities(); err != nil {
			return fmt.Errorf("failed to list entities: %w", err)
		}
		switches := client.Entities().Switches()
		if len(switches) == 0 {
			fmt.Println("No switch entities found on this device.")
			return nil
		}

		sort.Slice(switches, func(i, j int) bool { return switches[i].Name < switches[j].Name })

		fmt.Println("Available switches:")
		for _, s := range switches {
			state := "OFF"
			if s.State {
				state = "ON"
			}
			fmt.Printf("  [0x%08X] %s = %s\n", s.Key, s.Name, state)
		}
		fmt.Println("\nUsage: switch --switch-key <KEY> --switch-state on|off")
		return nil
	}

	var state bool
	switch strings.ToLower(stateStr) {
	case "on", "true", "1":
		state = true
	case "off", "false", "0":
		state = false
	default:
		return fmt.Errorf("invalid switch state: %q (expected on/off)", stateStr)
	}

	// Populate entities for validation
	if _, err := client.ListEntities(); err != nil {
		log.Printf("Warning: could not list entities for validation: %v", err)
	}

	if err := client.SetSwitch(key, state); err != nil {
		return fmt.Errorf("failed to set switch: %w", err)
	}

	stateLabel := "OFF"
	if state {
		stateLabel = "ON"
	}
	fmt.Printf("Switch 0x%08X set to %s\n", key, stateLabel)
	return nil
}
