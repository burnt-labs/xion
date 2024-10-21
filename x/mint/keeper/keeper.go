package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/mint/types"
)

// Keeper of the mint store
type Keeper struct {
	cdc              codec.BinaryCodec
	storeService     store.KVStoreService
	stakingKeeper    types.StakingKeeper
	bankKeeper       types.BankKeeper
	accountKeeper    types.AccountKeeper
	feeCollectorName string

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

// NewKeeper creates a new mint Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	sk types.StakingKeeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	feeCollectorName string,
	authority string,
) Keeper {
	// ensure mint module account is set
	if addr := ak.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("the x/%s module account has not been set", types.ModuleName))
	}

	return Keeper{
		cdc:              cdc,
		storeService:     storeService,
		stakingKeeper:    sk,
		bankKeeper:       bk,
		accountKeeper:    ak,
		feeCollectorName: feeCollectorName,
		authority:        authority,
	}
}

// GetAuthority returns the x/mint module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// GetMinter returns the minter.
func (k Keeper) GetMinter(ctx sdk.Context) (minter types.Minter, err error) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.MinterKey)
	if err != nil {
		return types.Minter{}, err
	}
	if bz == nil {
		return types.Minter{}, err
	}

	k.cdc.MustUnmarshal(bz, &minter)
	return
}

// SetMinter sets the minter.
func (k Keeper) SetMinter(ctx sdk.Context, minter types.Minter) (err error) {
	store := k.storeService.OpenKVStore(ctx)
	bz := k.cdc.MustMarshal(&minter)
	err = store.Set(types.MinterKey, bz)
	return err
}

// SetParams sets the x/mint module parameters.
func (k Keeper) SetParams(ctx sdk.Context, p types.Params) error {
	if err := p.Validate(); err != nil {
		return err
	}

	store := k.storeService.OpenKVStore(ctx)
	bz := k.cdc.MustMarshal(&p)
	err := store.Set(types.ParamsKey, bz)
	if err != nil {
		return err
	}

	return nil
}

// GetParams returns the current x/mint module parameters.
func (k Keeper) GetParams(ctx sdk.Context) (p types.Params, err error) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil {
		return types.Params{}, err
	}
	if bz == nil {
		return p, nil
	}

	k.cdc.MustUnmarshal(bz, &p)
	return p, nil
}

// StakingTokenSupply implements an alias call to the underlying staking keeper's
// StakingTokenSupply to be used in BeginBlocker.
func (k Keeper) StakingTokenSupply(ctx sdk.Context) (math.Int, error) {
	return k.stakingKeeper.StakingTokenSupply(ctx)
}

// BondedTokenSupply implements an alian call to the underlying staking keeper's
// BondedTokenSupply to be used in BeginBlocker
func (k Keeper) BondedTokenSupply(ctx sdk.Context) (math.Int, error) {
	return k.stakingKeeper.TotalBondedTokens(ctx)
}

// BondedRatio implements an alias call to the underlying staking keeper's
// BondedRatio to be used in BeginBlocker.
func (k Keeper) BondedRatio(ctx sdk.Context) (math.LegacyDec, error) {
	return k.stakingKeeper.BondedRatio(ctx)
}

// MintCoins implements an alias call to the underlying supply keeper's
// MintCoins to be used in BeginBlocker.
func (k Keeper) MintCoins(ctx sdk.Context, newCoins sdk.Coins) error {
	if newCoins.Empty() {
		// skip as no coins need to be minted
		return nil
	}

	return k.bankKeeper.MintCoins(ctx, types.ModuleName, newCoins)
}

// AddCollectedFees implements an alias call to the underlying supply keeper's
// AddCollectedFees to be used in BeginBlocker.
func (k Keeper) AddCollectedFees(ctx sdk.Context, fees sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, k.feeCollectorName, fees)
}

// CountCollectedFees implements an alias call to the underlying supply keeper's
// CountCollectedFees to be used in BeginBlocker.
func (k Keeper) CountCollectedFees(ctx sdk.Context, denom string) sdk.Coin {
	return k.bankKeeper.GetBalance(ctx, k.accountKeeper.GetModuleAccount(ctx, k.feeCollectorName).GetAddress(), denom)
}

// BurnFees implements an alias call to the underlying supply keeper's
// BurnFees to be used in BeginBlocker.
func (k Keeper) BurnFees(ctx sdk.Context, fees sdk.Coins) error {
	return k.bankKeeper.BurnCoins(ctx, k.feeCollectorName, fees)
}
