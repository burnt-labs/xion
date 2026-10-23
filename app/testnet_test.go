package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtcrypto "github.com/cometbft/cometbft/crypto"
	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cryptoenc "github.com/cometbft/cometbft/crypto/encoding"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const expectedTestnetValidatorPower int64 = 900_000_000_000_000

func TestInitForTestnetReplacesValidatorInfrastructureAndSchedulesUpgrade(t *testing.T) {
	wasmApp := setupMultiValidatorTestnetApp(t, 3)
	snapshotHeight := wasmApp.LastBlockHeight()
	chainID := "xion-rehearsal-1"
	pubKey := cmted25519.GenPrivKey().PubKey()
	operator := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	beforeCtx := testnetContext(wasmApp, chainID)
	beforePowerValidators, err := wasmApp.StakingKeeper.GetBondedValidatorsByPower(beforeCtx)
	require.NoError(t, err)
	require.Len(t, beforePowerValidators, 3)
	beforeLastValidators, err := wasmApp.StakingKeeper.GetLastValidators(beforeCtx)
	require.NoError(t, err)
	require.Len(t, beforeLastValidators, 3)

	require.NoError(t, wasmApp.InitForTestnet(pubKey, operator, chainID, "v32"))

	ctx := testnetContext(wasmApp, chainID)
	operatorBytes, err := wasmApp.AccountKeeper.AddressCodec().StringToBytes(operator)
	require.NoError(t, err)
	expectedOperator, err := wasmApp.StakingKeeper.ValidatorAddressCodec().BytesToString(operatorBytes)
	require.NoError(t, err)
	consAddr := sdk.ConsAddress(pubKey.Address())

	replacement, err := wasmApp.StakingKeeper.GetValidatorByConsAddr(ctx, consAddr)
	require.NoError(t, err)
	require.Equal(t, expectedOperator, replacement.OperatorAddress)
	require.Equal(t, expectedTestnetValidatorPower, replacement.ConsensusPower(wasmApp.StakingKeeper.PowerReduction(ctx)))
	allValidators, err := wasmApp.StakingKeeper.GetAllValidators(ctx)
	require.NoError(t, err)
	require.Len(t, allValidators, 4)

	powerValidators, err := wasmApp.StakingKeeper.GetBondedValidatorsByPower(ctx)
	require.NoError(t, err)
	require.Len(t, powerValidators, 1)
	require.Equal(t, expectedOperator, powerValidators[0].OperatorAddress)

	lastValidators, err := wasmApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	require.Len(t, lastValidators, 1)
	require.Equal(t, expectedOperator, lastValidators[0].OperatorAddress)

	signingInfo, err := wasmApp.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
	require.NoError(t, err)
	require.Equal(t, consAddr.String(), signingInfo.Address)
	require.Equal(t, snapshotHeight-1, signingInfo.StartHeight)

	replacementPubKey, err := replacement.ConsPubKey()
	require.NoError(t, err)
	trackedPubKey, err := wasmApp.SlashingKeeper.GetPubkey(ctx, replacementPubKey.Address())
	require.NoError(t, err)
	require.Equal(t, replacementPubKey.Bytes(), trackedPubKey.Bytes())

	valAddr := sdk.ValAddress(operatorBytes)
	lastPower, err := wasmApp.StakingKeeper.GetLastValidatorPower(ctx, valAddr)
	require.NoError(t, err)
	require.Zero(t, lastPower)
	lastTotalPower, err := wasmApp.StakingKeeper.GetLastTotalPower(ctx)
	require.NoError(t, err)
	require.True(t, lastTotalPower.IsZero())

	historicalRewards, err := wasmApp.DistrKeeper.GetValidatorHistoricalRewards(ctx, valAddr, 0)
	require.NoError(t, err)
	require.Equal(t, uint32(1), historicalRewards.ReferenceCount)
	currentRewards, err := wasmApp.DistrKeeper.GetValidatorCurrentRewards(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t, uint64(1), currentRewards.Period)
	_, err = wasmApp.DistrKeeper.GetValidatorOutstandingRewards(ctx, valAddr)
	require.NoError(t, err)
	govParams, err := wasmApp.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)
	bondDenom, err := wasmApp.StakingKeeper.BondDenom(ctx)
	require.NoError(t, err)
	operatorBalance := wasmApp.BankKeeper.GetBalance(ctx, sdk.AccAddress(operatorBytes), bondDenom)
	for _, deposit := range govParams.MinDeposit {
		if deposit.Denom == bondDenom {
			require.True(t, operatorBalance.Amount.GTE(deposit.Amount), "operator must be able to deposit a governance proposal")
		}
	}
	require.Equal(t, 20*time.Second, *govParams.VotingPeriod)
	require.Equal(t, 10*time.Second, *govParams.ExpeditedVotingPeriod)

	plan, err := wasmApp.UpgradeKeeper.GetUpgradePlan(ctx)
	require.NoError(t, err)
	require.Equal(t, "v32", plan.Name)
	require.Equal(t, snapshotHeight+10, plan.Height)
}

