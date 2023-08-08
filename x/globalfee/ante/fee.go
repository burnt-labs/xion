package ante

import (
	"errors"

	tmstrings "github.com/cometbft/cometbft/libs/strings"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/burnt-labs/xion/x/globalfee"

	"github.com/burnt-labs/xion/x/globalfee/types"
)

// FeeWithBypassDecorator checks if the transaction's fee is at least as large
// as the local validator's minimum gasFee (defined in validator config) and global fee, and the fee denom should be in the global fees' denoms.
//
// If fee is too low, decorator returns error and tx is rejected from mempool.
// Note this only applies when ctx.CheckTx = true. If fee is high enough or not
// CheckTx, then call next AnteHandler.
//
// CONTRACT: Tx must implement FeeTx to use FeeDecorator
// If the tx msg type is one of the bypass msg types, the tx is valid even if the min fee is lower than normally required.
// If the bypass tx still carries fees, the fee denom should be the same as global fee required.

var _ sdk.AnteDecorator = FeeDecorator{}

type FeeDecorator struct {
	GlobalMinFeeParamSource globalfee.ParamSource
	StakingKeeperBondDenom  func(sdk.Context) string
}

func NewFeeDecorator(globalfeeSubspace paramtypes.Subspace, stakingKeeperDenom func(sdk.Context) string) FeeDecorator {
	if !globalfeeSubspace.HasKeyTable() {
		panic("global fee paramspace was not set up via module")
	}

	return FeeDecorator{
		GlobalMinFeeParamSource: globalfeeSubspace,
		StakingKeeperBondDenom:  stakingKeeperDenom,
	}
}

// AnteHandle implements the AnteDecorator interface
func (mfd FeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must implement the sdk.FeeTx interface")
	}

	// Do not check minimum-gas-prices and global fees during simulations
	if simulate {
		return next(ctx, tx, simulate)
	}

	// Get the required fees, as per xion specification max(network_fees, local_validator_fees)
	feeRequired, err := mfd.GetTxFeeRequired(ctx, feeTx)
	if err != nil {
		return ctx, err
	}

	// If message should be bypassed short-circuit, don't modify gas
	if mfd.ContainsOnlyBypassMinFeeMsgs(ctx, feeTx.GetMsgs()) {
		return next(ctx, tx, simulate)
	}
	return next(ctx.WithMinGasPrices(sdk.NewDecCoinsFromCoins(feeRequired...)), tx, simulate)
}

// GetTxFeeRequired returns the required fees for the given FeeTx.
// In case the FeeTx's mode is CheckTx, it returns the combined requirements
// of local min gas prices and global fees. Otherwise, in DeliverTx, it returns the global fee.
func (mfd FeeDecorator) GetTxFeeRequired(ctx sdk.Context, tx sdk.FeeTx) (sdk.Coins, error) {
	// Get required global fee min gas prices
	// Note that it should never be empty since its default value is set to coin={"StakingBondDenom", 0}
	globalFees, err := mfd.GetGlobalFee(ctx, tx)
	if err != nil {
		return sdk.Coins{}, err
	}

	// In DeliverTx, the global fee min gas prices are the only tx fee requirements.
	if !ctx.IsCheckTx() {
		return globalFees, nil
	}

	// In CheckTx mode, the local and global fee min gas prices are combined
	// to form the tx fee requirements

	// Get local minimum-gas-prices
	localFees := GetMinGasPrice(ctx, int64(tx.GetGas()))
	return MaxCoins(globalFees, localFees), nil
}

