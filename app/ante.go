package app

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/larry0x/abstract-account/x/abstractaccount"
	aakeeper "github.com/larry0x/abstract-account/x/abstractaccount/keeper"
	feeabsante "github.com/osmosis-labs/fee-abstraction/v8/x/feeabs/ante"
	feeabskeeper "github.com/osmosis-labs/fee-abstraction/v8/x/feeabs/keeper"

	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"
	"github.com/cosmos/ibc-go/v8/modules/core/keeper"

	corestoretypes "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/posthandler"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	globalfeeante "github.com/burnt-labs/xion/x/globalfee/ante"
)

// HandlerOptions extend the SDK's AnteHandler options by requiring the IBC
// channel keeper.
type HandlerOptions struct {
	ante.HandlerOptions

	IBCKeeper             *keeper.Keeper
	WasmConfig            *wasmTypes.WasmConfig
	TXCounterStoreService corestoretypes.KVStoreService
	GlobalFeeSubspace     paramtypes.Subspace
	StakingKeeper         *stakingkeeper.Keeper
	AbstractAccountKeeper aakeeper.Keeper
	FeeAbsKeeper          *feeabskeeper.Keeper
}

func NewAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	if options.AccountKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "account keeper is required for AnteHandler")
	}
	if options.BankKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "bank keeper is required for AnteHandler")
	}
	if options.StakingKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "stakin keeper is required for AnteHandler")
	}
	if options.SignModeHandler == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "sign mode handler is required for ante builder")
	}
	if options.WasmConfig == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "wasm config is required for ante builder")
	}
	if options.GlobalFeeSubspace.Name() == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrNotFound, "globalfee param store is required for AnteHandler")
	}
	if options.TXCounterStoreService == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "wasm store service is required for ante builder")
	}
	if options.FeeAbsKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "fee abstraction keeper is required for AnteHandler")
	}

	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		wasmkeeper.NewLimitSimulationGasDecorator(options.WasmConfig.SimulationGasLimit), // after setup context to enforce limits early
		wasmkeeper.NewCountTXDecorator(options.TXCounterStoreService),
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		// this changes the minGasFees,
		// and must occur before gas fee checks
		globalfeeante.NewFeeDecorator(options.GlobalFeeSubspace, func(context sdk.Context) string {
			bondDenom, err := options.StakingKeeper.BondDenom(context)
			if err != nil {
				panic(err)
			}
			return bondDenom
		}),
		feeabsante.NewFeeAbstrationMempoolFeeDecorator(*options.FeeAbsKeeper),

		// validation checks
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		// fee abstraction fee deduction replaces sdk fee deduction
		feeabsante.NewFeeAbstractionDeductFeeDecorate(options.AccountKeeper, options.BankKeeper, *options.FeeAbsKeeper, options.FeegrantKeeper),
		ante.NewSetPubKeyDecorator(options.AccountKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		// BeforeTxDecorator replaces the default NewSigVerificationDecorator
		abstractaccount.NewBeforeTxDecorator(
			options.AbstractAccountKeeper,
			options.AccountKeeper,
			options.SignModeHandler,
		),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}

type PostHandlerOptions struct {
	posthandler.HandlerOptions

	AccountKeeper         ante.AccountKeeper
	AbstractAccountKeeper aakeeper.Keeper
}

func NewPostHandler(options PostHandlerOptions) (sdk.PostHandler, error) {
	if options.AccountKeeper == nil {
		return nil, sdkerrors.ErrLogic.Wrap("account keeper is required for AnteHandler")
	}

	postDecorators := []sdk.PostDecorator{
		abstractaccount.NewAfterTxDecorator(options.AbstractAccountKeeper),
	}

	return sdk.ChainPostDecorators(postDecorators...), nil
}
