package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/xion/types"
)

func TestGenesisStateProtobufMethods(t *testing.T) {
	genesisState := &types.GenesisState{
		PlatformPercentage: 5000,
	}

	// Test String() method
	str := genesisState.String()
	require.NotEmpty(t, str)
	require.Contains(t, str, "5000")

	// Test Reset() method
	require.NotPanics(t, func() {
		genesisState.Reset()
	})

	// Test ProtoMessage() method
	require.NotPanics(t, func() {
		genesisState.ProtoMessage()
	})

	// Test Descriptor() method
	desc, indices := genesisState.Descriptor()
	require.NotNil(t, desc)
	require.NotNil(t, indices)

	// Test GetPlatformPercentage getter
	genesisState.PlatformPercentage = 1000
	require.Equal(t, uint32(1000), genesisState.GetPlatformPercentage())

	// Test GetPlatformMinimums getter
	minimums := genesisState.GetPlatformMinimums()
	_ = minimums // May be nil, but shouldn't panic
}

func TestMsgSendProtobufMethods(t *testing.T) {
	msg := &types.MsgSend{
		FromAddress: "cosmos1test",
		ToAddress:   "cosmos1test2",
	}

	// Test String() method
	str := msg.String()
	require.NotEmpty(t, str)

	// Test Reset() method
	require.NotPanics(t, func() {
		msg.Reset()
	})

	// Test ProtoMessage() method
	require.NotPanics(t, func() {
		msg.ProtoMessage()
	})

	// Test Descriptor() method
	desc, indices := msg.Descriptor()
	require.NotNil(t, desc)
	require.NotNil(t, indices)
}

func TestMsgMultiSendProtobufMethods(t *testing.T) {
	msg := &types.MsgMultiSend{}

	// Test Reset() method
	require.NotPanics(t, func() {
		msg.Reset()
	})

	// Test ProtoMessage() method
	require.NotPanics(t, func() {
		msg.ProtoMessage()
	})

	// Test Descriptor() method
	desc, indices := msg.Descriptor()
	require.NotNil(t, desc)
	require.NotNil(t, indices)

	// Test getters - may return nil/empty slices
	inputs := msg.GetInputs()
	outputs := msg.GetOutputs()
	_ = inputs  // May be nil
	_ = outputs // May be nil
}

func TestMsgSetPlatformPercentageProtobufMethods(t *testing.T) {
	msg := &types.MsgSetPlatformPercentage{
		Authority:          "cosmos1authority",
		PlatformPercentage: 5000,
	}

	// Test String() method
	str := msg.String()
	require.NotEmpty(t, str)

	// Test Reset() method
	require.NotPanics(t, func() {
		msg.Reset()
	})

	// Test ProtoMessage() method
	require.NotPanics(t, func() {
		msg.ProtoMessage()
	})

	// Test Descriptor() method
	desc, indices := msg.Descriptor()
	require.NotNil(t, desc)
	require.NotNil(t, indices)

	// Test getters
	authority := msg.GetAuthority()
	percentage := msg.GetPlatformPercentage()
	_ = authority  // May be empty
	_ = percentage // May be 0 if not set
}

func TestAuthzAllowanceProtobufMethods(t *testing.T) {
	allowance := &types.AuthzAllowance{
		AuthzGrantee: "cosmos1test",
	}

	// Test String() method
	str := allowance.String()
	require.NotEmpty(t, str)

	// Test Reset() method
	require.NotPanics(t, func() {
		allowance.Reset()
	})

	// Test ProtoMessage() method
	require.NotPanics(t, func() {
		allowance.ProtoMessage()
	})

	// Test Descriptor() method
	desc, indices := allowance.Descriptor()
	require.NotNil(t, desc)
	require.NotNil(t, indices)

	// Test XXX_Size() method
	size := allowance.XXX_Size()
	require.GreaterOrEqual(t, size, 0)
}

func TestContractsAllowanceProtobufMethods(t *testing.T) {
	allowance := &types.ContractsAllowance{
		ContractAddresses: []string{"cosmos1contract"},
	}

	// Test String() method
	str := allowance.String()
	require.NotEmpty(t, str)

	// Test Reset() method
	require.NotPanics(t, func() {
		allowance.Reset()
	})

	// Test ProtoMessage() method
	require.NotPanics(t, func() {
		allowance.ProtoMessage()
	})

	// Test Descriptor() method
	desc, indices := allowance.Descriptor()
	require.NotNil(t, desc)
	require.NotNil(t, indices)

	// Test XXX_Size() method
	size := allowance.XXX_Size()
	require.GreaterOrEqual(t, size, 0)
}

func TestMultiAnyAllowanceProtobufMethods(t *testing.T) {
	allowance := &types.MultiAnyAllowance{}

	// Test Reset() method
	require.NotPanics(t, func() {
		allowance.Reset()
	})

	// Test ProtoMessage() method
	require.NotPanics(t, func() {
		allowance.ProtoMessage()
	})

	// Test Descriptor() method
	desc, indices := allowance.Descriptor()
	require.NotNil(t, desc)
	require.NotNil(t, indices)
}
