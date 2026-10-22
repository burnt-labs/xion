package app

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	cmtcrypto "github.com/cometbft/cometbft/crypto"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"

	corestore "cosmossdk.io/core/store"
	sdkmath "cosmossdk.io/math"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const testnetValidatorPower int64 = 900_000_000_000_000

// publicNetworkChainFamilies are the chain-ID names, without the revision
// number, of public XION networks. Transactions and consensus votes are signed
// over the chain ID, so a converted copy that reuses one of these names makes
// its signatures valid on the real network: replayable transactions and, with a
// production validator key, double-sign evidence.
var publicNetworkChainFamilies = []string{"xion-mainnet", "xion-testnet"}

// ValidateTestnetChainID rejects a replacement chain ID that reuses a public
// XION network name in any revision (xion-mainnet-N, xion-testnet-N, compared
// case-insensitively), or, when sourceChainID is not empty, that equals the
// source chain ID or is another revision of it.
func ValidateTestnetChainID(chainID, sourceChainID string) error {
	if strings.TrimSpace(chainID) == "" {
		return errors.New("testnet chain ID is required")
	}
	family := testnetChainIDFamily(chainID)
	for _, public := range publicNetworkChainFamilies {
		if family == public {
			return fmt.Errorf("testnet chain ID %q reuses the public network name %s-N; choose a chain ID outside the %s family", chainID, public, public)
		}
	}
	if sourceChainID == "" {
		return nil
	}
	if chainID == sourceChainID {
		return fmt.Errorf("testnet chain ID %q must differ from source chain ID", chainID)
	}
	if family == testnetChainIDFamily(sourceChainID) {
		return fmt.Errorf("testnet chain ID %q is a revision of source chain ID %q; choose a different name, not a different revision number", chainID, sourceChainID)
	}
	return nil
}

// testnetChainIDFamily returns the lower-cased chain ID without a trailing
// "-<revision number>", following the IBC {name}-{revision} convention.
func testnetChainIDFamily(chainID string) string {
	id := strings.ToLower(strings.TrimSpace(chainID))
	if i := strings.LastIndexByte(id, '-'); i > 0 {
		if revision := id[i+1:]; revision != "" && strings.Trim(revision, "0123456789") == "" {
			return id[:i]
		}
	}
	return id
}