// After conversion the next block is decided by the local validator alone.
// Every begin and end blocker must accept that block, and the only validator
// update CometBFT receives is the local validator at the testnet power, which
// matches the consensus set the command writes. Without the staking,
// distribution and slashing records this block halts the chain.
func TestInitForTestnetFirstBlocksCommit(t *testing.T) {
	wasmApp := setupMultiValidatorTestnetApp(t, 3)
	_, err := wasmApp.Commit()
	require.NoError(t, err)
	snapshotHeight := wasmApp.LastBlockHeight()
	require.Positive(t, snapshotHeight)

	chainID := "xion-rehearsal-1"
	pubKey := cmted25519.GenPrivKey().PubKey()
	operator := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	require.NoError(t, wasmApp.InitForTestnet(pubKey, operator, chainID, ""))

	localVote := abci.VoteInfo{
		Validator:   abci.Validator{Address: pubKey.Address(), Power: expectedTestnetValidatorPower},
		BlockIdFlag: tmproto.BlockIDFlagCommit,
	}
	newValidatorSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{
		cmttypes.NewValidator(pubKey, expectedTestnetValidatorPower),
	})
	blockTime := time.Now().UTC()
	finalizeBlock := func(height int64) *abci.ResponseFinalizeBlock {
		t.Helper()
		blockTime = blockTime.Add(5 * time.Second)
		response, err := wasmApp.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height:             height,
			Time:               blockTime,
			Hash:               wasmApp.LastCommitID().Hash,
			ProposerAddress:    pubKey.Address(),
			NextValidatorsHash: newValidatorSet.Hash(),
			DecidedLastCommit:  abci.CommitInfo{Round: 0, Votes: []abci.VoteInfo{localVote}},
		})
		require.NoError(t, err)
		_, err = wasmApp.Commit()
		require.NoError(t, err)
		require.Equal(t, height, wasmApp.LastBlockHeight())
		return response
	}

	first := finalizeBlock(snapshotHeight + 1)
	expectedPubKey, err := cryptoenc.PubKeyToProto(pubKey)
	require.NoError(t, err)
	require.Equal(t, []abci.ValidatorUpdate{{
		PubKey: expectedPubKey,
		Power:  expectedTestnetValidatorPower,
	}}, first.ValidatorUpdates)

	second := finalizeBlock(snapshotHeight + 2)
	require.Empty(t, second.ValidatorUpdates, "the committed validator set must already match consensus")

	ctx := testnetContext(wasmApp, chainID)
	lastValidators, err := wasmApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	require.Len(t, lastValidators, 1)
	lastValidatorAddress, err := wasmApp.StakingKeeper.ValidatorAddressCodec().StringToBytes(lastValidators[0].GetOperator())
	require.NoError(t, err)
	localValidator, err := wasmApp.StakingKeeper.GetValidatorByConsAddr(ctx, sdk.ConsAddress(pubKey.Address()))
	require.NoError(t, err)
	require.Equal(t, localValidator.OperatorAddress, lastValidators[0].OperatorAddress)
	lastPower, err := wasmApp.StakingKeeper.GetLastValidatorPower(ctx, lastValidatorAddress)
	require.NoError(t, err)
	require.Equal(t, expectedTestnetValidatorPower, lastPower)
	signingInfo, err := wasmApp.SlashingKeeper.GetValidatorSigningInfo(ctx, sdk.ConsAddress(pubKey.Address()))
	require.NoError(t, err)
	require.Zero(t, signingInfo.MissedBlocksCounter)
	require.False(t, signingInfo.Tombstoned)
}

