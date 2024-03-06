package v4

import (
	store "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/burnt-labs/xion/app/upgrades"
	jwktypes "github.com/burnt-labs/xion/x/jwk/types"
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
