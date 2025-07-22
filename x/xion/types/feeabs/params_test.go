package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ParamString(t *testing.T) {
	tests := []struct {
		name     string
		params   Params
		expected string
	}{
		{
			name:     "default true",
			params:   DefaultParams(),
			expected: "native_ibced_in_osmosis:\"ibc/C053D637CCA2A2BA030E2C5EE1B28A16F71CCB0E45E8BE52766DC1B241B77878\" osmosis_query_twap_path:\"/osmosis.twap.v1beta1.Query/ArithmeticTwapToNow\" chain_name:\"feeappd-t1\" ",
		},
		{
			name: "all filled",
			params: Params{
				OsmosisQueryTwapPath:         DefaultOsmosisQueryTwapPath,
				NativeIbcedInOsmosis:         "ibc/123abc456",
				ChainName:                    "feeapp-1",
				IbcTransferChannel:           "channel-0",
				IbcQueryIcqChannel:           "channel-3",
				OsmosisCrosschainSwapAddress: "osmo1abc123",
			},
			expected: "native_ibced_in_osmosis:\"ibc/123abc456\" osmosis_query_twap_path:\"/osmosis.twap.v1beta1.Query/ArithmeticTwapToNow\" chain_name:\"feeapp-1\" ibc_transfer_channel:\"channel-0\" ibc_query_icq_channel:\"channel-3\" osmosis_crosschain_swap_address:\"osmo1abc123\" ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(tt *testing.T) {
			actual := tc.params.String()
			assert.Equal(tt, tc.expected, actual)
		})
	}
}
