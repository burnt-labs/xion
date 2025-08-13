package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestDefaultParams(t *testing.T) {
	p := DefaultParams()
	require.EqualValues(t, p.MinimumGasPrices, sdk.DecCoins{})
	require.EqualValues(t, p.BypassMinFeeMsgTypes, DefaultBypassMinFeeMsgTypes)
	require.EqualValues(t, p.MaxTotalBypassMinFeeMsgGasUsage, DefaultmaxTotalBypassMinFeeMsgGasUsage)
}

func Test_validateMinGasPrices(t *testing.T) {
	tests := map[string]struct {
		coins     interface{}
		expectErr bool
	}{
		"DefaultParams, pass": {
			DefaultParams().MinimumGasPrices,
			false,
		},
		"DecCoins conversion fails, fail": {
			sdk.Coins{sdk.NewCoin("photon", math.OneInt())},
			true,
		},
		"coins amounts are zero, pass": {
			sdk.DecCoins{
				sdk.NewDecCoin("atom", math.ZeroInt()),
				sdk.NewDecCoin("photon", math.ZeroInt()),
			},
			false,
		},
		"duplicate coins denoms, fail": {
			sdk.DecCoins{
				sdk.NewDecCoin("photon", math.OneInt()),
				sdk.NewDecCoin("photon", math.OneInt()),
			},
			true,
		},
		"coins are not sorted by denom alphabetically, fail": {
			sdk.DecCoins{
				sdk.NewDecCoin("photon", math.OneInt()),
				sdk.NewDecCoin("atom", math.OneInt()),
			},
			true,
		},
		"negative amount, fail": {
			sdk.DecCoins{
				sdk.DecCoin{Denom: "photon", Amount: math.LegacyOneDec().Neg()},
			},
			true,
		},
		"invalid denom, fail": {
			sdk.DecCoins{
				sdk.DecCoin{Denom: "photon!", Amount: math.LegacyOneDec().Neg()},
			},
			true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateMinimumGasPrices(test.coins)
			if test.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func Test_validateBypassMinFeeMsgTypes(t *testing.T) {
	tests := map[string]struct {
		msgTypes  interface{}
		expectErr bool
	}{
		"DefaultParams, pass": {
			DefaultParams().BypassMinFeeMsgTypes,
			false,
		},
		"wrong msg type should make conversion fail, fail": {
			[]int{0, 1, 2, 3},
			true,
		},
		"empty msg types, pass": {
			[]string{},
			false,
		},
		"empty msg type, fail": {
			[]string{""},
			true,
		},
		"invalid msg type name, fail": {
			[]string{"ibc.core.channel.v1.MsgRecvPacket"},
			true,
		},
		"mixed valid and invalid msgs, fail": {
			[]string{
				"/ibc.core.channel.v1.MsgRecvPacket",
				"",
			},
			true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateBypassMinFeeMsgTypes(test.msgTypes)
			if test.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func Test_validateMaxTotalBypassMinFeeMsgGasUsage(t *testing.T) {
	tests := map[string]struct {
		msgTypes  interface{}
		expectErr bool
	}{
		"DefaultParams, pass": {
			DefaultParams().MaxTotalBypassMinFeeMsgGasUsage,
			false,
		},
		"zero value, pass": {
			uint64(0),
			false,
		},
		"negative value, fail": {
			-1,
			true,
		},
		"invalid type, fail": {
			"5",
			true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateMaxTotalBypassMinFeeMsgGasUsage(test.msgTypes)
			if test.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestParamSetPairs(t *testing.T) {
	params := DefaultParams()
	pairs := params.ParamSetPairs()
	require.Len(t, pairs, 3)

	// Check each param set pair
	expectedKeys := [][]byte{
		ParamStoreKeyMinGasPrices,
		ParamStoreKeyBypassMinFeeMsgTypes,
		ParamStoreKeyMaxTotalBypassMinFeeMsgGasUsage,
	}

	for i, pair := range pairs {
		require.Equal(t, expectedKeys[i], pair.Key)
		require.NotNil(t, pair.Value)
		require.NotNil(t, pair.ValidatorFn)
	}
}

func TestParamKeyTable(t *testing.T) {
	table := ParamKeyTable()
	require.NotNil(t, table)

	// Test that the table is properly constructed
	require.NotNil(t, table)
}

func TestValidateBasic(t *testing.T) {
	tests := map[string]struct {
		params    Params
		expectErr bool
	}{
		"default params, pass": {
			DefaultParams(),
			false,
		},
		"invalid minimum gas prices, fail": {
			Params{
				MinimumGasPrices: sdk.DecCoins{
					sdk.NewDecCoin("photon", math.OneInt()),
					sdk.NewDecCoin("atom", math.OneInt()), // Not sorted
				},
				BypassMinFeeMsgTypes:            DefaultBypassMinFeeMsgTypes,
				MaxTotalBypassMinFeeMsgGasUsage: DefaultmaxTotalBypassMinFeeMsgGasUsage,
			},
			true,
		},
		"invalid bypass msg types, fail": {
			Params{
				MinimumGasPrices:                sdk.DecCoins{},
				BypassMinFeeMsgTypes:            []string{"invalid"},
				MaxTotalBypassMinFeeMsgGasUsage: DefaultmaxTotalBypassMinFeeMsgGasUsage,
			},
			true,
		},
		"empty bypass msg types, pass": {
			Params{
				MinimumGasPrices:                sdk.DecCoins{},
				BypassMinFeeMsgTypes:            []string{},
				MaxTotalBypassMinFeeMsgGasUsage: DefaultmaxTotalBypassMinFeeMsgGasUsage,
			},
			false,
		},
		"valid non-default params, pass": {
			Params{
				MinimumGasPrices: sdk.DecCoins{
					sdk.NewDecCoin("atom", math.NewInt(1000)),
					sdk.NewDecCoin("stake", math.NewInt(2000)),
				},
				BypassMinFeeMsgTypes:            []string{"/cosmos.bank.v1beta1.MsgSend"},
				MaxTotalBypassMinFeeMsgGasUsage: 500_000,
			},
			false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := test.params.ValidateBasic()
			if test.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidate(t *testing.T) {
	// Test the standalone ValidateBasic function
	params := DefaultParams()
	err := params.ValidateBasic()
	require.NoError(t, err)

	// Test with invalid params
	invalidParams := Params{
		MinimumGasPrices: sdk.DecCoins{
			sdk.NewDecCoin("", math.OneInt()), // Empty denom
		},
		BypassMinFeeMsgTypes:            DefaultBypassMinFeeMsgTypes,
		MaxTotalBypassMinFeeMsgGasUsage: DefaultmaxTotalBypassMinFeeMsgGasUsage,
	}
	err = invalidParams.ValidateBasic()
	require.Error(t, err)
}
