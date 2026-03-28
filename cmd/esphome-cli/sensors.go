package main

import (
	"context"
	"fmt"

	esphome "github.com/richard87/esphome-apiclient"
	"github.com/richard87/esphome-apiclient/pb"
	"github.com/urfave/cli/v3"
	"google.golang.org/protobuf/proto"
)

func runSensorsCmd(ctx context.Context, cmd *cli.Command) error {
	client, err := getClient(ctx, cmd)
	if err != nil {
		return err
	}
	defer client.Close()

	// First list entities to populate the registry
	if _, err := client.ListEntities(); err != nil {
		return fmt.Errorf("failed to list entities: %w", err)
	}

	fmt.Println("Streaming sensor values (press Ctrl+C to stop)...")
	fmt.Println()

	_, err = client.SubscribeStates(func(msg proto.Message) {
		reg := client.Entities()
		switch m := msg.(type) {
		case *pb.SensorStateResponse:
			entity := reg.ByKey(m.Key)
			name := fmt.Sprintf("0x%08X", m.Key)
			unit := ""
			if entity != nil {
				name = entity.GetName()
				if s, ok := entity.(*esphome.SensorEntity); ok {
					unit = s.UnitOfMeasurement
				}
			}
			if m.MissingState {
				fmt.Printf("[sensor] %s = <missing>\n", name)
			} else {
				fmt.Printf("[sensor] %s = %.4g %s\n", name, m.State, unit)
			}
		case *pb.BinarySensorStateResponse:
			entity := reg.ByKey(m.Key)
			name := fmt.Sprintf("0x%08X", m.Key)
			if entity != nil {
				name = entity.GetName()
			}
			if m.MissingState {
				fmt.Printf("[binary_sensor] %s = <missing>\n", name)
			} else {
				fmt.Printf("[binary_sensor] %s = %v\n", name, m.State)
			}
		case *pb.SwitchStateResponse:
			entity := reg.ByKey(m.Key)
			name := fmt.Sprintf("0x%08X", m.Key)
			if entity != nil {
				name = entity.GetName()
			}
			fmt.Printf("[switch] %s = %v\n", name, m.State)
		case *pb.TextSensorStateResponse:
			entity := reg.ByKey(m.Key)
			name := fmt.Sprintf("0x%08X", m.Key)
			if entity != nil {
				name = entity.GetName()
			}
			if m.MissingState {
				fmt.Printf("[text_sensor] %s = <missing>\n", name)
			} else {
				fmt.Printf("[text_sensor] %s = %s\n", name, m.State)
			}
		default:
			// Print generic info for other state types
			fmt.Printf("[state] type=%T\n", msg)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to states: %w", err)
	}

	<-ctx.Done()
	fmt.Println("\nStopping...")
	return nil
}