func TestInitForTestnetPreservesUnrelatedParamsAndModuleVersions(t *testing.T) {
	wasmApp := setupMultiValidatorTestnetApp(t, 2)
	chainID := "xion-rehearsal-1"
	ctx := testnetContext(wasmApp, chainID)

	bankParams := wasmApp.BankKeeper.GetParams(ctx)
	bankParams.DefaultSendEnabled = false
	require.NoError(t, wasmApp.BankKeeper.SetParams(ctx, bankParams))

	versions, err := wasmApp.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	versions["testnet-fixture"] = 99
	require.NoError(t, wasmApp.UpgradeKeeper.SetModuleVersionMap(ctx, versions))

	pubKey := cmted25519.GenPrivKey().PubKey()
	operator := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	require.NoError(t, wasmApp.InitForTestnet(pubKey, operator, chainID, ""))

	afterCtx := testnetContext(wasmApp, chainID)
	require.Equal(t, bankParams, wasmApp.BankKeeper.GetParams(afterCtx))
	afterVersions, err := wasmApp.UpgradeKeeper.GetModuleVersionMap(afterCtx)
	require.NoError(t, err)
	require.Equal(t, versions, afterVersions)
	_, err = wasmApp.UpgradeKeeper.GetUpgradePlan(afterCtx)
	require.ErrorIs(t, err, upgradetypes.ErrNoUpgradePlanFound)
}

func TestInitForTestnetInvalidInputLeavesStateUnmutated(t *testing.T) {
	tests := []struct {
		name       string
		mutateArgs func(cmtcrypto.PubKey, string, string) (cmtcrypto.PubKey, string, string)
	}{
		{
			name: "missing public key",
			mutateArgs: func(_ cmtcrypto.PubKey, operator, chainID string) (cmtcrypto.PubKey, string, string) {
				return nil, operator, chainID
			},
		},
		{
			name: "malformed operator",
			mutateArgs: func(pubKey cmtcrypto.PubKey, _ string, chainID string) (cmtcrypto.PubKey, string, string) {
				return pubKey, "not-an-address", chainID
			},
		},
		{
			name: "empty chain ID",
			mutateArgs: func(pubKey cmtcrypto.PubKey, operator, _ string) (cmtcrypto.PubKey, string, string) {
				return pubKey, operator, ""
			},
		},
		{
			name: "mainnet chain ID",
			mutateArgs: func(pubKey cmtcrypto.PubKey, operator, _ string) (cmtcrypto.PubKey, string, string) {
				return pubKey, operator, "xion-mainnet-1"
			},
		},
		{
			name: "testnet chain ID",
			mutateArgs: func(pubKey cmtcrypto.PubKey, operator, _ string) (cmtcrypto.PubKey, string, string) {
				return pubKey, operator, "xion-testnet-2"
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wasmApp := setupMultiValidatorTestnetApp(t, 2)
			chainID := "xion-rehearsal-1"
			pubKey := cmted25519.GenPrivKey().PubKey()
			operator := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
			pubKey, operator, chainID = tc.mutateArgs(pubKey, operator, chainID)

			before := captureTestnetState(t, wasmApp, "source-chain")
			err := wasmApp.InitForTestnet(pubKey, operator, chainID, "v32")
			require.Error(t, err)
			after := captureTestnetState(t, wasmApp, "source-chain")
			require.Equal(t, before, after)

			_, err = wasmApp.UpgradeKeeper.GetUpgradePlan(testnetContext(wasmApp, "source-chain"))
			require.ErrorIs(t, err, upgradetypes.ErrNoUpgradePlanFound)
		})
	}
}

type testnetStateSnapshot struct {
	validators      []stakingtypes.Validator
	powerValidators []stakingtypes.Validator
	lastValidators  []stakingtypes.Validator
	lastTotalPower  sdkmath.Int
	bondSupply      sdk.Coin
	bankParams      banktypes.Params
	moduleVersions  module.VersionMap
}

