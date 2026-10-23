package keeper

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

// The tests in this file run the function returned by StakedInflationMintFn
// against a real x/mint keeper backed by an in-memory store, with the bank,
// account and staking dependencies replaced by recording fakes.

// mintRunBankKeeper serves as both the x/mint bank keeper (MintCoins,
// SendCoinsFromModuleToModule) and the x/xion bank keeper (GetBalance,
// BurnCoins) so a single instance can be handed to both.
type mintRunBankKeeper struct {
	balance sdk.Coin
	mintErr error
	sendErr error
	burnErr error

	minted    sdk.Coins
	collected sdk.Coins
	burned    sdk.Coins
	burnCalls int
}

func (b *mintRunBankKeeper) GetBalance(_ context.Context, _ sdk.AccAddress, _ string) sdk.Coin {
	return b.balance
}

func (b *mintRunBankKeeper) MintCoins(_ context.Context, _ string, amt sdk.Coins) error {
	if b.mintErr != nil {
		return b.mintErr
	}
	b.minted = amt
	return nil
}

func (b *mintRunBankKeeper) SendCoinsFromModuleToModule(_ context.Context, _, _ string, amt sdk.Coins) error {
	if b.sendErr != nil {
		return b.sendErr
	}
	b.collected = amt
	return nil
}

func (b *mintRunBankKeeper) BurnCoins(_ context.Context, _ string, amt sdk.Coins) error {
	b.burnCalls++
	if b.burnErr != nil {
		return b.burnErr
	}
	b.burned = amt
	return nil
}

func (*mintRunBankKeeper) SendCoinsFromModuleToAccount(context.Context, string, sdk.AccAddress, sdk.Coins) error {
	return nil
}

func (*mintRunBankKeeper) SendCoinsFromAccountToModule(context.Context, sdk.AccAddress, string, sdk.Coins) error {
	return nil
}

func (b *mintRunBankKeeper) GetSupply(_ context.Context, denom string) sdk.Coin {
	return sdk.NewCoin(denom, math.ZeroInt())
}

func (*mintRunBankKeeper) IsSendEnabledCoins(context.Context, ...sdk.Coin) error { return nil }

func (*mintRunBankKeeper) BlockedAddr(sdk.AccAddress) bool { return false }

func (*mintRunBankKeeper) SendCoins(context.Context, sdk.AccAddress, sdk.AccAddress, sdk.Coins) error {
	return nil
}

func (*mintRunBankKeeper) InputOutputCoins(context.Context, banktypes.Input, []banktypes.Output) error {
	return nil
}

// mintRunAccountKeeper satisfies the x/mint and x/xion account keeper
// contracts with deterministic module accounts.
type mintRunAccountKeeper struct{}

func (mintRunAccountKeeper) GetModuleAddress(name string) sdk.AccAddress {
	return authtypes.NewModuleAddress(name)
}

func (mintRunAccountKeeper) SetModuleAccount(context.Context, sdk.ModuleAccountI) {}

func (mintRunAccountKeeper) GetModuleAccount(_ context.Context, name string) sdk.ModuleAccountI {
	return authtypes.NewEmptyModuleAccount(name)
}

// mintRunStakingKeeper satisfies the x/mint and x/xion staking keeper
// contracts.
type mintRunStakingKeeper struct {
	bonded    math.Int
	bondedErr error
	ratio     math.LegacyDec
	ratioErr  error
}

func (s mintRunStakingKeeper) TotalBondedTokens(context.Context) (math.Int, error) {
	return s.bonded, s.bondedErr
}

func (s mintRunStakingKeeper) TotalValidatorPower(context.Context) (math.Int, error) {
	return s.bonded, s.bondedErr
}

func (s mintRunStakingKeeper) BondedRatio(context.Context) (math.LegacyDec, error) {
	return s.ratio, s.ratioErr
}

func (s mintRunStakingKeeper) StakingTokenSupply(context.Context) (math.Int, error) {
	return s.bonded, nil
}

// With DefaultParams (6,311,520 blocks per year), a fixed 10% inflation on
// this bonded supply yields a block provision of exactly 100 stake.
const (
	mintRunBondedTokens   = int64(6_311_520_000)
	mintRunBlockProvision = int64(100)
)

