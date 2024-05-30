package v3

import (
	feeabstypes "github.com/osmosis-labs/fee-abstraction/v7/x/feeabs/types"

	store "cosmossdk.io/store/types"

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
