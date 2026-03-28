package command

import (
	"context"
	"encoding/hex"
	"fmt"

	esphome "github.com/richard87/esphome-apiclient"
	"github.com/richard87/esphome-apiclient/pb"
	"google.golang.org/protobuf/proto"
)

func RunBluetooth(ctx context.Context, client *esphome.Client) error {
	fmt.Println("Streaming Bluetooth LE advertisements (press Ctrl+C to stop)...")

	unsubscribe, err := client.SubscribeBluetoothAdvertisements(func(msg proto.Message) {
		switch m := msg.(type) {
		case *pb.BluetoothLEAdvertisementResponse:
			fmt.Printf("[%s] RSSI: %d, Name: %s\n", formatAddr(m.Address), m.Rssi, string(m.Name))
		case *pb.BluetoothLERawAdvertisementsResponse:
			for _, adv := range m.Advertisements {
				fmt.Printf("[%s] RSSI: %d\n", formatAddr(adv.Address), adv.Rssi)
			}
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to bluetooth advertisements: %w", err)
	}
	defer unsubscribe()

	<-ctx.Done()
	fmt.Println("\nStopping...")
	return nil
}

func formatAddr(addr uint64) string {
	b := make([]byte, 8)
	for i := 0; i < 8; i++ {
		b[i] = byte(addr >> (56 - i*8))
	}
	// Bluetooth addresses are 6 bytes, usually the last 6 in the uint64
	return hex.EncodeToString(b[2:])
}
