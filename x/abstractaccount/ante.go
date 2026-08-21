package abstractaccount

import (
	"encoding/json"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"cosmossdk.io/core/gas"
	"cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	txsign "cosmossdk.io/x/tx/signing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	txsigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/burnt-labs/xion/x/abstractaccount/keeper"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

var (
	_ sdk.AnteDecorator = &BeforeTxDecorator{}
	_ sdk.PostDecorator = &AfterTxDecorator{}
)

// -------------------------------- GasComsumer --------------------------------

func SigVerificationGasConsumer(
	meter storetypes.GasMeter, sig txsigning.SignatureV2, params authtypes.Params,
) error {
	// If the pubkey is a NilPubKey, for now we do not consume any gas (the
	// contract execution consumes it)
	// Otherwise, we simply delegate to the default consumer
	switch sig.PubKey.(type) {
	case *types.NilPubKey:
		return nil
	default:
		return authante.DefaultSigVerificationGasConsumer(meter, sig, params)
	}
}

// --------------------------------- BeforeTx ----------------------------------

type BeforeTxDecorator struct {
	aak             keeper.Keeper
	ak              authante.AccountKeeper
	signModeHandler *txsign.HandlerMap
}

func NewBeforeTxDecorator(aak keeper.Keeper, ak authante.AccountKeeper, signModeHandler *txsign.HandlerMap) BeforeTxDecorator {
	return BeforeTxDecorator{aak, ak, signModeHandler}
}

func (d BeforeTxDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// first we need to determine whether the rules of account abstraction should
	// apply to this tx. there are two criteria:
	//
	// - the tx has exactly one signer and one signature
	// - this one signer is an AbstractAccount
	//
	// both criteria must be satisfied for this be to be qualified as an AA tx.
	isAbstractAccountTx, signerAcc, sig, err := IsAbstractAccountTx(ctx, tx, d.ak)
	if err != nil {
		return ctx, err
	}

	// if the tx isn't an AA tx, we simply delegate the ante task to the default
	// SigVerificationDecorator
	if !isAbstractAccountTx {
		svd := authante.NewSigVerificationDecorator(d.ak, d.signModeHandler)
		return svd.AnteHandle(ctx, tx, simulate, next)
	}

	// handle the AA transaction validation and contract invocation
	if err := d.handleAATransaction(ctx, tx, signerAcc, sig, simulate); err != nil {
		return ctx, err
	}

	return next(ctx, tx, simulate)
}

// handleAATransaction processes an AbstractAccount transaction by validating
// the context, checking sequence/nonce, and invoking the before_tx handler.
func (d BeforeTxDecorator) handleAATransaction(
	ctx sdk.Context, tx sdk.Tx, signerAcc *types.AbstractAccount,
	sig *txsigning.SignatureV2, simulate bool,
) error {
	if ctx.BlockTime().UnixNano() <= 0 {
		return types.ErrNoBlockTime.Wrapf("expected a positive block time, received %d", ctx.BlockTime().UnixNano())
	}

	// save the account address to the module store. we will need it in the
	// posthandler
	d.aak.SetSignerAddress(ctx, signerAcc.GetAddress())

	// Determine whether this is an unordered AA transaction. Unordered txs use
	// a per-tx nonce + timeout_timestamp for replay protection instead of the
	// monotonic account sequence, so the sequence check and nonce handling
	// must be branched here.
	utx, isUnordered := tx.(sdk.TxWithUnordered)
	if isUnordered && utx.GetUnordered() {
		if err := d.handleUnorderedTx(ctx, tx, signerAcc, sig, simulate); err != nil {
			return err
		}
	} else {
		// check account sequence number for ordered transactions
		if sig.Sequence != signerAcc.GetSequence() {
			return sdkerrors.ErrWrongSequence.Wrapf("account sequence mismatch, expected %d, got %d", signerAcc.GetSequence(), sig.Sequence)
		}
	}

	// invoke the account contract's before_tx handler
	return d.invokeBeforeTx(ctx, tx, signerAcc, simulate)
}

// invokeBeforeTx prepares the SudoMsg and invokes the account contract's before_tx handler.
func (d BeforeTxDecorator) invokeBeforeTx(
	ctx sdk.Context, tx sdk.Tx, signerAcc *types.AbstractAccount, simulate bool,
) error {
	// prepare the messages in the tx, converted to []Any
	msgAnys, err := SdkMsgsToAnys(tx.GetMsgs())
	if err != nil {
		return err
	}

	// prepare the sign bytes and credential
	// logics here are mostly copied over from the SigVerificationDecorator.
	sigs, err := tx.(authsigning.SigVerifiableTx).GetSignaturesV2()
	if err != nil {
		return err
	}
	signBytes, sigBytes, err := prepareCredentials(ctx, tx, signerAcc, &sigs[0], d.signModeHandler)
	if err != nil {
		return err
	}

	sudoMsgBytes, err := json.Marshal(&types.AccountSudoMsg{
		BeforeTx: &types.BeforeTx{
			Msgs:    msgAnys,
			TxBytes: signBytes,
			// Note that we call this field "cred_bytes" (credential bytes) instead of
			// signature. There is an important reason for this!
			//
			// For EOAs, the credential used to prove a tx is authenticated is a
			// cryptographic signature. For AbstractAccounts however, this is not
			// necessarily the case. The account contract can be programmed to take
			// any credential, not limited to cryptographic signatures. An example of
			// this can be a zk proof that the sender has undergone certain KYC
			// procedures. Therefore, instead of calling this "signature", we choose a
			// more generalized term: credentials.
			CredBytes: sigBytes,
			Simulate:  simulate,
		},
	})
	if err != nil {
		return err
	}

	params, err := d.aak.GetParams(ctx)
	if err != nil {
		return err
	}

	return sudoWithGasLimit(ctx, d.aak.ContractKeeper(), signerAcc.GetAddress(), sudoMsgBytes, params.MaxGasBefore)
}

// DefaultMaxUnorderedTTL is the default maximum time-to-live for unordered transactions.
// Mirrors the value used by SigVerificationDecorator in the Cosmos SDK.
const DefaultMaxUnorderedTTL = 10 * time.Minute

// DefaultUnorderedTxGasCost is the extra gas charged for registering an
// unordered transaction nonce (store read + write).
// Mirrors the DefaultUnorderedTxGasCost constant in the Cosmos SDK.
const DefaultUnorderedTxGasCost = uint64(2240)

// handleUnorderedTx enforces replay-protection for unordered AbstractAccount
// transactions.
//
// BeforeTxDecorator replaces SigVerificationDecorator for AA txs, which means
// the unordered-nonce checks that SigVerificationDecorator normally performs
// are otherwise skipped. This method restores those invariants:
//
//  1. Unordered transactions must be enabled on the chain.
//  2. The sig.Sequence field must be zero (not applicable for unordered mode).
//  3. timeout_timestamp must be present, non-expired, and within the max TTL.
//  4. The nonce (sender + timeout_timestamp) is marked as used so the same tx
//     cannot be replayed within the timeout window.
func (d BeforeTxDecorator) handleUnorderedTx(ctx sdk.Context, tx sdk.Tx, signerAcc sdk.AccountI, sig *txsigning.SignatureV2, simulate bool) error {
	if !d.ak.UnorderedTransactionsEnabled() {
		return sdkerrors.ErrNotSupported.Wrap("unordered transactions are not enabled")
	}

	// For unordered txs the sequence field in the signature must be zero.
	if sig.Sequence > 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("sequence is not allowed for unordered transactions")
	}

	utx := tx.(sdk.TxWithUnordered)
	blockTime := ctx.BlockTime()
	timeoutTimestamp := utx.GetTimeoutTimeStamp()

	// Validate timeout_timestamp is set
	if timeoutTimestamp.IsZero() || timeoutTimestamp.Unix() == 0 {
		return errors.Wrap(
			sdkerrors.ErrInvalidRequest,
			"unordered transaction must have timeout_timestamp set",
		)
	}

	// Validate timeout has not already passed
	if timeoutTimestamp.Before(blockTime) {
		return errors.Wrap(
			sdkerrors.ErrInvalidRequest,
			"unordered transaction has a timeout_timestamp that has already passed",
		)
	}

	// Validate TTL does not exceed maximum
	if timeoutTimestamp.After(blockTime.Add(DefaultMaxUnorderedTTL)) {
		return errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"unordered tx ttl exceeds %s",
			DefaultMaxUnorderedTTL.String(),
		)
	}

	// Charge gas for the nonce store operations.
	ctx.GasMeter().ConsumeGas(DefaultUnorderedTxGasCost, "unordered tx nonce")

	// Skip state-mutating nonce recording during simulation so that simulated
	// transactions don't consume nonce slots.
	if simulate || ctx.ExecMode() == sdk.ExecModeSimulate {
		return nil
	}

	// Record the unordered nonce to prevent replay attacks.
	// The nonce is the combination of (sender address, timeout_timestamp).
	if err := d.ak.TryAddUnorderedNonce(ctx, signerAcc.GetAddress(), timeoutTimestamp); err != nil {
		return errors.Wrapf(
			sdkerrors.ErrInvalidRequest,
			"failed to add unordered nonce: %s", err,
		)
	}

	return nil
}