func fixedInflation(_ context.Context, _ minttypes.Minter, _ minttypes.Params, _ math.LegacyDec) math.LegacyDec {
	return math.LegacyNewDecWithPrec(10, 2)
}

func newMintRunStakingKeeper() mintRunStakingKeeper {
	return mintRunStakingKeeper{
		bonded: math.NewInt(mintRunBondedTokens),
		ratio:  math.LegacyNewDecWithPrec(67, 2),
	}
}

func newMintRunKeeper(t *testing.T, sk mintRunStakingKeeper, bk *mintRunBankKeeper) (sdk.Context, *mintkeeper.Keeper) {
	t.Helper()

	key := storetypes.NewKVStoreKey(minttypes.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_mint"))
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	k := mintkeeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(key),
		sk,
		mintRunAccountKeeper{},
		bk,
		authtypes.FeeCollectorName,
		"authority",
	)

	return testCtx.Ctx, &k
}

func seedMintState(t *testing.T, ctx sdk.Context, k *mintkeeper.Keeper) {
	t.Helper()
	require.NoError(t, k.Params.Set(ctx, minttypes.DefaultParams()))
	require.NoError(t, k.Minter.Set(ctx, minttypes.DefaultInitialMinter()))
}

func mintEventAttribute(t *testing.T, ctx sdk.Context, key string) string {
	t.Helper()
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type != minttypes.EventTypeMint {
			continue
		}
		for _, attr := range ev.Attributes {
			if attr.Key == key {
				return attr.Value
			}
		}
	}
	t.Fatalf("mint event attribute %q not emitted", key)
	return ""
}

func TestStakedInflationMintFn_MintsShortfall(t *testing.T) {
	bk := &mintRunBankKeeper{balance: sdk.NewInt64Coin(sdk.DefaultBondDenom, 40)}
	ctx, k := newMintRunKeeper(t, newMintRunStakingKeeper(), bk)
	seedMintState(t, ctx, k)

	mintFn := StakedInflationMintFn(authtypes.FeeCollectorName, fixedInflation, bk, mintRunAccountKeeper{}, newMintRunStakingKeeper())
	require.NoError(t, mintFn(ctx, k))

	shortfall := sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, mintRunBlockProvision-40))
	require.Equal(t, shortfall, bk.minted)
	require.Equal(t, shortfall, bk.collected)
	require.Zero(t, bk.burnCalls)

	minter, err := k.Minter.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDecWithPrec(10, 2), minter.Inflation)
	require.Equal(t, math.LegacyNewDec(mintRunBondedTokens/10), minter.AnnualProvisions)

	require.Equal(t, "100", mintEventAttribute(t, ctx, AttributeKeyNeededAmount))
	require.Equal(t, "60", mintEventAttribute(t, ctx, AttributeKeyMintedAmount))
	require.Equal(t, "40", mintEventAttribute(t, ctx, AttributeKeyCollectedAmount))
	require.Equal(t, "0", mintEventAttribute(t, ctx, AttributeKeyBurnedAmount))
	require.Equal(t, "0.670000000000000000", mintEventAttribute(t, ctx, minttypes.AttributeKeyBondedRatio))
}

func TestStakedInflationMintFn_BurnsSurplus(t *testing.T) {
	bk := &mintRunBankKeeper{balance: sdk.NewInt64Coin(sdk.DefaultBondDenom, 250)}
	ctx, k := newMintRunKeeper(t, newMintRunStakingKeeper(), bk)
	seedMintState(t, ctx, k)

	mintFn := StakedInflationMintFn(authtypes.FeeCollectorName, fixedInflation, bk, mintRunAccountKeeper{}, newMintRunStakingKeeper())
	require.NoError(t, mintFn(ctx, k))

	require.Nil(t, bk.minted)
	require.Nil(t, bk.collected)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 250-mintRunBlockProvision)), bk.burned)

	require.Equal(t, "0", mintEventAttribute(t, ctx, AttributeKeyMintedAmount))
	require.Equal(t, "150", mintEventAttribute(t, ctx, AttributeKeyBurnedAmount))
	require.Equal(t, "250", mintEventAttribute(t, ctx, AttributeKeyCollectedAmount))
}

