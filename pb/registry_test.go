package pb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestMessageRegistryRoundTrip(t *testing.T) {
	for id, factory := range MessageRegistry {
		msg := factory()
		assert.NotNil(t, msg, "Factory for ID %d returned nil", id)

		data, err := proto.Marshal(msg)
		assert.NoError(t, err, "Failed to marshal message ID %d", id)

		newMsg := factory()
		err = proto.Unmarshal(data, newMsg)
		assert.NoError(t, err, "Failed to unmarshal message ID %d", id)

		assert.True(t, proto.Equal(msg, newMsg), "Message ID %d round-trip failed", id)
	}
}
