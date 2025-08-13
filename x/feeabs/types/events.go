package types

const (
	EventTypeTimeout        = "timeout"
	EventTypePacket         = "receive_feechain_verification_packet"
	EventTypeEpochEnd       = "epoch_end" // TODO: need to clean up (not use)
	EventTypeEpochStart     = "epoch_start"
	AttributeKeyAckSuccess  = "success"
	AttributeKeyAck         = "acknowledgement"
	AttributeKeyAckError    = "ack_error"
	AttributeEpochNumber    = "epoch_number"
	AttributeEpochStartTime = "start_time"
	AttributeKeyFailureType = "failure_type"
	AttributeKeyPacket      = "packet"
)