// ---------------------------------- AfterTx ----------------------------------

type AfterTxDecorator struct {
	aak keeper.Keeper
}

func NewAfterTxDecorator(aak keeper.Keeper) AfterTxDecorator {
	return AfterTxDecorator{aak}
}

func (d AfterTxDecorator) PostHandle(ctx sdk.Context, tx sdk.Tx, simulate, success bool, next sdk.PostHandler) (newCtx sdk.Context, err error) {
	// load the signer address, which we determined during the AnteHandler
	//
	// if not found, it means this tx is simply not an AA tx. we skip
	signerAddr := d.aak.GetSignerAddress(ctx)
	if signerAddr == nil {
		return next(ctx, tx, simulate, success)
	}

	d.aak.DeleteSignerAddress(ctx)

	sudoMsgBytes, err := json.Marshal(&types.AccountSudoMsg{
		AfterTx: &types.AfterTx{
			Simulate: simulate,
			// we don't need to pass the `success` parameter into the contract,
			// because the Posthandler is only executed if the tx is successful, so it
			// should always be true anyways
		},
	})
	if err != nil {
		return ctx, err
	}

	params, err := d.aak.GetParams(ctx)
	if err != nil {
		return ctx, err
	}

	if err := sudoWithGasLimit(ctx, d.aak.ContractKeeper(), signerAddr, sudoMsgBytes, params.MaxGasAfter); err != nil {
		return ctx, err
	}

	return next(ctx, tx, simulate, success)
}

