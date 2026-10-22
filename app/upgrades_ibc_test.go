package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v11/modules/core/03-connection/types"
	channelv11 "github.com/cosmos/ibc-go/v11/modules/core/04-channel/migrations/v11"
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v11/modules/core/23-commitment/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v11/modules/light-clients/07-tendermint"

	sdk "github.com/cosmos/cosmos-sdk/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

// Seed the v10 layout directly: the v11 setter would hide the missing migration.
func TestNextUpgradeHandlerMovesIBCSendSequences(t *testing.T) {
	gapp := Setup(t)
	t.Cleanup(func() { require.NoError(t, gapp.Close()) })
	ctx := gapp.NewContext(false).WithChainID("xion-mainnet-1").
		WithBlockHeight(100).WithBlockTime(time.Unix(1_700_000_000, 0).UTC())
	const port, connection = "transfer", "connection-3"

	clientState := ibctm.NewClientState(
		"counterparty-1", ibctm.DefaultTrustLevel, 24*time.Hour, 48*time.Hour, time.Minute,
		clienttypes.NewHeight(1, 10), commitmenttypes.GetSDKSpecs(), []string{"upgrade", "upgradedIBCState"},
	)
	consensusState := ibctm.NewConsensusState(
		ctx.BlockTime(), commitmenttypes.NewMerkleRoot(make([]byte, 32)), make([]byte, 32),
	)
	client, err := gapp.IBCKeeper.ClientKeeper.CreateClient(ctx, ibcexported.Tendermint,
		gapp.AppCodec().MustMarshal(clientState), gapp.AppCodec().MustMarshal(consensusState))
	require.NoError(t, err)
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

	channels := []struct {
		id       string
		state    channeltypes.State
		ordering channeltypes.Order
		sequence uint64
	}{
		{"channel-7", channeltypes.OPEN, channeltypes.UNORDERED, 42},
		{"channel-8", channeltypes.OPEN, channeltypes.ORDERED, 67},
		{"channel-9", channeltypes.CLOSED, channeltypes.UNORDERED, 93},
	}
	ibcStore := ctx.KVStore(gapp.GetKey(ibcexported.StoreKey))
	for _, ch := range channels {
		gapp.IBCKeeper.ChannelKeeper.SetChannel(ctx, port, ch.id, channeltypes.NewChannel(
			ch.state, ch.ordering, channeltypes.NewCounterparty(port, "channel-0"), []string{connection}, "ics20-1",
		))
		ibcStore.Set(channelv11.NextSequenceSendV1Key(port, ch.id), sdk.Uint64ToBigEndian(ch.sequence))
	}

	data := []byte(`{"denom":"uxion","amount":"1","sender":"sender","receiver":"receiver"}`)
	timeout := uint64(ctx.BlockTime().Add(time.Hour).UnixNano())
	_, err = gapp.IBCKeeper.ChannelKeeper.SendPacket(ctx, port, channels[0].id, clienttypes.ZeroHeight(), timeout, data)
	require.ErrorIs(t, err, channeltypes.ErrSequenceSendNotFound)

	fromVM := gapp.ModuleManager.GetVersionMap()
	for attempt := range 2 {
		fromVM, err = gapp.NextUpgradeHandler(ctx, upgradetypes.Plan{Name: UpgradeName, Height: 100}, fromVM)
		require.NoError(t, err)

		for _, ch := range channels {
			seq, found := gapp.IBCKeeper.ChannelKeeper.GetNextSequenceSend(ctx, port, ch.id)
			require.True(t, found, ch.id)
			expected := ch.sequence
			if attempt == 1 && ch.state == channeltypes.OPEN {
				expected++ // A rerun must preserve sequences advanced by packet sends.
			}
			require.Equal(t, expected, seq, ch.id)
			require.False(t, ibcStore.Has(channelv11.NextSequenceSendV1Key(port, ch.id)))
			// Alias/counterparty eligibility (OPEN+UNORDERED only) is upstream
			// ibc-go v11.0 behavior, asserted here to document the snapshot we
			// migrated under. Revisit intentionally if a future ibc-go bump
			// relaxes or extends the eligibility rule.
			eligible := ch.state == channeltypes.OPEN && ch.ordering == channeltypes.UNORDERED
			alias, hasAlias := gapp.IBCKeeper.ChannelKeeperV2.GetClientForAlias(ctx, ch.id)
			counterparty, hasCounterparty := gapp.IBCKeeper.ClientV2Keeper.GetClientCounterparty(ctx, ch.id)
			if !eligible {
				require.False(t, hasAlias && hasCounterparty, "non-eligible %s must not gain v2 routing state", ch.id)
			} else {
				require.True(t, hasAlias, ch.id)
				require.True(t, hasCounterparty, ch.id)
				require.Equal(t, client, alias)
				require.Equal(t, "channel-0", counterparty.ClientId)
			}
			if ch.state != channeltypes.OPEN {
				continue
			}

			seq, err = gapp.IBCKeeper.ChannelKeeper.SendPacket(ctx, port, ch.id, clienttypes.ZeroHeight(), timeout, data)
			require.NoError(t, err)
			require.Equal(t, expected, seq)
			packet := channeltypes.NewPacket(data, seq, port, ch.id, port, "channel-0", clienttypes.ZeroHeight(), timeout)
			require.Equal(t, channeltypes.CommitPacket(packet), gapp.IBCKeeper.ChannelKeeper.GetPacketCommitment(ctx, port, ch.id, seq))
			nextSequence, found := gapp.IBCKeeper.ChannelKeeper.GetNextSequenceSend(ctx, port, ch.id)
			require.True(t, found)
			require.Equal(t, seq+1, nextSequence)
			t.Logf("upgrade pass %d: %s sent packet %d; next sequence %d", attempt+1, ch.id, seq, nextSequence)
		}
	}
}

// Fresh v11 state has no legacy keys; it must not be rewritten by the upgrade.
func TestNextUpgradeHandlerPreservesV11SendSequences(t *testing.T) {
	gapp := Setup(t)
	t.Cleanup(func() { require.NoError(t, gapp.Close()) })
	ctx := gapp.NewContext(false).WithChainID("xion-mainnet-1")
	const port, channel = "transfer", "channel-7"
	gapp.IBCKeeper.ChannelKeeper.SetChannel(ctx, port, channel, channeltypes.NewChannel(
		channeltypes.CLOSED, channeltypes.UNORDERED, channeltypes.NewCounterparty(port, "channel-0"), []string{"connection-0"}, "ics20-1",
	))
	gapp.IBCKeeper.ChannelKeeper.SetNextSequenceSend(ctx, port, channel, 75)

	_, err := gapp.NextUpgradeHandler(ctx, upgradetypes.Plan{Name: UpgradeName, Height: 100}, gapp.ModuleManager.GetVersionMap())
	require.NoError(t, err)
	seq, found := gapp.IBCKeeper.ChannelKeeper.GetNextSequenceSend(ctx, port, channel)
	require.True(t, found)
	require.Equal(t, uint64(75), seq)
}
