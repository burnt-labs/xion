package app

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

// Message type URLs of every proposal stored on xion-mainnet-1 at v31; gov queries and export must decode them.
var mainnetProposalTypeURLs = []string{
	"/abstractaccount.v1.MsgUpdateParams",
	"/cosmos.gov.v1.MsgExecLegacyContent",
	"/cosmos.gov.v1.MsgUpdateParams",
	"/cosmos.mint.v1beta1.MsgUpdateParams",
	"/cosmos.params.v1beta1.ParameterChangeProposal",
	"/cosmos.slashing.v1beta1.MsgUpdateParams",
	"/cosmos.staking.v1beta1.MsgUpdateParams",
	"/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
	"/cosmwasm.wasm.v1.MsgPinCodes",
	"/cosmwasm.wasm.v1.MsgStoreCode",
	"/cosmwasm.wasm.v1.MsgUpdateParams",
	"/ibc.core.client.v1.MsgRecoverClient",
	"/osmosis.tokenfactory.v1beta1.MsgUpdateParams",
	"/xion.feeabs.v1beta1.MsgAddHostZone",
	"/xion.feeabs.v1beta1.MsgRemoveHostZone",
	"/xion.feeabs.v1beta1.MsgUpdateParams",
	"/xion.jwk.v1.MsgUpdateAudience",
	"/xion.mint.v1.MsgUpdateParams",
	"/xion.v1.MsgSetPlatformMinimum",
}

func TestMainnetProposalTypesResolve(t *testing.T) {
	app := Setup(t)
	for _, url := range mainnetProposalTypeURLs {
		_, err := app.InterfaceRegistry().Resolve(url)
		require.NoError(t, err, url)
	}
}

func TestLegacyParameterChangeProposalDecodes(t *testing.T) {
	app := Setup(t)

	var change []byte
	change = protowire.AppendTag(change, 1, protowire.BytesType)
	change = protowire.AppendString(change, "globalfee")
	change = protowire.AppendTag(change, 2, protowire.BytesType)
	change = protowire.AppendString(change, "MinimumGasPricesParam")
	change = protowire.AppendTag(change, 3, protowire.BytesType)
	change = protowire.AppendString(change, `[{"denom":"uxion","amount":"0.001"}]`)
	var content []byte
	content = protowire.AppendTag(content, 1, protowire.BytesType)
	content = protowire.AppendString(content, "set min gas")
	content = protowire.AppendTag(content, 2, protowire.BytesType)
	content = protowire.AppendString(content, "desc")
	content = protowire.AppendTag(content, 3, protowire.BytesType)
	content = protowire.AppendBytes(content, change)

	exec := &govv1.MsgExecLegacyContent{
		Content:   &codectypes.Any{TypeUrl: "/cosmos.params.v1beta1.ParameterChangeProposal", Value: content},
		Authority: "xion10d07y265gmmuvt4z0w9aw880jnsr700jctf8qc",
	}
	execAny, err := codectypes.NewAnyWithValue(exec)
	require.NoError(t, err)
	bz, err := app.AppCodec().Marshal(&govv1.Proposal{Id: 4, Messages: []*codectypes.Any{execAny}})
	require.NoError(t, err)

	var decoded govv1.Proposal
	require.NoError(t, app.AppCodec().Unmarshal(bz, &decoded))
	msgs, err := decoded.GetMsgs()
	require.NoError(t, err)
	got, ok := msgs[0].(*govv1.MsgExecLegacyContent)
	require.True(t, ok)
	c, ok := got.Content.GetCachedValue().(govv1beta1.Content)
	require.True(t, ok)
	require.Equal(t, "set min gas", c.GetTitle())
}
