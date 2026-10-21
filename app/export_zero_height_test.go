package app

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	cmttypes "github.com/cometbft/cometbft/types"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// TestExportZeroHeightResetsStakingState drives a zero-height export over a
// chain that has a redelegation, an unbonding delegation and a jailed
// validator, and checks that the exported genesis is the fresh-start shape:
// creation and start heights reset, and every validator outside the allowed
// list jailed.
func TestExportZeroHeightResetsStakingState(t *testing.T) {
	validators := make([]*cmttypes.Validator, 2)
	for i := range validators {
		pubKey, err := mock.NewPV().GetPubKey()
		require.NoError(t, err)
		validators[i] = cmttypes.NewValidator(pubKey, 1)
	}
	valSet := cmttypes.NewValidatorSet(validators)
	// NewValidatorSet orders by voting power then address; read the final order back.
	valAddr1 := sdk.ValAddress(valSet.Validators[0].Address)
	valAddr2 := sdk.ValAddress(valSet.Validators[1].Address)

	delegatorKey := secp256k1.GenPrivKey()
	delegator := authtypes.NewBaseAccount(delegatorKey.PubKey().Address().Bytes(), delegatorKey.PubKey(), 0, 0)
	balance := banktypes.Balance{
		Address: delegator.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100000000000000))),
	}
	gapp := SetupWithGenesisValSet(t, valSet, []authtypes.GenesisAccount{delegator}, "testing", nil, balance)
	// Commit the genesis block so the export, which reads the last committed
	// state, sees it. The staking moves below are made on that same state.
	_, err := gapp.Commit()
	require.NoError(t, err)

	ctx := gapp.NewContext(true)
	quarter := sdkmath.LegacyNewDecWithPrec(25, 2)
	_, err = gapp.StakingKeeper.BeginRedelegation(ctx, delegator.GetAddress(), valAddr1, valAddr2, quarter)
	require.NoError(t, err)
	_, _, err = gapp.StakingKeeper.Undelegate(ctx, delegator.GetAddress(), valAddr1, quarter)
	require.NoError(t, err)
	// A jailed validator leaves the power index, which is what lets the
	// export mark it jailed without tripping the validator set update.
	require.NoError(t, gapp.StakingKeeper.Jail(ctx, sdk.ConsAddress(valSet.Validators[0].Address)))

	exported, err := gapp.ExportAppStateAndValidators(true, []string{valAddr2.String()}, nil)
	require.NoError(t, err)
	require.Zero(t, exported.Height)
	require.NotEmpty(t, exported.Validators)

	var genState GenesisState
	require.NoError(t, json.Unmarshal(exported.AppState, &genState))

	var stakingGenesis stakingtypes.GenesisState
	gapp.AppCodec().MustUnmarshalJSON(genState[stakingtypes.ModuleName], &stakingGenesis)

	require.Len(t, stakingGenesis.Redelegations, 1)
	for _, entry := range stakingGenesis.Redelegations[0].Entries {
		require.Zero(t, entry.CreationHeight)
	}
	require.Len(t, stakingGenesis.UnbondingDelegations, 1)
	for _, entry := range stakingGenesis.UnbondingDelegations[0].Entries {
		require.Zero(t, entry.CreationHeight)
	}

	exportedValidators := map[string]stakingtypes.Validator{}
	for _, validator := range stakingGenesis.Validators {
		exportedValidators[validator.OperatorAddress] = validator
	}
	require.True(t, exportedValidators[valAddr1.String()].Jailed, "validator outside the allowed list must be jailed")
	allowed := exportedValidators[valAddr2.String()]
	require.False(t, allowed.Jailed, "allowed validator must stay unjailed")
	require.Zero(t, allowed.UnbondingHeight)
	require.Equal(t, stakingtypes.Bonded, allowed.Status)

	var slashingGenesis slashingtypes.GenesisState
	gapp.AppCodec().MustUnmarshalJSON(genState[slashingtypes.ModuleName], &slashingGenesis)
	require.Len(t, slashingGenesis.SigningInfos, 2)
	for _, info := range slashingGenesis.SigningInfos {
		require.Zero(t, info.ValidatorSigningInfo.StartHeight)
	}
}

func TestExportRejectsUnknownModule(t *testing.T) {
	gapp := Setup(t)

	_, err := gapp.ExportAppStateAndValidators(false, nil, []string{"no-such-module"})
	require.ErrorContains(t, err, "module no-such-module does not exist")
}
