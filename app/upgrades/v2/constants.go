package v2

import (
	xiontypes "github.com/burnt-labs/xion/x/xion/types"
	store "github.com/cosmos/cosmos-sdk/store/types"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v7/router/types"

	"github.com/burnt-labs/xion/app/upgrades"

	"github.com/burnt-labs/xion/x/globalfee"
	ibchookstypes "github.com/cosmos/ibc-apps/modules/ibc-hooks/v7/types"
	aatypes "github.com/larry0x/abstract-account/x/abstractaccount/types"
)

const (
	// UpgradeName defines the on-chain upgrade name.
	UpgradeName = "v2"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			globalfee.ModuleName,
			aatypes.ModuleName,
			ibchookstypes.StoreKey,
			packetforwardtypes.ModuleName,
			xiontypes.ModuleName,
		},
	},
}
