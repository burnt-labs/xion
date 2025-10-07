package module

import (
	"cosmossdk.io/core/appmodule"
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

/*
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
*/