func TestStakedInflationMintFn_ExactFeesBurnNothing(t *testing.T) {
	bk := &mintRunBankKeeper{balance: sdk.NewInt64Coin(sdk.DefaultBondDenom, mintRunBlockProvision)}
	ctx, k := newMintRunKeeper(t, newMintRunStakingKeeper(), bk)
	seedMintState(t, ctx, k)

	mintFn := StakedInflationMintFn(authtypes.FeeCollectorName, fixedInflation, bk, mintRunAccountKeeper{}, newMintRunStakingKeeper())
	require.NoError(t, mintFn(ctx, k))

	require.Nil(t, bk.minted)
	require.Equal(t, 1, bk.burnCalls)
	require.True(t, bk.burned.IsZero())
	require.Equal(t, "0", mintEventAttribute(t, ctx, AttributeKeyBurnedAmount))
	require.Equal(t, "0", mintEventAttribute(t, ctx, AttributeKeyMintedAmount))
}

func TestStakedInflationMintFn_ErrorPaths(t *testing.T) {
	boom := errors.New("boom")

	testCases := []struct {
		name     string
		balance  int64
		staking  func() mintRunStakingKeeper
		bank     func(bk *mintRunBankKeeper)
		seed     func(t *testing.T, ctx sdk.Context, k *mintkeeper.Keeper)
		expected error
	}{
		{
			name:    "minter not stored",
			balance: 40,
			staking: newMintRunStakingKeeper,
			seed: func(t *testing.T, ctx sdk.Context, k *mintkeeper.Keeper) {
				t.Helper()
				require.NoError(t, k.Params.Set(ctx, minttypes.DefaultParams()))
			},
			expected: collections.ErrNotFound,
		},
		{
			name:    "params not stored",
			balance: 40,
			staking: newMintRunStakingKeeper,
			seed: func(t *testing.T, ctx sdk.Context, k *mintkeeper.Keeper) {
				t.Helper()
				require.NoError(t, k.Minter.Set(ctx, minttypes.DefaultInitialMinter()))
			},
			expected: collections.ErrNotFound,
		},
		{
			name:    "bonded ratio lookup fails",
			balance: 40,
			staking: func() mintRunStakingKeeper {
				sk := newMintRunStakingKeeper()
				sk.ratioErr = boom
				return sk
			},
			seed:     seedMintState,
			expected: boom,
		},
		{
			name:    "bonded tokens lookup fails",
			balance: 40,
			staking: func() mintRunStakingKeeper {
				sk := newMintRunStakingKeeper()
				sk.bondedErr = boom
				return sk
			},
			seed:     seedMintState,
			expected: boom,
		},
		{
			name:     "minting the shortfall fails",
			balance:  40,
			staking:  newMintRunStakingKeeper,
			bank:     func(bk *mintRunBankKeeper) { bk.mintErr = boom },
			seed:     seedMintState,
			expected: boom,
		},
		{
			name:     "forwarding minted coins to the fee collector fails",
			balance:  40,
			staking:  newMintRunStakingKeeper,
			bank:     func(bk *mintRunBankKeeper) { bk.sendErr = boom },
			seed:     seedMintState,
			expected: boom,
		},
		{
			name:     "burning the surplus fails",
			balance:  250,
			staking:  newMintRunStakingKeeper,
			bank:     func(bk *mintRunBankKeeper) { bk.burnErr = boom },
			seed:     seedMintState,
			expected: boom,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bk := &mintRunBankKeeper{balance: sdk.NewInt64Coin(sdk.DefaultBondDenom, tc.balance)}
			if tc.bank != nil {
				tc.bank(bk)
			}
			sk := tc.staking()
			ctx, k := newMintRunKeeper(t, sk, bk)
			tc.seed(t, ctx, k)

			mintFn := StakedInflationMintFn(authtypes.FeeCollectorName, fixedInflation, bk, mintRunAccountKeeper{}, sk)
			err := mintFn(ctx, k)
			require.ErrorIs(t, err, tc.expected)

			// A failed run must not emit the mint event.
			for _, ev := range ctx.EventManager().Events() {
				require.NotEqual(t, minttypes.EventTypeMint, ev.Type)
			}
		})
	}
}
