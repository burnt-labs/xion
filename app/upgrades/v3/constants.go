package v3

import (
	store "github.com/cosmos/cosmos-sdk/store/types"
	feeabstypes "github.com/osmosis-labs/fee-abstraction/v7/x/feeabs/types"

	"github.com/burnt-labs/xion/app/upgrades"
)

const (
	// UpgradeName defines the on-chain upgrade name.
	UpgradeName = "v3"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			feeabstypes.ModuleName,
		},
	},
}
