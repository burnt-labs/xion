package ibcwasm_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/burnt-labs/xion/app/legacy/ibcwasm"
)

// Raw values read out of the xion-testnet-2 ibc store on 2026-09-04, at
// clients/08-wasm-15/clientState and clients/08-wasm-15/consensusStates/0-100.
// Each is a marshalled Any, exactly the shape 02-client hands to
// UnmarshalClientState and UnmarshalConsensusState. Both wrap a Parlia light
// client, which is what 08-wasm was carrying on that chain.
const (
	testnet2ClientStateHex = "0a252f6962632e6c69676874636c69656e74732e7761736d2e76312e436c69656e745374617465129c010a730a272f696263" +
		"2e6c69676874636c69656e74732e7061726c69612e76312e436c69656e7453746174651248088f4e1214aa43d337145e8930" +
		"d01cb4e60abf6595c692921e1a201ee222554989dda120e26ecacf756fe1235cd8d726706b57517715dde4f0c900220310c4" +
		"012a040880a305320012203aa082e78af6e8b2fd3b91b4c1059682651bf9466838aaa6b959de975d6ea1f11a0310c401"

	testnet2ConsensusStateHex = "0a282f6962632e6c69676874636c69656e74732e7761736d2e76312e436f6e73656e7375735374617465129d010a9a010a2a" +
		"2f6962632e6c69676874636c69656e74732e7061726c69612e76312e436f6e73656e7375735374617465126c0a20491a1030" +
		"a826f55e2d9332b8b68c845e0ac399f4c3d5de24e31790233ed0134510e5cdcfbf061a20030b2218fe0947d10e4a75e88298" +
		"dd73223c5cc403775815324fd48e9c13634322205ee9ddd34d3c14124f32c4881d43c6ee803190c5f57ef85ac1531d43fa7a" +
		"9d7c"

	testnet2Checksum = "3aa082e78af6e8b2fd3b91b4c1059682651bf9466838aaa6b959de975d6ea1f1"
)

// newCodec builds the codec the way the app does for these types: core IBC
// registers the client interfaces, and the shim registers the 08-wasm
// implementations of them.
func newCodec(t *testing.T) codec.BinaryCodec {
	t.Helper()

	registry := codectypes.NewInterfaceRegistry()
	clienttypes.RegisterInterfaces(registry)
	ibcwasm.RegisterInterfaces(registry)

	return codec.NewProtoCodec(registry)
}

func mustDecodeHex(t *testing.T, s string) []byte {
	t.Helper()

	bz, err := hex.DecodeString(s)
	require.NoError(t, err)

	return bz
}

// TestUnmarshalTestnet2ClientState is the case that motivates this package: a
// real record written by the removed module must still decode.
func TestUnmarshalTestnet2ClientState(t *testing.T) {
	cdc := newCodec(t)

	clientState, err := clienttypes.UnmarshalClientState(cdc, mustDecodeHex(t, testnet2ClientStateHex))
	require.NoError(t, err)
	require.Equal(t, ibcwasm.Wasm, clientState.ClientType())
	require.NoError(t, clientState.Validate())

	wasmClientState, ok := clientState.(*ibcwasm.ClientState)
	require.True(t, ok, "expected the concrete 08-wasm client state, got %T", clientState)
	require.Len(t, wasmClientState.Data, 115)
	require.Equal(t, testnet2Checksum, hex.EncodeToString(wasmClientState.Checksum))
	require.Equal(t, clienttypes.NewHeight(0, 196), wasmClientState.LatestHeight)
}

func TestUnmarshalTestnet2ConsensusState(t *testing.T) {
	cdc := newCodec(t)

	consensusState, err := clienttypes.UnmarshalConsensusState(cdc, mustDecodeHex(t, testnet2ConsensusStateHex))
	require.NoError(t, err)
	require.Equal(t, ibcwasm.Wasm, consensusState.ClientType())
	require.NoError(t, consensusState.ValidateBasic())
	require.Zero(t, consensusState.GetTimestamp())

	wasmConsensusState, ok := consensusState.(*ibcwasm.ConsensusState)
	require.True(t, ok, "expected the concrete 08-wasm consensus state, got %T", consensusState)
	require.Len(t, wasmConsensusState.Data, 154)
}

