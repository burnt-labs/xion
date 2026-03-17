package app

import (
	"os"
	"strings"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/libs/bytes"
	tmos "github.com/cometbft/cometbft/libs/os"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

// InitXionAppForTestnet modifies an existing app's state to create a single-validator testnet.
// This is used by the "in-place-testnet" (testnet-from-state) command to fork mainnet state.
//
// Required changes prevent the testnet from halting or panicking.
// Optional changes customize the testnet for development use.
func InitXionAppForTestnet(app *WasmApp, newValAddr bytes.HexBytes, newValPubKey crypto.PubKey, newOperatorAddress, upgradeToTrigger string) *WasmApp {
	//
	// Required Changes
	//

	ctx := app.BaseApp.NewUncachedContext(true, tmproto.Header{})

	pubkey := &ed25519.PubKey{Key: newValPubKey.Bytes()}
	pubkeyAny, err := codectypes.NewAnyWithValue(pubkey)
	if err != nil {
		tmos.Exit(err.Error())
	}

	// STAKING — replace all validators with a single new one

	_, bz, err := bech32.DecodeAndConvert(newOperatorAddress)
	if err != nil {
		tmos.Exit(err.Error())
	}
	bech32Addr, err := bech32.ConvertAndEncode(Bech32PrefixValAddr, bz)
	if err != nil {
		tmos.Exit(err.Error())
	}

	newVal := stakingtypes.Validator{
		OperatorAddress: bech32Addr,
		ConsensusPubkey: pubkeyAny,
		Jailed:          false,
		Status:          stakingtypes.Bonded,
		Tokens:          math.NewInt(900_000_000_000_000),
		DelegatorShares: math.LegacyMustNewDecFromStr("10000000"),
		Description: stakingtypes.Description{
			Moniker: "Testnet Validator",
		},
		Commission: stakingtypes.Commission{
			CommissionRates: stakingtypes.CommissionRates{
				Rate:          math.LegacyMustNewDecFromStr("0.05"),
				MaxRate:       math.LegacyMustNewDecFromStr("0.1"),
				MaxChangeRate: math.LegacyMustNewDecFromStr("0.05"),
			},
		},
		MinSelfDelegation: math.OneInt(),
	}

	// Remove all validators from power store
	stakingKey := app.GetKey(stakingtypes.ModuleName)
	stakingStore := ctx.KVStore(stakingKey)

	iterator, err := app.StakingKeeper.ValidatorsPowerStoreIterator(ctx)
	if err != nil {
		tmos.Exit(err.Error())
	}
	for ; iterator.Valid(); iterator.Next() {
		stakingStore.Delete(iterator.Key())
	}
	iterator.Close()

	// Remove all validators from last validators store
	iterator, err = app.StakingKeeper.LastValidatorsIterator(ctx)
	if err != nil {
		tmos.Exit(err.Error())
	}
	for ; iterator.Valid(); iterator.Next() {
		stakingStore.Delete(iterator.Key())
	}
	iterator.Close()

	// Remove all validators from validators store
	iterator = storetypes.KVStorePrefixIterator(stakingStore, stakingtypes.ValidatorsKey)
	for ; iterator.Valid(); iterator.Next() {
		stakingStore.Delete(iterator.Key())
	}
	iterator.Close()

	// Remove all validators from unbonding queue
	iterator = storetypes.KVStorePrefixIterator(stakingStore, stakingtypes.ValidatorQueueKey)
	for ; iterator.Valid(); iterator.Next() {
		stakingStore.Delete(iterator.Key())
	}
	iterator.Close()

	// Add the new validator
	if err = app.StakingKeeper.SetValidator(ctx, newVal); err != nil {
		tmos.Exit(err.Error())
	}
	if err = app.StakingKeeper.SetValidatorByConsAddr(ctx, newVal); err != nil {
		tmos.Exit(err.Error())
	}
	if err = app.StakingKeeper.SetValidatorByPowerIndex(ctx, newVal); err != nil {
		tmos.Exit(err.Error())
	}

	valAddr, err := sdk.ValAddressFromBech32(newVal.GetOperator())
	if err != nil {
		tmos.Exit(err.Error())
	}
	if err = app.StakingKeeper.SetLastValidatorPower(ctx, valAddr, 0); err != nil {
		tmos.Exit(err.Error())
	}
	if err = app.StakingKeeper.Hooks().AfterValidatorCreated(ctx, valAddr); err != nil {
		tmos.Exit(err.Error())
	}

	// DISTRIBUTION — initialize records for the new validator

	if err = app.DistrKeeper.SetValidatorHistoricalRewards(ctx, valAddr, 0, distrtypes.NewValidatorHistoricalRewards(sdk.DecCoins{}, 1)); err != nil {
		tmos.Exit(err.Error())
	}
	if err = app.DistrKeeper.SetValidatorCurrentRewards(ctx, valAddr, distrtypes.NewValidatorCurrentRewards(sdk.DecCoins{}, 1)); err != nil {
		tmos.Exit(err.Error())
	}
	if err = app.DistrKeeper.SetValidatorAccumulatedCommission(ctx, valAddr, distrtypes.InitialValidatorAccumulatedCommission()); err != nil {
		tmos.Exit(err.Error())
	}
	if err = app.DistrKeeper.SetValidatorOutstandingRewards(ctx, valAddr, distrtypes.ValidatorOutstandingRewards{Rewards: sdk.DecCoins{}}); err != nil {
		tmos.Exit(err.Error())
	}

	// SLASHING — register signing info so the new validator doesn't get jailed

	newConsAddr := sdk.ConsAddress(newValAddr.Bytes())
	newValidatorSigningInfo := slashingtypes.ValidatorSigningInfo{
		Address:     newConsAddr.String(),
		StartHeight: app.LastBlockHeight() - 1,
		Tombstoned:  false,
	}
	if err = app.SlashingKeeper.SetValidatorSigningInfo(ctx, newConsAddr, newValidatorSigningInfo); err != nil {
		tmos.Exit(err.Error())
	}

	//
	// Optional Changes
	//

	// GOV — shorten voting periods for testnet
	govParams, err := app.GovKeeper.Params.Get(ctx)
	if err != nil {
		tmos.Exit(err.Error())
	}
	newExpeditedVotingPeriod := time.Minute
	newVotingPeriod := time.Minute * 2
	govParams.ExpeditedVotingPeriod = &newExpeditedVotingPeriod
	govParams.VotingPeriod = &newVotingPeriod
	govParams.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin("uxion", 100_000_000))
	govParams.ExpeditedMinDeposit = sdk.NewCoins(sdk.NewInt64Coin("uxion", 150_000_000))
	if err = app.GovKeeper.Params.Set(ctx, govParams); err != nil {
		tmos.Exit(err.Error())
	}

	// WASM — allow operator to upload contracts
	wasmParams := app.WasmKeeper.GetParams(ctx)
	wasmParams.CodeUploadAccess = wasmtypes.AccessTypeAnyOfAddresses.With(sdk.MustAccAddressFromBech32(newOperatorAddress))
	wasmParams.InstantiateDefaultPermission = wasmtypes.AccessTypeEverybody
	if err = app.WasmKeeper.SetParams(ctx, wasmParams); err != nil {
		tmos.Exit(err.Error())
	}

	// BANK — fund accounts
	//
	// Env vars (comma-separated):
	//   XION_TESTNET_FUND_ACCOUNTS — addresses to fund (operator is always included)
	//   XION_TESTNET_FUND_DENOMS   — coins to mint per account (e.g. "1000000uxion,500000uatom")
	//                                 defaults to 1M XION if not set
	fundCoins := sdk.NewCoins(sdk.NewInt64Coin("uxion", 1_000_000_000_000))
	if denomsEnv := os.Getenv("XION_TESTNET_FUND_DENOMS"); denomsEnv != "" {
		parsed, err := sdk.ParseCoinsNormalized(denomsEnv)
		if err != nil {
			tmos.Exit("invalid XION_TESTNET_FUND_DENOMS: " + err.Error())
		}
		fundCoins = parsed
	}

	fundAddrs := []string{newOperatorAddress}
	if acctEnv := os.Getenv("XION_TESTNET_FUND_ACCOUNTS"); acctEnv != "" {
		for _, addr := range strings.Split(acctEnv, ",") {
			addr = strings.TrimSpace(addr)
			if addr != "" {
				fundAddrs = append(fundAddrs, addr)
			}
		}
	}

	for _, addr := range fundAddrs {
		acct := sdk.MustAccAddressFromBech32(addr)
		if err := app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, fundCoins); err != nil {
			tmos.Exit(err.Error())
		}
		if err := app.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, acct, fundCoins); err != nil {
			tmos.Exit(err.Error())
		}
	}

	// UPGRADE — optionally schedule an upgrade
	if upgradeToTrigger != "" {
		upgradePlan := upgradetypes.Plan{
			Name:   upgradeToTrigger,
			Height: app.LastBlockHeight() + 10,
		}
		if err = app.UpgradeKeeper.ScheduleUpgrade(ctx, upgradePlan); err != nil {
			tmos.Exit(err.Error())
		}
	}

	return app
}