func captureTestnetState(t *testing.T, wasmApp *WasmApp, chainID string) testnetStateSnapshot {
	t.Helper()

	ctx := testnetContext(wasmApp, chainID)
	validators, err := wasmApp.StakingKeeper.GetAllValidators(ctx)
	require.NoError(t, err)
	powerValidators, err := wasmApp.StakingKeeper.GetBondedValidatorsByPower(ctx)
	require.NoError(t, err)
	lastValidators, err := wasmApp.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)
	lastTotalPower, err := wasmApp.StakingKeeper.GetLastTotalPower(ctx)
	require.NoError(t, err)
	moduleVersions, err := wasmApp.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)

	return testnetStateSnapshot{
		validators:      validators,
		powerValidators: powerValidators,
		lastValidators:  lastValidators,
		lastTotalPower:  lastTotalPower,
		bondSupply:      wasmApp.BankKeeper.GetSupply(ctx, sdk.DefaultBondDenom),
		bankParams:      wasmApp.BankKeeper.GetParams(ctx),
		moduleVersions:  moduleVersions,
	}
}

func testnetContext(wasmApp *WasmApp, chainID string) sdk.Context {
	return wasmApp.NewUncachedContext(false, tmproto.Header{
		Height:  wasmApp.LastBlockHeight(),
		ChainID: chainID,
	})
}

func setupMultiValidatorTestnetApp(t *testing.T, validatorCount int) *WasmApp {
	t.Helper()

	validators := make([]*cmttypes.Validator, validatorCount)
	for i := range validators {
		validators[i] = cmttypes.NewValidator(cmted25519.GenPrivKey().PubKey(), 1)
	}
	validatorSet := cmttypes.NewValidatorSet(validators)

	accountKey := secp256k1.GenPrivKey()
	account := authtypes.NewBaseAccount(accountKey.PubKey().Address().Bytes(), accountKey.PubKey(), 0, 0)
	balance := banktypes.Balance{
		Address: account.GetAddress().String(),
		Coins: sdk.NewCoins(sdk.NewCoin(
			sdk.DefaultBondDenom,
			sdkmath.NewInt(100_000_000_000_000),
		)),
	}

	return SetupWithGenesisValSet(
		t,
		validatorSet,
		[]authtypes.GenesisAccount{account},
		"source-chain",
		nil,
		balance,
	)
}

func TestValidateTestnetChainID(t *testing.T) {
	tests := []struct {
		name    string
		chainID string
		source  string
		wantErr string
	}{
		{name: "fresh name", chainID: "xion-rehearsal-1", source: "xion-mainnet-1"},
		{name: "fresh name from testnet", chainID: "rehearsal-1", source: "xion-testnet-2"},
		{name: "fresh name without source", chainID: "xion-rehearsal-1"},
		{name: "name containing a public family", chainID: "xion-mainnet-fork-1", source: "xion-mainnet-1"},
		{name: "local devnet family", chainID: "xion-local-testnet-1", source: "xion-mainnet-1"},
		{name: "empty", chainID: " ", source: "xion-mainnet-1", wantErr: "testnet chain ID is required"},
		{name: "mainnet", chainID: "xion-mainnet-1", source: "xion-testnet-2", wantErr: "reuses the public network name xion-mainnet-N"},
		{name: "testnet", chainID: "xion-testnet-2", source: "xion-mainnet-1", wantErr: "reuses the public network name xion-testnet-N"},
		{name: "mainnet without source", chainID: "xion-mainnet-1", wantErr: "reuses the public network name xion-mainnet-N"},
		{name: "mainnet other revision", chainID: "xion-mainnet-2", source: "xion-mainnet-1", wantErr: "reuses the public network name xion-mainnet-N"},
		{name: "retired testnet revision", chainID: "xion-testnet-1", source: "xion-testnet-2", wantErr: "reuses the public network name xion-testnet-N"},
		{name: "mainnet without revision", chainID: "xion-mainnet", source: "xion-testnet-2", wantErr: "reuses the public network name xion-mainnet-N"},
		{name: "mainnet different case", chainID: "XION-Mainnet-1", source: "xion-testnet-2", wantErr: "reuses the public network name xion-mainnet-N"},
		{name: "same as source", chainID: "xion-devnet-1", source: "xion-devnet-1", wantErr: "must differ from source chain ID"},
		{name: "source other revision", chainID: "xion-devnet-2", source: "xion-devnet-1", wantErr: "is a revision of source chain ID"},
		{name: "source without revision", chainID: "localxion-2", source: "localxion", wantErr: "is a revision of source chain ID"},
		{name: "source revision dropped", chainID: "xion-devnet", source: "xion-devnet-1", wantErr: "is a revision of source chain ID"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTestnetChainID(tc.chainID, tc.source)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
