package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEventConstants(t *testing.T) {
	// Test all event type constants
	require.Equal(t, "timeout", EventTypeTimeout)
	require.Equal(t, "receive_feechain_verification_packet", EventTypePacket)
	require.Equal(t, "epoch_end", EventTypeEpochEnd)
	require.Equal(t, "epoch_start", EventTypeEpochStart)

	// Test all attribute key constants
	require.Equal(t, "success", AttributeKeyAckSuccess)
	require.Equal(t, "acknowledgement", AttributeKeyAck)
	require.Equal(t, "ack_error", AttributeKeyAckError)
	require.Equal(t, "epoch_number", AttributeEpochNumber)
	require.Equal(t, "start_time", AttributeEpochStartTime)
	require.Equal(t, "failure_type", AttributeKeyFailureType)
	require.Equal(t, "packet", AttributeKeyPacket)
}

func TestEventConstantsUniqueness(t *testing.T) {
	// Collect all event constants to ensure they're unique
	eventTypes := []string{
		EventTypeTimeout,
		EventTypePacket,
		EventTypeEpochEnd,
		EventTypeEpochStart,
	}

	attributeKeys := []string{
		AttributeKeyAckSuccess,
		AttributeKeyAck,
		AttributeKeyAckError,
		AttributeEpochNumber,
		AttributeEpochStartTime,
		AttributeKeyFailureType,
		AttributeKeyPacket,
	}

	// Check that event types are unique
	for i := 0; i < len(eventTypes); i++ {
		for j := i + 1; j < len(eventTypes); j++ {
			require.NotEqual(t, eventTypes[i], eventTypes[j],
				"Event types should be unique")
		}
	}

	// Check that attribute keys are unique
	for i := 0; i < len(attributeKeys); i++ {
		for j := i + 1; j < len(attributeKeys); j++ {
			require.NotEqual(t, attributeKeys[i], attributeKeys[j],
				"Attribute keys should be unique")
		}
	}
}
