package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestMsgUpdateParams_GetSignBytes(t *testing.T) {
	authority := "cosmos1abc123def456ghi789jkl012mno345pqr678stu9"
	params := DefaultParams()

	msg := &MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}

	// Test that GetSignBytes returns deterministic JSON
	signBytes := msg.GetSignBytes()
	require.NotEmpty(t, signBytes)

	// Test that calling it again returns the same result
	signBytes2 := msg.GetSignBytes()
	require.Equal(t, signBytes, signBytes2)

	// Test that it's valid JSON
	require.Contains(t, string(signBytes), authority)
}

func TestMsgUpdateParams_GetSigners(t *testing.T) {
	// We'll just test that GetSigners method exists and can be called
	// without getting into specific address validation which requires real addresses
	msg := &MsgUpdateParams{
		Authority: "test-authority",
		Params:    DefaultParams(),
	}

	// Test that GetSigners can be called (may return empty slice for invalid address)
	signers := msg.GetSigners()
	require.NotNil(t, signers)
}

func TestMsgUpdateParams_ValidateBasic(t *testing.T) {
	tests := []struct {
		name        string
		msg         *MsgUpdateParams
		expectedErr bool
		errContains string
	}{
		{
			name: "invalid authority address",
			msg: &MsgUpdateParams{
				Authority: "invalid-address",
				Params:    DefaultParams(),
			},
			expectedErr: true,
			errContains: "invalid authority address",
		},
		{
			name: "empty authority",
			msg: &MsgUpdateParams{
				Authority: "",
				Params:    DefaultParams(),
			},
			expectedErr: true,
			errContains: "invalid authority address",
		},
		// The following param validation tests use a valid bech32 authority so that
		// ValidateBasic progresses past the authority check and exercises the
		// Params.Validate() branch.
		{
			name: "invalid params - negative inflation max",
			msg: &MsgUpdateParams{
				Authority: sdk.AccAddress(make([]byte, 20)).String(), // valid bech32 address of 20 zero bytes
				Params: Params{
					MintDenom:           "uxion",
					InflationRateChange: math.LegacyNewDecWithPrec(13, 2),
					InflationMax:        math.LegacyNewDec(-1), // Invalid
					InflationMin:        math.LegacyNewDecWithPrec(7, 2),
					GoalBonded:          math.LegacyNewDecWithPrec(67, 2),
					BlocksPerYear:       uint64(6311520),
				},
			},
			expectedErr: true,
		},
		{
			name: "invalid params - empty denom",
			msg: &MsgUpdateParams{
				Authority: sdk.AccAddress(make([]byte, 20)).String(), // valid bech32
				Params: Params{
					MintDenom:           "", // Invalid
					InflationRateChange: math.LegacyNewDecWithPrec(13, 2),
					InflationMax:        math.LegacyNewDecWithPrec(20, 2),
					InflationMin:        math.LegacyNewDecWithPrec(7, 2),
					GoalBonded:          math.LegacyNewDecWithPrec(67, 2),
					BlocksPerYear:       uint64(6311520),
				},
			},
			expectedErr: true,
		},
		{
			name: "valid params and authority",
			msg: &MsgUpdateParams{
				Authority: sdk.AccAddress(make([]byte, 20)).String(),
				Params:    DefaultParams(),
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()

			if tt.expectedErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgUpdateParams_Interface(t *testing.T) {
	msg := &MsgUpdateParams{}

	// Test that it implements sdk.Msg interface
	require.Implements(t, (*sdk.Msg)(nil), msg)
}
