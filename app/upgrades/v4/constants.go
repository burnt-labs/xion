package v4

import (
	"github.com/burnt-labs/xion/app/upgrades"
	jwktypes "github.com/burnt-labs/xion/x/jwk/types"

	store "github.com/cosmos/cosmos-sdk/store/types"
)

const (
	// UpgradeName defines the on-chain upgrade name.
	UpgradeName = "v4"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			jwktypes.ModuleName,
		},
	},
}