// InitForTestnet replaces the active validator set with the local testnet
// validator and optionally schedules an upgrade after ten blocks, allowing the
// converted application state to commit before the old binary halts.
// It is only called by the in-place-testnet app creator.
func (app *WasmApp) InitForTestnet(pubKey cmtcrypto.PubKey, operator, chainID, upgradeName string) error {
	if pubKey == nil {
		return errors.New("testnet validator public key is required")
	}
	if err := ValidateTestnetChainID(chainID, ""); err != nil {
		return err
	}
	if len(chainID) > cmttypes.MaxChainIDLen {
		return fmt.Errorf("testnet chain ID is too long: got %d bytes, maximum is %d", len(chainID), cmttypes.MaxChainIDLen)
	}

	sdkPubKey, err := cryptocodec.FromCmtPubKeyInterface(pubKey)
	if err != nil {
		return fmt.Errorf("convert testnet validator public key: %w", err)
	}

	operatorBytes, err := app.AccountKeeper.AddressCodec().StringToBytes(operator)
	if err != nil {
		operatorBytes, err = app.StakingKeeper.ValidatorAddressCodec().StringToBytes(operator)
		if err != nil {
			return fmt.Errorf("decode testnet validator operator address: %w", err)
		}
	}
	operatorAddress, err := app.StakingKeeper.ValidatorAddressCodec().BytesToString(operatorBytes)
	if err != nil {
		return fmt.Errorf("encode testnet validator operator address: %w", err)
	}
	validatorAddress := sdk.ValAddress(operatorBytes)

	newValidator, err := stakingtypes.NewValidator(
		operatorAddress,
		sdkPubKey,
		stakingtypes.Description{Moniker: "Testnet Validator"},
	)
	if err != nil {
		return fmt.Errorf("create testnet validator: %w", err)
	}

	height := app.LastBlockHeight()
	baseCtx := app.NewUncachedContext(true, tmproto.Header{ChainID: chainID, Height: height})
	validatorTokens := app.StakingKeeper.TokensFromConsensusPower(baseCtx, testnetValidatorPower)
	newValidator.Status = stakingtypes.Bonded
	// The synthetic stake and matching shares make the rehearsal validator
	// operational; they intentionally do not replicate production staking accounting.
	newValidator.Tokens = validatorTokens
	newValidator.DelegatorShares = sdkmath.LegacyNewDecFromInt(validatorTokens)
	newValidator.Commission = stakingtypes.NewCommission(
		sdkmath.LegacyMustNewDecFromStr("0.05"),
		sdkmath.LegacyMustNewDecFromStr("0.10"),
		sdkmath.LegacyMustNewDecFromStr("0.05"),
	)
	newValidator.MinSelfDelegation = sdkmath.OneInt()
	if power := newValidator.ConsensusPower(app.StakingKeeper.PowerReduction(baseCtx)); power != testnetValidatorPower {
		return fmt.Errorf("testnet validator power mismatch: got %d, expected %d", power, testnetValidatorPower)
	}

	consensusAddressBytes, err := newValidator.GetConsAddr()
	if err != nil {
		return fmt.Errorf("derive testnet validator consensus address: %w", err)
	}
	if !bytes.Equal(consensusAddressBytes, pubKey.Address()) {
		return errors.New("converted testnet validator public key produced a different consensus address")
	}
	consensusAddress := sdk.ConsAddress(consensusAddressBytes)

	if _, err := app.StakingKeeper.GetValidator(baseCtx, validatorAddress); err == nil {
		return fmt.Errorf("testnet validator operator address %s is already in use", operatorAddress)
	} else if !errors.Is(err, stakingtypes.ErrNoValidatorFound) {
		return fmt.Errorf("check testnet validator operator address: %w", err)
	}
	if existing, err := app.StakingKeeper.GetValidatorByConsAddr(baseCtx, consensusAddress); err == nil {
		return fmt.Errorf("testnet validator consensus key is already used by %s", existing.OperatorAddress)
	} else if !errors.Is(err, stakingtypes.ErrNoValidatorFound) {
		return fmt.Errorf("check testnet validator consensus key: %w", err)
	}

	bondDenom, err := app.StakingKeeper.BondDenom(baseCtx)
	if err != nil {
		return fmt.Errorf("get staking bond denomination: %w", err)
	}
	mintAddress := app.AccountKeeper.GetModuleAddress(minttypes.ModuleName)
	if mintAddress == nil {
		return fmt.Errorf("%s module account is not configured", minttypes.ModuleName)
	}
	mintAccount, ok := app.AccountKeeper.GetAccount(baseCtx, mintAddress).(sdk.ModuleAccountI)
	if !ok || !mintAccount.HasPermission(authtypes.Minter) {
		return fmt.Errorf("%s module account cannot mint testnet validator stake", minttypes.ModuleName)
	}
	bondedPoolAddress := app.AccountKeeper.GetModuleAddress(stakingtypes.BondedPoolName)
	if bondedPoolAddress == nil {
		return errors.New("bonded staking pool module account is not configured")
	}
	if _, ok := app.AccountKeeper.GetAccount(baseCtx, bondedPoolAddress).(sdk.ModuleAccountI); !ok {
		return errors.New("bonded staking pool module account does not exist")
	}

	var upgradePlan *upgradetypes.Plan
	if upgradeName != "" {
		plan := upgradetypes.Plan{Name: upgradeName, Height: height + 10}
		if err := plan.ValidateBasic(); err != nil {
			return fmt.Errorf("validate testnet upgrade plan: %w", err)
		}
		doneHeight, err := app.UpgradeKeeper.GetDoneHeight(baseCtx, upgradeName)
		if err != nil {
			return fmt.Errorf("check completed testnet upgrade: %w", err)
		}
		if doneHeight != 0 {
			return fmt.Errorf("upgrade %q was already completed at height %d", upgradeName, doneHeight)
		}
		upgradePlan = &plan
	}

	powerKeys, err := testnetIndexKeys(app.StakingKeeper.ValidatorsPowerStoreIterator(baseCtx))
	if err != nil {
		return fmt.Errorf("read validator power index: %w", err)
	}
	lastValidatorKeys, err := testnetIndexKeys(app.StakingKeeper.LastValidatorsIterator(baseCtx))
	if err != nil {
		return fmt.Errorf("read last-validator index: %w", err)
	}

	ctx, write := baseCtx.CacheContext()
	stakingStore := runtime.NewKVStoreService(app.keys[stakingtypes.StoreKey]).OpenKVStore(ctx)
	for _, key := range powerKeys {
		if err := stakingStore.Delete(key); err != nil {
			return fmt.Errorf("clear validator power index: %w", err)
		}
	}
	for _, key := range lastValidatorKeys {
		if err := stakingStore.Delete(key); err != nil {
			return fmt.Errorf("clear last-validator index: %w", err)
		}
	}

	if err := app.StakingKeeper.SetValidator(ctx, newValidator); err != nil {
		return fmt.Errorf("store testnet validator: %w", err)
	}
	if err := app.StakingKeeper.SetValidatorByConsAddr(ctx, newValidator); err != nil {
		return fmt.Errorf("index testnet validator consensus address: %w", err)
	}
	if err := app.StakingKeeper.SetValidatorByPowerIndex(ctx, newValidator); err != nil {
		return fmt.Errorf("index testnet validator power: %w", err)
	}
	if err := app.StakingKeeper.SetLastValidatorPower(ctx, validatorAddress, 0); err != nil {
		return fmt.Errorf("initialize testnet validator last power: %w", err)
	}
	if err := app.StakingKeeper.SetLastTotalPower(ctx, sdkmath.ZeroInt()); err != nil {
		return fmt.Errorf("reset staking last total power: %w", err)
	}
	if err := app.StakingKeeper.Hooks().AfterValidatorCreated(ctx, validatorAddress); err != nil {
		return fmt.Errorf("initialize testnet validator hooks: %w", err)
	}

	signingInfo := slashingtypes.NewValidatorSigningInfo(
		consensusAddress,
		height-1,
		0,
		time.Unix(0, 0).UTC(),
		false,
		0,
	)
	if err := app.SlashingKeeper.SetValidatorSigningInfo(ctx, consensusAddress, signingInfo); err != nil {
		return fmt.Errorf("initialize testnet validator signing info: %w", err)
	}

	validatorStake := sdk.NewCoins(sdk.NewCoin(bondDenom, validatorTokens))
	if err := app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, validatorStake); err != nil {
		return fmt.Errorf("mint testnet validator stake: %w", err)
	}
	if err := app.BankKeeper.SendCoinsFromModuleToModule(
		ctx,
		minttypes.ModuleName,
		stakingtypes.BondedPoolName,
		validatorStake,
	); err != nil {
		return fmt.Errorf("fund bonded staking pool for testnet validator: %w", err)
	}

	// Give the local operator a usable account for proposals, votes and test
	// transactions. These coins exist only in the explicitly converted clone.
	operatorFunds := sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 1_000_000_000_000))
	if err := app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, operatorFunds); err != nil {
		return fmt.Errorf("mint local operator funds: %w", err)
	}
	if err := app.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, sdk.AccAddress(operatorBytes), operatorFunds); err != nil {
		return fmt.Errorf("fund local operator account: %w", err)
	}
	govParams, err := app.GovKeeper.Params.Get(ctx)
	if err != nil {
		return fmt.Errorf("read testnet governance params: %w", err)
	}
	votingPeriod, expeditedPeriod := 20*time.Second, 10*time.Second
	govParams.VotingPeriod = &votingPeriod
	govParams.ExpeditedVotingPeriod = &expeditedPeriod
	if err := app.GovKeeper.Params.Set(ctx, govParams); err != nil {
		return fmt.Errorf("set testnet governance voting periods: %w", err)
	}

	if upgradePlan != nil {
		if err := app.UpgradeKeeper.ScheduleUpgrade(ctx, *upgradePlan); err != nil {
			return fmt.Errorf("schedule testnet upgrade: %w", err)
		}
	}

	// Flush into the loaded root multistore so the first normal block sees these
	// changes. Do not commit here: committing outside the ABCI lifecycle would
	// advance application state ahead of CometBFT at the same snapshot height.
	write()
	return nil
}

func testnetIndexKeys(iterator corestore.Iterator, iteratorErr error) ([][]byte, error) {
	if iteratorErr != nil {
		return nil, iteratorErr
	}

	keys := make([][]byte, 0)
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, bytes.Clone(iterator.Key()))
	}
	if err := errors.Join(iterator.Error(), iterator.Close()); err != nil {
		return nil, err
	}
	return keys, nil
}
