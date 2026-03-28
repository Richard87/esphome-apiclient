package esphome_apiclient

import (
	"fmt"

	"github.com/richard87/esphome-apiclient/pb"
	"google.golang.org/protobuf/proto"
)

// SubscribeLogs subscribes to device log messages at or above the given level.
// The handler is called for every incoming log message.
// Returns an unsubscribe function.
func (c *Client) SubscribeLogs(level pb.LogLevel, handler func(msg *pb.SubscribeLogsResponse)) (unsubscribe func(), err error) {
	// SubscribeLogsResponse has message type ID 29
	remove := c.On(29, func(msg proto.Message) {
		logMsg, ok := msg.(*pb.SubscribeLogsResponse)
		if !ok {
			return
		}
		if handler != nil {
			handler(logMsg)
		}
	})

	unsubscribe = func() {
		remove()
	}

	// SubscribeLogsRequest has message type ID 28
	req := &pb.SubscribeLogsRequest{
		Level:      level,
		DumpConfig: true,
	}
	if err := c.SendMessage(req, 28); err != nil {
		unsubscribe()
		return nil, fmt.Errorf("SubscribeLogs: failed to send request: %w", err)
	}

	return unsubscribe, nil
}
