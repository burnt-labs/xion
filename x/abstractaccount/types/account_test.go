package types_test

import (
	types "github.com/burnt-labs/xion/x/abstractaccount/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAmino(t *testing.T) {
	aminoCodec := codec.NewLegacyAmino()

	// register crypto and aa amino codecs
	std.RegisterLegacyAminoCodec(aminoCodec)
	types.RegisterLegacyAminoCodec(aminoCodec)

	testAccount := types.NewAbstractAccount("test-string", 1, 0)
	pubkey := testAccount.GetPubKey()

	// recreate what ConsumeTxSizeGasDecorator does
	simSecp256k1Sig := [64]byte{}
	simSig := legacytx.StdSignature{ //nolint:staticcheck // SA1019: legacytx.StdSignature is deprecated
		Signature: simSecp256k1Sig[:],
		PubKey:    pubkey,
	}

	_, err := aminoCodec.Marshal(simSig)
	require.NoError(t, err)
}
