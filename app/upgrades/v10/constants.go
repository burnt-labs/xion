package v10

import (
	store "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/burnt-labs/xion/app/upgrades"
)

const (
	UpgradeName = "v10"
)

// Upgrade in this context, is a placeholder to contrast with an UpgradeV10 during app init.
var Upgrade = upgrades.Upgrade{
	UpgradeName:          "placeholder",
	CreateUpgradeHandler: nil,
	StoreUpgrades:        store.StoreUpgrades{},
}

// UpgradeV10 extends the original Upgrade struct with a minion
type UpgradeV10 struct {
	upgrades.Upgrade
	Minion *UpgradeMinion
}

// NewUpgradeV10 initializes a new UpgradeV10 instance
func NewUpgradeV10(
	minion *UpgradeMinion,
) UpgradeV10 {
	return UpgradeV10{
		Upgrade: upgrades.Upgrade{
			UpgradeName:   UpgradeName,
			StoreUpgrades: store.StoreUpgrades{},
		},
		Minion: minion,
	}
}

// CreateUpgradeHandler overrides the base function to add a minion
func (u UpgradeV10) CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return CreateUpgradeHandler(mm, configurator, u.Minion)
}
