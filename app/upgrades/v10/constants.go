package v10

import (
	storetypes "cosmossdk.io/store/types"

	"github.com/burnt-labs/xion/app/upgrades"
)

const (
	UpgradeName = "v10"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: storetypes.StoreUpgrades{
		Added: []string{},
	},
}
