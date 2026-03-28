package command

import (
	"context"
	"fmt"
	"strings"

	esphome "github.com/richard87/esphome-apiclient"
)

func RunEntities(ctx context.Context, client *esphome.Client) error {
	entities, err := client.ListEntities()
	if err != nil {
		return fmt.Errorf("failed to list entities: %w", err)
	}
	fmt.Printf("Found %d entities:\n\n", len(entities))

	reg := client.Entities()

	// Sensors
	sensors := reg.Sensors()
	if len(sensors) > 0 {
		fmt.Printf("Sensors (%d):\n", len(sensors))
		for _, s := range sensors {
			fmt.Printf("  [0x%08X] %s (%s) unit=%s\n", s.Key, s.Name, s.ObjectID, s.UnitOfMeasurement)
		}
		fmt.Println()
	}

	// Binary Sensors
	binarySensors := reg.BinarySensors()
	if len(binarySensors) > 0 {
		fmt.Printf("Binary Sensors (%d):\n", len(binarySensors))
		for _, s := range binarySensors {
			fmt.Printf("  [0x%08X] %s (%s)\n", s.Key, s.Name, s.ObjectID)
		}
		fmt.Println()
	}

	// Switches
	switches := reg.Switches()
	if len(switches) > 0 {
		fmt.Printf("Switches (%d):\n", len(switches))
		for _, s := range switches {
			fmt.Printf("  [0x%08X] %s (%s)\n", s.Key, s.Name, s.ObjectID)
		}
		fmt.Println()
	}

	// Lights
	lights := reg.Lights()
	if len(lights) > 0 {
		fmt.Printf("Lights (%d):\n", len(lights))
		for _, l := range lights {
			fmt.Printf("  [0x%08X] %s (%s)\n", l.Key, l.Name, l.ObjectID)
		}
		fmt.Println()
	}

	// Services
	services := client.Services().All()
	if len(services) > 0 {
		fmt.Printf("Services (%d):\n", len(services))
		for _, svc := range services {
			argNames := make([]string, 0, len(svc.Args))
			for _, a := range svc.Args {
				argNames = append(argNames, fmt.Sprintf("%s:%s", a.Name, a.Type))
			}
			fmt.Printf("  [0x%08X] %s(%s)\n", svc.Key, svc.Name, strings.Join(argNames, ", "))
		}
		fmt.Println()
	}
	return nil
}
