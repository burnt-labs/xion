package v6

import (
	tokenfactorytypes "github.com/CosmosContracts/juno/v21/x/tokenfactory/types"

	store "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/burnt-labs/xion/app/upgrades"
)

const (
	// UpgradeName defines the on-chain upgrade name.
	UpgradeName = "v6"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			tokenfactorytypes.ModuleName,
		},
	},
}