// ---------------------------------- Helpers ----------------------------------

func IsAbstractAccountTx(ctx sdk.Context, tx sdk.Tx, ak authante.AccountKeeper) (bool, *types.AbstractAccount, *txsigning.SignatureV2, error) {
	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return false, nil, nil, errors.Wrap(sdkerrors.ErrTxDecode, "tx is not a SigVerifiableTx")
	}

	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		return false, nil, nil, err
	}

	signerAddrs, err := sigTx.GetSigners()
	if err != nil {
		return false, nil, nil, err
	}
	if len(signerAddrs) != 1 || len(sigs) != 1 {
		return false, nil, nil, nil
	}

	signerAcc, err := authante.GetSignerAcc(ctx, ak, signerAddrs[0])
	if err != nil {
		return false, nil, nil, err
	}

	absAcc, ok := signerAcc.(*types.AbstractAccount)
	if !ok {
		return false, nil, nil, nil
	}

	return true, absAcc, &sigs[0], nil
}

func prepareCredentials(
	ctx sdk.Context, tx sdk.Tx, signerAcc sdk.AccountI,
	sig *txsigning.SignatureV2, handler *txsign.HandlerMap,
) ([]byte, []byte, error) {
	signerData := authsigning.SignerData{
		ChainID:       ctx.ChainID(),
		AccountNumber: signerAcc.GetAccountNumber(),
		// Use the sequence carried by the signature, matching the SDK's
		// SigVerificationDecorator. For ordered txs this equals the account
		// sequence (enforced in handleAATransaction); for unordered txs it is
		// always zero (enforced in handleUnorderedTx) even after the account
		// sequence has advanced, and the sign bytes must reproduce what the
		// client actually signed.
		Sequence: sig.Sequence,
		PubKey:   signerAcc.GetPubKey(),
		Address:  signerAcc.GetAddress().String(),
	}

	data, ok := sig.Data.(*txsigning.SingleSignatureData)
	if !ok {
		return nil, nil, types.ErrNotSingleSignature
	}

	signBytes, err := authsigning.GetSignBytesAdapter(ctx, handler, data.SignMode, signerData, tx)
	if err != nil {
		return nil, nil, err
	}

	return signBytes, data.Signature, nil
}

func SdkMsgsToAnys(msgs []sdk.Msg) ([]*types.Any, error) {
	anys := []*types.Any{}

	for _, msg := range msgs {
		msgAny, err := types.NewAnyFromProtoMsg(msg)
		if err != nil {
			return nil, err
		}

		anys = append(anys, msgAny)
	}

	return anys, nil
}

// Call a contract's sudo entry point with a gas limit.
//
// Copied from Osmosis' protorev posthandler:
// https://github.com/osmosis-labs/osmosis/blob/98025f185ab2ee1b060511ed22679112abcc08fa/x/protorev/keeper/posthandler.go#L42-L43
//
// Thanks Roman and Jorge for the helpful discussion.
func sudoWithGasLimit(
	ctx sdk.Context, contractKeeper wasmtypes.ContractOpsKeeper,
	contractAddr sdk.AccAddress, msg []byte, maxGas gas.Gas,
) error {
	// Cap the child budget by the gas remaining in the tx: any work beyond
	// that would fail the tx when the parent meter is charged below, so there
	// is no reason to let the contract burn it first.
	if remaining := ctx.GasMeter().GasRemaining(); remaining < maxGas {
		maxGas = remaining
	}

	cacheCtx, write := ctx.CacheContext()
	cacheCtx = cacheCtx.WithGasMeter(storetypes.NewGasMeter(maxGas))

	// Charge the parent meter for the child's consumption on every exit —
	// success, contract error, and out-of-gas panic alike. Without this a tx
	// that fails in the sudo call would have executed contract code that no
	// gas meter ever accounted for.
	defer func() {
		ctx.GasMeter().ConsumeGas(cacheCtx.GasMeter().GasConsumed(), "sudoWithGasLimit")
	}()

	if _, err := contractKeeper.Sudo(cacheCtx, contractAddr, msg); err != nil {
		return err
	}

	write()
	// EmitEvents method is deprecated in favor EmitTypedEvent
	// however, here we're not creating events ourselves, but rather just
	// forwarding events emitted by another process (contractKeeper.Sudo)
	// so we have to stick with the legacy EmitEvents here.
	ctx.EventManager().EmitEvents(cacheCtx.EventManager().Events())

	return nil
}
