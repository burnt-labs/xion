package module

import (
	"os"

	"cosmossdk.io/core/address"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	modulev1 "github.com/burnt-labs/xion/api/xion/dkim/module/v1"
	"github.com/burnt-labs/xion/x/dkim/keeper"
)

var _ appmodule.AppModule = AppModule{}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {
	_ = am
}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {
	_ = am
}

func init() {
	appmodule.Register(
		&modulev1.Module{},
		appmodule.Provide(ProvideModule),
	)
}

type Inputs struct {
	depinject.In

	Cdc          codec.Codec
	StoreService store.KVStoreService
	AddressCodec address.Codec
}

type Outputs struct {
	depinject.Out

	Module appmodule.AppModule
	Keeper keeper.Keeper
}

func ProvideModule(in Inputs) Outputs {
	govAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	k := keeper.NewKeeper(in.Cdc, in.StoreService, log.NewLogger(os.Stderr), govAddr)
	m := NewAppModule(in.Cdc, k)

	return Outputs{Module: m, Keeper: k, Out: depinject.Out{}}
}
