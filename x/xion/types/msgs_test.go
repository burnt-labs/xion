package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

func TestMsgSend(t *testing.T) {
	fromAddr := sdk.AccAddress("from_address_12345678")
	toAddr := sdk.AccAddress("to_address_12345678901")
	amount := sdk.NewCoins(sdk.NewInt64Coin("atom", 100))

	// Test NewMsgSend
	msg := types.NewMsgSend(fromAddr, toAddr, amount)
	require.NotNil(t, msg)
	require.Equal(t, fromAddr.String(), msg.FromAddress)
	require.Equal(t, toAddr.String(), msg.ToAddress)
	require.Equal(t, amount, msg.Amount)

	// Test Route
	require.Equal(t, types.RouterKey, msg.Route())

	// Test Type
	require.Equal(t, types.TypeMsgSend, msg.Type())

	// Test ValidateBasic - valid message
	require.NoError(t, msg.ValidateBasic())

	// Test GetSigners
	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, fromAddr, signers[0])

	// Test GetSignBytes
	signBytes := msg.GetSignBytes()
	require.NotNil(t, signBytes)
	require.True(t, len(signBytes) > 0)
}

func TestMsgSend_ValidateBasic(t *testing.T) {
	validFromAddr := sdk.AccAddress("from_address_12345678")
	validToAddr := sdk.AccAddress("to_address_12345678901")
	validAmount := sdk.NewCoins(sdk.NewInt64Coin("atom", 100))

	tests := []struct {
		name    string
		msg     *types.MsgSend
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid message",
			msg:     types.NewMsgSend(validFromAddr, validToAddr, validAmount),
			wantErr: false,
		},
		{
			name: "invalid from address",
			msg: &types.MsgSend{
				FromAddress: "invalid_address",
				ToAddress:   validToAddr.String(),
				Amount:      validAmount,
			},
			wantErr: true,
			errMsg:  "invalid from address",
		},
		{
			name: "invalid to address",
			msg: &types.MsgSend{
				FromAddress: validFromAddr.String(),
				ToAddress:   "invalid_address",
				Amount:      validAmount,
			},
			wantErr: true,
			errMsg:  "invalid to address",
		},
		{
			name: "invalid coins - negative amount",
			msg: &types.MsgSend{
				FromAddress: validFromAddr.String(),
				ToAddress:   validToAddr.String(),
				Amount:      sdk.Coins{sdk.Coin{Denom: "atom", Amount: math.NewInt(-100)}},
			},
			wantErr: true,
			errMsg:  "invalid coins",
		},
		{
			name: "invalid coins - zero amount",
			msg: &types.MsgSend{
				FromAddress: validFromAddr.String(),
				ToAddress:   validToAddr.String(),
				Amount:      sdk.NewCoins(sdk.NewInt64Coin("atom", 0)),
			},
			wantErr: true,
			errMsg:  "invalid coins",
		},
		{
			name: "empty amount",
			msg: &types.MsgSend{
				FromAddress: validFromAddr.String(),
				ToAddress:   validToAddr.String(),
				Amount:      sdk.Coins{},
			},
			wantErr: true,
			errMsg:  "invalid coins",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgMultiSend(t *testing.T) {
	addr1 := sdk.AccAddress("addr1_12345678901234567890")
	addr2 := sdk.AccAddress("addr2_12345678901234567890")
	addr3 := sdk.AccAddress("addr3_12345678901234567890")

	input := banktypes.Input{
		Address: addr1.String(),
		Coins:   sdk.NewCoins(sdk.NewInt64Coin("atom", 200)),
	}
	outputs := []banktypes.Output{
		{
			Address: addr2.String(),
			Coins:   sdk.NewCoins(sdk.NewInt64Coin("atom", 100)),
		},
		{
			Address: addr3.String(),
			Coins:   sdk.NewCoins(sdk.NewInt64Coin("atom", 100)),
		},
	}

	// Test NewMsgMultiSend
	msg := types.NewMsgMultiSend([]banktypes.Input{input}, outputs)
	require.NotNil(t, msg)
	require.Equal(t, []banktypes.Input{input}, msg.Inputs)
	require.Equal(t, outputs, msg.Outputs)

	// Test Route
	require.Equal(t, types.RouterKey, msg.Route())

	// Test Type
	require.Equal(t, types.TypeMsgMultiSend, msg.Type())

	// Test ValidateBasic - valid message
	require.NoError(t, msg.ValidateBasic())

	// Test GetSigners
	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, addr1, signers[0])

	// Test GetSignBytes
	signBytes := msg.GetSignBytes()
	require.NotNil(t, signBytes)
	require.True(t, len(signBytes) > 0)
}

func TestMsgMultiSend_ValidateBasic(t *testing.T) {
	addr1 := sdk.AccAddress("addr1_12345678901234567890")
	addr2 := sdk.AccAddress("addr2_12345678901234567890")

	validInput := banktypes.Input{
		Address: addr1.String(),
		Coins:   sdk.NewCoins(sdk.NewInt64Coin("atom", 100)),
	}
	validOutput := banktypes.Output{
		Address: addr2.String(),
		Coins:   sdk.NewCoins(sdk.NewInt64Coin("atom", 100)),
	}

	tests := []struct {
		name    string
		msg     *types.MsgMultiSend
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid message",
			msg:     types.NewMsgMultiSend([]banktypes.Input{validInput}, []banktypes.Output{validOutput}),
			wantErr: false,
		},
		{
			name: "no inputs",
			msg: &types.MsgMultiSend{
				Inputs:  []banktypes.Input{},
				Outputs: []banktypes.Output{validOutput},
			},
			wantErr: true,
			errMsg:  "no inputs to send transaction",
		},
		{
			name: "multiple inputs",
			msg: &types.MsgMultiSend{
				Inputs:  []banktypes.Input{validInput, validInput},
				Outputs: []banktypes.Output{validOutput},
			},
			wantErr: true,
			errMsg:  "multiple senders not allowed",
		},
		{
			name: "no outputs",
			msg: &types.MsgMultiSend{
				Inputs:  []banktypes.Input{validInput},
				Outputs: []banktypes.Output{},
			},
			wantErr: true,
			errMsg:  "no outputs to send transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgSetPlatformPercentage(t *testing.T) {
	percentage := uint32(2500) // 25%

	// Test NewMsgSetPlatformPercentage
	msg := types.NewMsgSetPlatformPercentage(percentage)
	require.NotNil(t, msg)
	require.Equal(t, percentage, msg.PlatformPercentage)

	// Test Route
	require.Equal(t, types.RouterKey, msg.Route())

	// Test Type
	require.Equal(t, types.TypeMsgSetPlatformPercentage, msg.Type())

	// Test ValidateBasic - valid message
	require.NoError(t, msg.ValidateBasic())

	// Test GetSignBytes
	signBytes := msg.GetSignBytes()
	require.NotNil(t, signBytes)
	require.True(t, len(signBytes) > 0)
}

func TestMsgSetPlatformPercentage_ValidateBasic(t *testing.T) {
	tests := []struct {
		name       string
		percentage uint32
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid percentage - 0%",
			percentage: 0,
			wantErr:    false,
		},
		{
			name:       "valid percentage - 50%",
			percentage: 5000,
			wantErr:    false,
		},
		{
			name:       "valid percentage - 100%",
			percentage: 10000,
			wantErr:    false,
		},
		{
			name:       "invalid percentage - over 100%",
			percentage: 10001,
			wantErr:    true,
			errMsg:     "unable to have a platform percentage that exceeds 100%",
		},
		{
			name:       "invalid percentage - way over 100%",
			percentage: 99999,
			wantErr:    true,
			errMsg:     "unable to have a platform percentage that exceeds 100%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := types.NewMsgSetPlatformPercentage(tt.percentage)
			err := msg.ValidateBasic()

			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgSetPlatformPercentage_GetSigners(t *testing.T) {
	addr := sdk.AccAddress("authority_12345678901234567890")
	msg := &types.MsgSetPlatformPercentage{
		Authority:          addr.String(),
		PlatformPercentage: 5000,
	}

	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, addr, signers[0])
}
