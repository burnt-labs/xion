package ante

import (
	"errors"
	"fmt"

	tmstrings "github.com/cometbft/cometbft/libs/strings"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/burnt-labs/xion/x/globalfee"

	"github.com/burnt-labs/xion/x/globalfee/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
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
	accountKeeper           authante.AccountKeeper
	bankKeeper              authtypes.BankKeeper
	feegrantKeeper          FeegrantKeeper
	txFeeChecker            authante.TxFeeChecker
}

func NewFeeDecorator(globalfeeSubspace paramtypes.Subspace, ak authante.AccountKeeper, bk authtypes.BankKeeper, fk authante.FeegrantKeeper, tfc authante.TxFeeChecker, stakingKeeperDenom func(sdk.Context) string) FeeDecorator {
	if !globalfeeSubspace.HasKeyTable() {
		panic("global fee paramspace was not set up via module")
	}

	if tfc == nil {
		tfc = checkTxFeeWithValidatorMinGasPrices
	}

	return FeeDecorator{
		GlobalMinFeeParamSource: globalfeeSubspace,
		StakingKeeperBondDenom:  stakingKeeperDenom,
		txFeeChecker:            tfc,
		accountKeeper:           ak,
		bankKeeper:              bk,
		feegrantKeeper:          fk,
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
		return mfd.Deduct(ctx, tx, simulate, next)
	}

	// Get the required fees, as per xion specification max(network_fees, local_validator_fees)
	feeRequired, err := mfd.GetTxFeeRequired(ctx, feeTx)
	if err != nil {
		return ctx, err
	}

	// reject the transaction early if the feeCoins have more denoms than the fee requirement
	// feeRequired cannot be empty
	if feeTx.GetFee().Len() > feeRequired.Len() {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrInvalidCoins, "fee is not a subset of required fees; got %s, required: %s", feeTx.GetFee().String(), feeRequired.String())
	}

	// Sort fee tx's coins, zero coins in feeCoins are already removed
	feeCoins := feeTx.GetFee().Sort()
	gas := feeTx.GetGas()
	msgs := feeTx.GetMsgs()

	// split feeRequired into zero and non-zero coins(nonZeroCoinFeesReq, zeroCoinFeesDenomReq), split feeCoins according to
	// nonZeroCoinFeesReq, zeroCoinFeesDenomReq,
	// so that feeCoins can be checked separately against them.
	nonZeroCoinFeesReq, zeroCoinFeesDenomReq := getNonZeroFees(feeRequired)

	// feeCoinsNonZeroDenom contains non-zero denominations from the feeRequired
	// feeCoinsNonZeroDenom is used to check if the fees meets the requirement imposed by nonZeroCoinFeesReq
	// when feeCoins does not contain zero coins' denoms in feeRequired
	feeCoinsNonZeroDenom, feeCoinsZeroDenom := splitCoinsByDenoms(feeCoins, zeroCoinFeesDenomReq)

	// Check that the fees are in expected denominations.
	// according to splitCoinsByDenoms(), feeCoinsZeroDenom must be in denom subset of zeroCoinFeesDenomReq.
	// check if feeCoinsNonZeroDenom is a subset of nonZeroCoinFeesReq.
	// special case: if feeCoinsNonZeroDenom=[], DenomsSubsetOf returns true
	// special case: if feeCoinsNonZeroDenom is not empty, but nonZeroCoinFeesReq empty, return false
	if !feeCoinsNonZeroDenom.DenomsSubsetOf(nonZeroCoinFeesReq) {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "fee is not a subset of required fees; got %s, required: %s", feeCoins.String(), feeRequired.String())
	}

	// If the feeCoins pass the denoms check, check they are bypass-msg types.
	//
	// Bypass min fee requires:
	// 	- the tx contains only message types that can bypass the minimum fee,
	//	see BypassMinFeeMsgTypes;
	//	- the total gas limit per message does not exceed MaxTotalBypassMinFeeMsgGasUsage,
	//	i.e., totalGas <=  MaxTotalBypassMinFeeMsgGasUsage
	// Otherwise, minimum fees and global fees are checked to prevent spam.
	maxTotalBypassMinFeeMsgGasUsage := mfd.GetMaxTotalBypassMinFeeMsgGasUsage(ctx)
	doesNotExceedMaxGasUsage := gas <= maxTotalBypassMinFeeMsgGasUsage
	allBypassMsgs := mfd.ContainsOnlyBypassMinFeeMsgs(ctx, msgs)
	allowedToBypassMinFee := allBypassMsgs && doesNotExceedMaxGasUsage

	if allowedToBypassMinFee {
		return mfd.Deduct(ctx, tx, simulate, next)
	}

	// if the msg does not satisfy bypass condition and the feeCoins denoms are subset of feeRequired,
	// check the feeCoins amount against feeRequired
	//
	// when feeCoins=[]
	// special case: and there is zero coin in fee requirement, pass,
	// otherwise, err
	if len(feeCoins) == 0 {
		if len(zeroCoinFeesDenomReq) != 0 {
			ctx.Logger().Info("Fees are Zero\n\n")
			return mfd.Deduct(ctx, tx, simulate, next)
		}
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %s required: %s", feeCoins.String(), feeRequired.String())
	}

	// when feeCoins != []
	// special case: if TX has at least one of the zeroCoinFeesDenomReq, then it should pass
	if len(feeCoinsZeroDenom) > 0 {
		ctx.Logger().Info("Fees are greater than zero\n\n")
		return mfd.Deduct(ctx, tx, simulate, next)
	}

	// After all the checks, the tx is confirmed:
	// feeCoins denoms subset off feeRequired
	// Not bypass
	// feeCoins != []
	// Not contain zeroCoinFeesDenomReq's denoms
	//
	// check if the feeCoins's feeCoinsNonZeroDenom part has coins' amount higher/equal to nonZeroCoinFeesReq
	if !feeCoinsNonZeroDenom.IsAnyGTE(nonZeroCoinFeesReq) {
		errMsg := fmt.Sprintf("Insufficient fees; got: %s required: %s", feeCoins.String(), feeRequired.String())
		if allBypassMsgs && !doesNotExceedMaxGasUsage {
			errMsg = fmt.Sprintf("Insufficient fees; bypass-min-fee-msg-types with gas consumption %v exceeds the maximum allowed gas value of %v.", gas, maxTotalBypassMinFeeMsgGasUsage)
		}

		return ctx, sdkerrors.Wrap(sdkerrors.ErrInsufficientFee, errMsg)
	}

	return mfd.DeductFee(ctx, tx, feeRequired, simulate, next) // needs to deduct requried fee
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
		ctx.Logger().Info(fmt.Sprintf("Height: %d\nGlobalFees: %s \n", ctx.BlockHeight(), globalFees.String()))
		ctx.Logger().Info("We are ignoring local fees!!!")
		return globalFees, nil
	}

	// In CheckTx mode, the local and global fee min gas prices are combined
	// to form the tx fee requirements

	// Get local minimum-gas-prices
	localFees := GetMinGasPrice(ctx, int64(tx.GetGas()))

	ctx.Logger().Info(fmt.Sprintf("GlobalFees: %s \n", globalFees.String()))
	ctx.Logger().Info(fmt.Sprintf("LocalFees: %s \n", localFees.String()))

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

