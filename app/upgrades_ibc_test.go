package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	connectiontypes "github.com/cosmos/ibc-go/v11/modules/core/03-connection/types"
	channelv11 "github.com/cosmos/ibc-go/v11/modules/core/04-channel/migrations/v11"
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v11/modules/core/23-commitment/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"

	sdk "github.com/cosmos/cosmos-sdk/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

// Channels opened under ibc-go v10 keep NextSequenceSend under the v1 key, which v11 no longer reads.
func TestNextUpgradeHandlerMovesIBCSendSequences(t *testing.T) {
	gapp := Setup(t)
	ctx := gapp.NewContext(false).WithChainID("xion-mainnet-1")
	const port, channel, connection, client = "transfer", "channel-7", "connection-3", "07-tendermint-5"

	gapp.IBCKeeper.ConnectionKeeper.SetConnection(ctx, connection, connectiontypes.ConnectionEnd{
		ClientId: client,
		Versions: connectiontypes.GetCompatibleVersions(),
		State:    connectiontypes.OPEN,
		Counterparty: connectiontypes.Counterparty{
			ClientId:     "07-tendermint-0",
			ConnectionId: "connection-0",
			Prefix:       commitmenttypes.NewMerklePrefix([]byte("ibc")),
		},
	})
	gapp.IBCKeeper.ChannelKeeper.SetChannel(ctx, port, channel, channeltypes.NewChannel(
		channeltypes.OPEN, channeltypes.UNORDERED, channeltypes.NewCounterparty(port, "channel-0"), []string{connection}, "ics20-1",
	))
	ibcStore := ctx.KVStore(gapp.GetKey(ibcexported.StoreKey))
	ibcStore.Set(channelv11.NextSequenceSendV1Key(port, channel), sdk.Uint64ToBigEndian(42))

	_, found := gapp.IBCKeeper.ChannelKeeper.GetNextSequenceSend(ctx, port, channel)
	require.False(t, found, "v11 must not read the v1 key")

	for range 2 {
		_, err := gapp.NextUpgradeHandler(ctx, upgradetypes.Plan{Name: UpgradeName, Height: 100}, gapp.ModuleManager.GetVersionMap())
		require.NoError(t, err)

		seq, found := gapp.IBCKeeper.ChannelKeeper.GetNextSequenceSend(ctx, port, channel)
		require.True(t, found)
		require.Equal(t, uint64(42), seq)
		require.False(t, ibcStore.Has(channelv11.NextSequenceSendV1Key(port, channel)))
		alias, found := gapp.IBCKeeper.ChannelKeeperV2.GetClientForAlias(ctx, channel)
		require.True(t, found)
		require.Equal(t, client, alias)
	}
}