// TestUnmarshalWithoutRegistrationFails pins the failure this package prevents.
// The registry is built the way the app would be without the shim — core IBC
// plus the Tendermint light client, which is the only one still routed — and the
// same bytes then fail to resolve.
func TestUnmarshalWithoutRegistrationFails(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	clienttypes.RegisterInterfaces(registry)
	ibctm.RegisterInterfaces(registry)

	_, err := clienttypes.UnmarshalClientState(codec.NewProtoCodec(registry), mustDecodeHex(t, testnet2ClientStateHex))
	require.ErrorContains(t, err, "/ibc.lightclients.wasm.v1.ClientState")
}

func TestClientStateRoundTrip(t *testing.T) {
	cdc := newCodec(t)

	original := &ibcwasm.ClientState{
		Data:         []byte("client state payload"),
		Checksum:     mustDecodeHex(t, testnet2Checksum),
		LatestHeight: clienttypes.NewHeight(2, 7),
	}

	bz, err := clienttypes.MarshalClientState(cdc, original)
	require.NoError(t, err)

	decoded, err := clienttypes.UnmarshalClientState(cdc, bz)
	require.NoError(t, err)
	require.Equal(t, original, decoded)
}

func TestClientStateValidate(t *testing.T) {
	checksum := mustDecodeHex(t, testnet2Checksum)

	testCases := []struct {
		name        string
		clientState ibcwasm.ClientState
		expErr      string
	}{
		{
			name:        "valid",
			clientState: ibcwasm.ClientState{Data: []byte("payload"), Checksum: checksum},
		},
		{
			name:        "empty data",
			clientState: ibcwasm.ClientState{Checksum: checksum},
			expErr:      "data cannot be empty",
		},
		{
			name:        "empty checksum",
			clientState: ibcwasm.ClientState{Data: []byte("payload")},
			expErr:      "expected 32 bytes, got 0",
		},
		{
			name:        "short checksum",
			clientState: ibcwasm.ClientState{Data: []byte("payload"), Checksum: checksum[:16]},
			expErr:      "expected 32 bytes, got 16",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.clientState.Validate()
			if tc.expErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.expErr)
		})
	}
}

func TestConsensusStateValidateBasic(t *testing.T) {
	require.NoError(t, ibcwasm.ConsensusState{Data: []byte("payload")}.ValidateBasic())
	require.ErrorContains(t, ibcwasm.ConsensusState{}.ValidateBasic(), "data cannot be empty")
}

func TestClientMessage(t *testing.T) {
	cdc := newCodec(t)

	original := &ibcwasm.ClientMessage{Data: []byte("client message payload")}
	require.Equal(t, ibcwasm.Wasm, original.ClientType())
	require.NoError(t, original.ValidateBasic())
	require.ErrorContains(t, ibcwasm.ClientMessage{}.ValidateBasic(), "data cannot be empty")

	bz, err := clienttypes.MarshalClientMessage(cdc, original)
	require.NoError(t, err)

	decoded, err := clienttypes.UnmarshalClientMessage(cdc, bz)
	require.NoError(t, err)
	require.Equal(t, original, decoded)
}

// TestClientTypeMatchesIdentifierPrefix guards the constant: 02-client genesis
// validation rejects a client whose state type disagrees with its identifier.
func TestClientTypeMatchesIdentifierPrefix(t *testing.T) {
	clientType, sequence, err := clienttypes.ParseClientIdentifier("08-wasm-15")
	require.NoError(t, err)
	require.Equal(t, uint64(15), sequence)
	require.Equal(t, ibcwasm.ClientState{}.ClientType(), clientType)
	require.Equal(t, ibcwasm.ConsensusState{}.ClientType(), clientType)
	require.Equal(t, ibcwasm.ClientMessage{}.ClientType(), clientType)
}