// Deduct Fee - Replaces Standard auth ante handler behavior by passing a network fee
// CONTRACT: the network fee in the xion network is determined by max(local_validator_fee, minimum_gas_param)
func (mfd FeeDecorator) DeductFee(ctx sdk.Context, tx sdk.Tx, networkFee sdk.Coins, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	feeTx, err := CheckFeeTx(ctx, tx)
	if err != nil {
		return ctx, err
	}

	ctx, err = CheckGas(ctx, feeTx, simulate)
	if err != nil {
		return ctx, err
	}

	var priority int64

	if err := mfd.checkDeductFee(ctx, tx, networkFee); err != nil {
		return ctx, err
	}

	newCtx := ctx.WithPriority(priority)

	return next(newCtx, tx, simulate)
}

// Deduct Fee - Replaces Standard auth ante handler.
func (mfd FeeDecorator) Deduct(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	feeTx, err := CheckFeeTx(ctx, tx)
	if err != nil {
		return ctx, err
	}

	ctx, err = CheckGas(ctx, feeTx, simulate)
	if err != nil {
		return ctx, err
	}

	var priority int64

	fee := feeTx.GetFee()

	if !simulate {
		fee, priority, err = mfd.txFeeChecker(ctx, tx)
		if err != nil {
			return ctx, err
		}
	}
	if err := mfd.checkDeductFee(ctx, tx, fee); err != nil {
		return ctx, err
	}

	newCtx := ctx.WithPriority(priority)

	return next(newCtx, tx, simulate)
}

func (mfd FeeDecorator) checkDeductFee(ctx sdk.Context, sdkTx sdk.Tx, fee sdk.Coins) error {
	feeTx, ok := sdkTx.(sdk.FeeTx)
	if !ok {
		return sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if addr := mfd.accountKeeper.GetModuleAddress(authtypes.FeeCollectorName); addr == nil {
		return fmt.Errorf("fee collector module account (%s) has not been set", authtypes.FeeCollectorName)
	}

	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()
	deductFeesFrom := feePayer

	// if feegranter set deduct fee from feegranter account.
	// this works with only when feegrant enabled.
	if feeGranter != nil {
		if mfd.feegrantKeeper == nil {
			return sdkerrors.ErrInvalidRequest.Wrap("fee grants are not enabled")
		} else if !feeGranter.Equals(feePayer) {
			err := mfd.feegrantKeeper.UseGrantedFees(ctx, feeGranter, feePayer, fee, sdkTx.GetMsgs())
			if err != nil {
				return sdkerrors.Wrapf(err, "%s does not allow to pay fees for %s", feeGranter, feePayer)
			}
		}

		deductFeesFrom = feeGranter
	}

	deductFeesFromAcc := mfd.accountKeeper.GetAccount(ctx, deductFeesFrom)
	if deductFeesFromAcc == nil {
		return sdkerrors.ErrUnknownAddress.Wrapf("fee payer address: %s does not exist", deductFeesFrom)
	}

	// deduct the fees
	if !fee.IsZero() {
		err := DeductFees(mfd.bankKeeper, ctx, deductFeesFromAcc, fee)
		if err != nil {
			return err
		}
	}

	events := sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeTx,
			sdk.NewAttribute(sdk.AttributeKeyFee, fee.String()),
			sdk.NewAttribute(sdk.AttributeKeyFeePayer, deductFeesFrom.String()),
		),
	}
	ctx.EventManager().EmitEvents(events)

	return nil
}
