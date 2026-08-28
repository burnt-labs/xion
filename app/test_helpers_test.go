package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// The random helpers hand every caller a fresh identity; a fixed return would
// silently make tests share accounts, so distinctness is the invariant worth
// pinning.

func TestRandomAccAddress(t *testing.T) {
	addr := RandomAccAddress()
	require.NoError(t, sdk.VerifyAddressFormat(addr))
	require.NotEqual(t, addr, RandomAccAddress())
}

func TestRandomPubKey(t *testing.T) {
	pubKey := RandomPubKey()
	require.IsType(t, &secp256k1.PubKey{}, pubKey)
	require.Len(t, pubKey.Bytes(), secp256k1.PubKeySize)
	require.NoError(t, sdk.VerifyAddressFormat(pubKey.Address().Bytes()))
	require.NotEqual(t, pubKey.Bytes(), RandomPubKey().Bytes())
}
