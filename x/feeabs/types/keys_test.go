package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeys(t *testing.T) {
	tests := []struct {
		name     string
		function func() []byte
		input    string
		expected string
	}{
		{
			name: "GetKeyHostZoneConfigByFeeabsIBCDenom",
			function: func() []byte {
				return GetKeyHostZoneConfigByFeeabsIBCDenom("test-denom")
			},
			expected: string(append(KeyHostChainConfigByFeeAbs, []byte("test-denom")...)),
		},
		{
			name: "GetKeyHostZoneConfigByOsmosisIBCDenom",
			function: func() []byte {
				return GetKeyHostZoneConfigByOsmosisIBCDenom("osmosis-denom")
			},
			expected: string(append(KeyHostChainConfigByOsmosis, []byte("osmosis-denom")...)),
		},
		{
			name: "GetKeyTwapExchangeRate",
			function: func() []byte {
				return GetKeyTwapExchangeRate("ibc-denom")
			},
			expected: string(append(OsmosisTwapExchangeRate, []byte("ibc-denom")...)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function()
			require.Equal(t, tt.expected, string(result))
		})
	}
}

func TestConstants(t *testing.T) {
	// Test module constants
	require.Equal(t, "feeabs", ModuleName)
	require.Equal(t, ModuleName, StoreKey)
	require.Equal(t, ModuleName, RouterKey)
	require.Equal(t, ModuleName, QuerierRoute)
	require.Equal(t, "mem_feeabs", MemStoreKey)
	require.Equal(t, "|", KeySeparator)
}

func TestStoreKeys(t *testing.T) {
	// Test that all store keys are unique
	keys := [][]byte{
		StoreExponentialBackoff,
		OsmosisTwapExchangeRate,
		KeyChannelID,
		KeyHostChainConfigByFeeAbs,
		KeyHostChainConfigByOsmosis,
		KeyPrefixEpoch,
		KeyTokenDenomPair,
	}

	// Verify all keys are different
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			require.NotEqual(t, string(keys[i]), string(keys[j]),
				"Store keys should be unique, but found duplicate at indices %d and %d", i, j)
		}
	}

	// Test specific key values
	require.Equal(t, []byte{0x10}, StoreExponentialBackoff)
	require.Equal(t, []byte{0x01}, OsmosisTwapExchangeRate)
	require.Equal(t, []byte{0x02}, KeyChannelID)
	require.Equal(t, []byte{0x03}, KeyHostChainConfigByFeeAbs)
	require.Equal(t, []byte{0x04}, KeyHostChainConfigByOsmosis)
	require.Equal(t, []byte{0x05}, KeyPrefixEpoch)
	require.Equal(t, []byte{0x06}, KeyTokenDenomPair)
}

func TestKeyTypes(t *testing.T) {
	// Test that key types are defined
	var _ ByPassMsgKey
	var _ ByPassExceedMaxGasUsageKey
	var _ GlobalFeeKey
}
