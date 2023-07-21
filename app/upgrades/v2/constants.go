package v2

import (
	store "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/burnt-labs/xion/app/upgrades"

	"github.com/burnt-labs/xion/x/globalfee"
	aatypes "github.com/larry0x/abstract-account/x/abstractaccount/types"
)

const (
	// UpgradeName defines the on-chain upgrade name.
	UpgradeName = "v3.0.0"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			globalfee.ModuleName,
			aatypes.ModuleName,
		},
	},
}