// GetGlobalFee returns the global fees for a given fee tx's gas
// (might also return 0denom if globalMinGasPrice is 0)
// sorted in ascending order.
// Note that ParamStoreKeyMinGasPrices type requires coins sorted.
func (mfd FeeDecorator) GetGlobalFee(ctx sdk.Context, feeTx sdk.FeeTx) (sdk.Coins, error) {
	var (
		globalMinGasPrices sdk.DecCoins
		err                error
	)

	if mfd.GlobalMinFeeParamSource.Has(ctx, types.ParamStoreKeyMinGasPrices) {
		mfd.GlobalMinFeeParamSource.Get(ctx, types.ParamStoreKeyMinGasPrices, &globalMinGasPrices)
	}
	// global fee is empty set, set global fee to 0uxion
	if len(globalMinGasPrices) == 0 {
		globalMinGasPrices, err = mfd.DefaultZeroGlobalFee(ctx)
		if err != nil {
			return sdk.Coins{}, err
		}
	}
	requiredGlobalFees := make(sdk.Coins, len(globalMinGasPrices))
	// Determine the required fees by multiplying each required minimum gas
	// price by the gas limit, where fee = ceil(minGasPrice * gasLimit).
	glDec := sdk.NewDec(int64(feeTx.GetGas()))
	for i, gp := range globalMinGasPrices {
		fee := gp.Amount.Mul(glDec)
		requiredGlobalFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
	}

	return requiredGlobalFees.Sort(), nil
}

// DefaultZeroGlobalFee returns a zero coin with the staking module bond denom
func (mfd FeeDecorator) DefaultZeroGlobalFee(ctx sdk.Context) ([]sdk.DecCoin, error) {
	bondDenom := mfd.getBondDenom(ctx)
	if bondDenom == "" {
		return nil, errors.New("empty staking bond denomination")
	}

	return []sdk.DecCoin{sdk.NewDecCoinFromDec(bondDenom, sdk.NewDec(0))}, nil
}

func (mfd FeeDecorator) getBondDenom(ctx sdk.Context) (bondDenom string) {
	return mfd.StakingKeeperBondDenom(ctx)
}

func (mfd FeeDecorator) ContainsOnlyBypassMinFeeMsgs(ctx sdk.Context, msgs []sdk.Msg) bool {
	bypassMsgTypes := mfd.GetBypassMsgTypes(ctx)
	for _, msg := range msgs {
		if tmstrings.StringInSlice(sdk.MsgTypeURL(msg), bypassMsgTypes) {
			continue
		}
		return false
	}

	return true
}

func (mfd FeeDecorator) GetBypassMsgTypes(ctx sdk.Context) (res []string) {
	if mfd.GlobalMinFeeParamSource.Has(ctx, types.ParamStoreKeyBypassMinFeeMsgTypes) {
		mfd.GlobalMinFeeParamSource.Get(ctx, types.ParamStoreKeyBypassMinFeeMsgTypes, &res)
	}

	return
}

func (mfd FeeDecorator) GetMaxTotalBypassMinFeeMsgGasUsage(ctx sdk.Context) (res uint64) {
	if mfd.GlobalMinFeeParamSource.Has(ctx, types.ParamStoreKeyMaxTotalBypassMinFeeMsgGasUsage) {
		mfd.GlobalMinFeeParamSource.Get(ctx, types.ParamStoreKeyMaxTotalBypassMinFeeMsgGasUsage, &res)
	}

	return
}

// GetMinGasPrice returns a nodes's local minimum gas prices
// fees given a gas limit
func GetMinGasPrice(ctx sdk.Context, gasLimit int64) sdk.Coins {
	minGasPrices := ctx.MinGasPrices()
	// special case: if minGasPrices=[], requiredFees=[]
	if minGasPrices.IsZero() {
		return sdk.Coins{}
	}

	requiredFees := make(sdk.Coins, len(minGasPrices))
	// Determine the required fees by multiplying each required minimum gas
	// price by the gas limit, where fee = ceil(minGasPrice * gasLimit).
	glDec := sdk.NewDec(gasLimit)
	for i, gp := range minGasPrices {
		fee := gp.Amount.Mul(glDec)
		requiredFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
	}

	return requiredFees.Sort()
}
