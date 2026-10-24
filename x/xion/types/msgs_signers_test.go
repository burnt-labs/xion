package types_test

import (
	"testing"

	"github.com/btcsuite/btcd/btcutil/bech32"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

// incompleteGroupAuthority is a well-formed bech32 string whose single 5-bit
// data group cannot be regrouped into bytes, so decoding succeeds while the
// base-256 conversion fails.
func incompleteGroupAuthority(t *testing.T) string {
	t.Helper()
	authority, err := bech32.Encode("xion", []byte{1})
	require.NoError(t, err)

	_, data, err := bech32.Decode(authority)
	require.NoError(t, err)
	_, err = bech32.ConvertBits(data, 5, 8, false)
	require.Error(t, err)

	return authority
}

func TestPlatformMsgsGetSignersFallBackOnConvertBitsError(t *testing.T) {
	authority := incompleteGroupAuthority(t)

	t.Run("set platform percentage", func(t *testing.T) {
		msg := &types.MsgSetPlatformPercentage{Authority: authority, PlatformPercentage: 1}
		signers := msg.GetSigners()
		require.Len(t, signers, 1)
		require.Empty(t, signers[0])
	})

	t.Run("set platform minimum", func(t *testing.T) {
		msg := &types.MsgSetPlatformMinimum{Authority: authority, Minimums: sdk.NewCoins(sdk.NewInt64Coin("uxion", 1))}
		signers := msg.GetSigners()
		require.Len(t, signers, 1)
		require.Empty(t, signers[0])
	})
}
