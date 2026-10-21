package types_test

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

func registryGenesis(t *testing.T, entries ...*types.AccountAddress) *types.GenesisState {
	t.Helper()
	params, err := types.NewParamsWithAddressDerivationHash(true, nil, types.DefaultMaxGas, types.DefaultMaxGas, bytes.Repeat([]byte{3}, sha256.Size))
	require.NoError(t, err)
	gs := types.NewGenesisState(1, params)
	gs.AccountAddresses = entries
	return gs
}

func TestGenesisStateRejectsMalformedRegistryEntries(t *testing.T) {
	sender := sdk.AccAddress("registry-sender").String()
	address := sdk.AccAddress("registry-account").String()

	for _, tc := range []struct {
		desc   string
		entry  *types.AccountAddress
		expErr string
	}{
		{
			desc:   "nil entry",
			entry:  nil,
			expErr: "account address registry entry cannot be nil",
		},
		{
			desc:   "sender is not bech32",
			entry:  &types.AccountAddress{Sender: "not-bech32", Salt: []byte("salt"), Address: address},
			expErr: "invalid account address registry sender",
		},
		{
			desc:   "address is not bech32",
			entry:  &types.AccountAddress{Sender: sender, Salt: []byte("salt"), Address: "not-bech32"},
			expErr: "invalid account address registry address",
		},
		{
			desc:   "empty salt",
			entry:  &types.AccountAddress{Sender: sender, Salt: nil, Address: address},
			expErr: "invalid account address registry salt",
		},
		{
			desc:   "salt too long",
			entry:  &types.AccountAddress{Sender: sender, Salt: bytes.Repeat([]byte{1}, 65), Address: address},
			expErr: "invalid account address registry salt",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := registryGenesis(t, tc.entry).Validate()
			require.ErrorContains(t, err, tc.expErr)
		})
	}

	require.NoError(t, registryGenesis(t, &types.AccountAddress{Sender: sender, Salt: []byte("salt"), Address: address}).Validate())
}

func TestNewParamsWithAddressDerivationHashRejectsInvalidBaseParams(t *testing.T) {
	params, err := types.NewParamsWithAddressDerivationHash(true, nil, 0, types.DefaultMaxGas, bytes.Repeat([]byte{3}, sha256.Size))
	require.ErrorIs(t, err, types.ErrZeroMaxGas)
	require.Nil(t, params)
}

func TestMsgUpdateParamsValidateBasicRejectsNilParams(t *testing.T) {
	msg := &types.MsgUpdateParams{Sender: sdk.AccAddress("update-params-sender").String()}

	err := msg.ValidateBasic()
	require.ErrorIs(t, err, sdkerrors.ErrInvalidRequest)
	require.ErrorContains(t, err, "params cannot be nil")
}

// unencodableMsg is a message whose protobuf encoding fails.
type unencodableMsg struct{}

func (*unencodableMsg) Reset()                   {}
func (*unencodableMsg) String() string           { return "unencodable" }
func (*unencodableMsg) ProtoMessage()            {}
func (*unencodableMsg) Marshal() ([]byte, error) { return nil, errors.New("cannot encode message") }

func TestNewAnyFromProtoMsgEncodingFailure(t *testing.T) {
	msgAny, err := types.NewAnyFromProtoMsg(&unencodableMsg{})
	require.EqualError(t, err, "cannot encode message")
	require.Nil(t, msgAny)
}
