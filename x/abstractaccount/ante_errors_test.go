package abstractaccount_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	txsigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	xionapp "github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/x/abstractaccount"
	"github.com/burnt-labs/xion/x/abstractaccount/testdata"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

// The tests in this file drive the ante and post decorators through the
// branches the happy-path tests do not reach: delegation for ordinary
// accounts, rejected transactions, missing module state and the gas cap on
// contract calls.

func deleteModuleParams(app *xionapp.WasmApp, ctx sdk.Context) {
	ctx.KVStore(app.GetKey(types.StoreKey)).Delete(types.KeyParams)
}

// ownerKey names the keyring entry that controls the abstract account
// registered by registeredAbstractAccount.
const ownerKey = "owner"

// registeredAbstractAccount registers an abstract account controlled by the
// ownerKey key and returns it along with a plain recipient account.
func registeredAbstractAccount(t *testing.T, app *xionapp.WasmApp, ctx sdk.Context, keybase keyring.Keyring) (*types.AbstractAccount, sdk.AccountI) {
	t.Helper()

	owner, err := makeMockAccount(app, ctx, keybase, ownerKey)
	require.NoError(t, err)
	absAcc, err := storeCodeAndRegisterAccount(ctx, app, owner.GetAddress(), testdata.AccountWasm,
		&BaseInstantiateMsg{PubKey: owner.GetPubKey().Bytes()}, sdk.NewCoins())
	require.NoError(t, err)

	recipient, err := makeMockAccount(app, ctx, keybase, "recipient")
	require.NoError(t, err)

	return absAcc, recipient
}

// ---------------------------- Fake transactions ---------------------------

// plainTx is a transaction that is not signature verifiable.
type plainTx struct{}

func (plainTx) GetMsgs() []sdk.Msg                    { return nil }
func (plainTx) GetMsgsV2() ([]protov2.Message, error) { return nil, nil }

// sigErrTx fails to report its signatures.
type sigErrTx struct {
	plainTx
}

func (sigErrTx) GetSigners() ([][]byte, error)             { return nil, nil }
func (sigErrTx) GetPubKeys() ([]cryptotypes.PubKey, error) { return nil, nil }
func (sigErrTx) GetSignaturesV2() ([]txsigning.SignatureV2, error) {
	return nil, errors.New("signatures unavailable")
}

// unencodableMsg is a message whose protobuf encoding fails.
type unencodableMsg struct{}

func (*unencodableMsg) Reset()                   {}
func (*unencodableMsg) String() string           { return "unencodable" }
func (*unencodableMsg) ProtoMessage()            {}
func (*unencodableMsg) Marshal() ([]byte, error) { return nil, errors.New("cannot encode message") }

// unencodableMsgTx is an abstract account transaction carrying a message
// that cannot be handed to the account contract.
type unencodableMsgTx struct {
	plainTx
	signer sdk.AccAddress
}

func (unencodableMsgTx) GetMsgs() []sdk.Msg                        { return []sdk.Msg{&unencodableMsg{}} }
func (tx unencodableMsgTx) GetSigners() ([][]byte, error)          { return [][]byte{tx.signer}, nil }
func (unencodableMsgTx) GetPubKeys() ([]cryptotypes.PubKey, error) { return nil, nil }
func (unencodableMsgTx) GetSignaturesV2() ([]txsigning.SignatureV2, error) {
	return []txsigning.SignatureV2{{
		Data: &txsigning.SingleSignatureData{SignMode: signMode, Signature: []byte("credential")},
	}}, nil
}

// orderedOnlyAccountKeeper is the app's account keeper with unordered
// transactions switched off.
type orderedOnlyAccountKeeper struct {
	authkeeper.AccountKeeper
}

func (orderedOnlyAccountKeeper) UnorderedTransactionsEnabled() bool { return false }

// -------------------------------- BeforeTx --------------------------------

func TestBeforeTx_DelegatesNonAbstractAccountTx(t *testing.T) {
	app := xionapp.Setup(t)
	keybase := keyring.NewInMemory(app.AppCodec())
	// Past genesis, so the default verification signs over the account number.
	ctx := app.NewContext(false).WithBlockHeight(1).WithBlockTime(time.Now()).WithChainID(mockChainID)

	acc, err := makeMockAccount(app, ctx, keybase, "plain")
	require.NoError(t, err)
	app.AccountKeeper.SetAccount(ctx, acc)
	signer := Signer{keyName: "plain", acc: acc}
	msg := banktypes.NewMsgSend(acc.GetAddress(), acc.GetAddress(), sdk.NewCoins())

	// A properly signed transaction passes the default signature verification.
	tx, err := prepareTx(ctx, app, keybase, []sdk.Msg{msg}, []Signer{signer}, mockChainID, true)
	require.NoError(t, err)
	_, err = makeBeforeTxDecorator(app).AnteHandle(ctx, tx, false, anteTerminator)
	require.NoError(t, err)
	require.Nil(t, app.AbstractAccountKeeper.GetSignerAddress(ctx), "no signer should be recorded for a non-AA tx")

	// A transaction signed for another chain fails it.
	tx, err = prepareTx(ctx, app, keybase, []sdk.Msg{msg}, []Signer{signer}, "other-chain", true)
	require.NoError(t, err)
	_, err = makeBeforeTxDecorator(app).AnteHandle(ctx, tx, false, anteTerminator)
	require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)
}

func TestBeforeTx_UnknownSignerRejected(t *testing.T) {
	app := xionapp.Setup(t)
	keybase := keyring.NewInMemory(app.AppCodec())
	ctx := app.NewContext(false).WithBlockTime(time.Now()).WithChainID(mockChainID)

	// The account is never stored, so the signer cannot be looked up.
	acc, err := makeMockAccount(app, ctx, keybase, "unknown")
	require.NoError(t, err)
	msg := banktypes.NewMsgSend(acc.GetAddress(), acc.GetAddress(), sdk.NewCoins())
	tx, err := prepareTx(ctx, app, keybase, []sdk.Msg{msg}, []Signer{{keyName: "unknown", acc: acc}}, mockChainID, true)
	require.NoError(t, err)

	is, _, _, err := abstractaccount.IsAbstractAccountTx(ctx, tx, app.AccountKeeper)
	require.ErrorIs(t, err, sdkerrors.ErrUnknownAddress)
	require.False(t, is)

	_, err = makeBeforeTxDecorator(app).AnteHandle(ctx, tx, false, anteTerminator)
	require.ErrorIs(t, err, sdkerrors.ErrUnknownAddress)
}

func TestIsAbstractAccountTx_FakeTxs(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)

	is, _, _, err := abstractaccount.IsAbstractAccountTx(ctx, plainTx{}, app.AccountKeeper)
	require.ErrorIs(t, err, sdkerrors.ErrTxDecode)
	require.False(t, is)

	is, _, _, err = abstractaccount.IsAbstractAccountTx(ctx, sigErrTx{}, app.AccountKeeper)
	require.EqualError(t, err, "signatures unavailable")
	require.False(t, is)
}

func TestSdkMsgsToAnys_UnencodableMsg(t *testing.T) {
	anys, err := abstractaccount.SdkMsgsToAnys([]sdk.Msg{&unencodableMsg{}})
	require.EqualError(t, err, "cannot encode message")
	require.Nil(t, anys)
}

func TestBeforeTx_UnencodableMsgRejected(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false).WithBlockTime(time.Now()).WithChainID(mockChainID)

	signer := xionapp.RandomAccAddress()
	app.AccountKeeper.SetAccount(ctx, types.NewAbstractAccount(signer.String(), app.AccountKeeper.NextAccountNumber(ctx), 0))

	_, err := makeBeforeTxDecorator(app).AnteHandle(ctx, unencodableMsgTx{signer: signer}, false, anteTerminator)
	require.EqualError(t, err, "cannot encode message")
}

func TestBeforeTx_UnsupportedSignMode(t *testing.T) {
	app := xionapp.Setup(t)
	keybase := keyring.NewInMemory(app.AppCodec())
	ctx := app.NewContext(false).WithBlockTime(time.Now()).WithChainID(mockChainID)
	absAcc, recipient := registeredAbstractAccount(t, app, ctx, keybase)

	txBuilder := app.TxConfig().NewTxBuilder()
	require.NoError(t, txBuilder.SetMsgs(banktypes.NewMsgSend(absAcc.GetAddress(), recipient.GetAddress(), sdk.NewCoins())))
	require.NoError(t, txBuilder.SetSignatures(txsigning.SignatureV2{
		PubKey: absAcc.GetPubKey(),
		Data: &txsigning.SingleSignatureData{
			SignMode:  txsigning.SignMode(99),
			Signature: []byte("credential"),
		},
		Sequence: absAcc.GetSequence(),
	}))

	_, err := makeBeforeTxDecorator(app).AnteHandle(ctx, txBuilder.GetTx(), false, anteTerminator)
	require.ErrorContains(t, err, "unsupported sign mode")
}

func TestBeforeTx_ParamsMissing(t *testing.T) {
	app := xionapp.Setup(t)
	keybase := keyring.NewInMemory(app.AppCodec())
	ctx := app.NewContext(false).WithBlockTime(time.Now()).WithChainID(mockChainID)
	absAcc, recipient := registeredAbstractAccount(t, app, ctx, keybase)

	msg := banktypes.NewMsgSend(absAcc.GetAddress(), recipient.GetAddress(), sdk.NewCoins())
	tx, err := prepareTx(ctx, app, keybase, []sdk.Msg{msg}, []Signer{{keyName: ownerKey, acc: absAcc}}, mockChainID, true)
	require.NoError(t, err)

	deleteModuleParams(app, ctx)
	_, err = makeBeforeTxDecorator(app).AnteHandle(ctx, tx, false, anteTerminator)
	require.ErrorIs(t, err, sdkerrors.ErrNotFound)
}

func TestBeforeTx_UnorderedTxDisabled(t *testing.T) {
	app := xionapp.Setup(t)
	keybase := keyring.NewInMemory(app.AppCodec())
	blockTime := time.Now()
	ctx := app.NewContext(false).WithBlockTime(blockTime).WithChainID(mockChainID)
	absAcc, recipient := registeredAbstractAccount(t, app, ctx, keybase)

	txBuilder := app.TxConfig().NewTxBuilder()
	require.NoError(t, txBuilder.SetMsgs(banktypes.NewMsgSend(absAcc.GetAddress(), recipient.GetAddress(), sdk.NewCoins())))
	txBuilder.SetUnordered(true)
	txBuilder.SetTimeoutTimestamp(blockTime.Add(time.Minute))
	require.NoError(t, txBuilder.SetSignatures(txsigning.SignatureV2{
		PubKey:   absAcc.GetPubKey(),
		Data:     &txsigning.SingleSignatureData{SignMode: signMode, Signature: nil},
		Sequence: 0,
	}))
	signerData := authsigning.SignerData{
		Address:       absAcc.GetAddress().String(),
		ChainID:       mockChainID,
		AccountNumber: absAcc.GetAccountNumber(),
		Sequence:      0,
		PubKey:        absAcc.GetPubKey(),
	}
	signBytes, err := authsigning.GetSignBytesAdapter(ctx, app.TxConfig().SignModeHandler(), signMode, signerData, txBuilder.GetTx())
	require.NoError(t, err)
	sigBytes, _, err := keybase.Sign(ownerKey, signBytes, signMode)
	require.NoError(t, err)
	require.NoError(t, txBuilder.SetSignatures(txsigning.SignatureV2{
		PubKey:   absAcc.GetPubKey(),
		Data:     &txsigning.SingleSignatureData{SignMode: signMode, Signature: sigBytes},
		Sequence: 0,
	}))
	tx := txBuilder.GetTx()

	// With unordered transactions enabled the tx is accepted.
	_, err = makeBeforeTxDecorator(app).AnteHandle(ctx, tx, false, anteTerminator)
	require.NoError(t, err)
	app.AbstractAccountKeeper.DeleteSignerAddress(ctx)

	// With them disabled on the chain it is refused before any nonce is used.
	decorator := abstractaccount.NewBeforeTxDecorator(app.AbstractAccountKeeper, orderedOnlyAccountKeeper{app.AccountKeeper}, app.TxConfig().SignModeHandler())
	_, err = decorator.AnteHandle(ctx, tx, false, anteTerminator)
	require.ErrorIs(t, err, sdkerrors.ErrNotSupported)
}

func TestBeforeTx_ContractCallCappedByRemainingGas(t *testing.T) {
	app := xionapp.Setup(t)
	keybase := keyring.NewInMemory(app.AppCodec())
	ctx := app.NewContext(false).WithBlockTime(time.Now()).WithChainID(mockChainID)
	absAcc, recipient := registeredAbstractAccount(t, app, ctx, keybase)

	msg := banktypes.NewMsgSend(absAcc.GetAddress(), recipient.GetAddress(), sdk.NewCoins())
	tx, err := prepareTx(ctx, app, keybase, []sdk.Msg{msg}, []Signer{{keyName: ownerKey, acc: absAcc}}, mockChainID, true)
	require.NoError(t, err)

	// The tx has less gas left than the module allows a before_tx call, so
	// the call runs under the tx budget and its consumption is charged back.
	params, err := app.AbstractAccountKeeper.GetParams(ctx)
	require.NoError(t, err)
	txGas := params.MaxGasBefore / 2
	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(txGas))

	_, err = makeBeforeTxDecorator(app).AnteHandle(ctx, tx, false, anteTerminator)
	require.NoError(t, err)
	require.Greater(t, ctx.GasMeter().GasConsumed(), storetypes.Gas(0))
	require.Less(t, ctx.GasMeter().GasConsumed(), txGas)
	require.Equal(t, absAcc.GetAddress(), app.AbstractAccountKeeper.GetSignerAddress(ctx))
}

// -------------------------------- AfterTx ---------------------------------

func TestAfterTx_ParamsMissing(t *testing.T) {
	app := xionapp.Setup(t)
	keybase := keyring.NewInMemory(app.AppCodec())
	ctx := app.NewContext(false).WithBlockTime(time.Now()).WithChainID(mockChainID)
	absAcc, recipient := registeredAbstractAccount(t, app, ctx, keybase)

	msg := banktypes.NewMsgSend(absAcc.GetAddress(), recipient.GetAddress(), sdk.NewCoins())
	tx, err := prepareTx(ctx, app, keybase, []sdk.Msg{msg}, []Signer{{keyName: ownerKey, acc: absAcc}}, mockChainID, true)
	require.NoError(t, err)

	app.AbstractAccountKeeper.SetSignerAddress(ctx, absAcc.GetAddress())
	deleteModuleParams(app, ctx)

	_, err = makeAfterTxDecorator(app).PostHandle(ctx, tx, false, true, postTerminator)
	require.ErrorIs(t, err, sdkerrors.ErrNotFound)
	require.Nil(t, app.AbstractAccountKeeper.GetSignerAddress(ctx), "the signer marker is cleared before the call")
}

func TestAfterTx_SignerIsNotAContract(t *testing.T) {
	app := xionapp.Setup(t)
	keybase := keyring.NewInMemory(app.AppCodec())
	ctx := app.NewContext(false).WithBlockTime(time.Now()).WithChainID(mockChainID)

	acc, err := makeMockAccount(app, ctx, keybase, "plain")
	require.NoError(t, err)
	app.AccountKeeper.SetAccount(ctx, acc)
	msg := banktypes.NewMsgSend(acc.GetAddress(), acc.GetAddress(), sdk.NewCoins())
	tx, err := prepareTx(ctx, app, keybase, []sdk.Msg{msg}, []Signer{{keyName: "plain", acc: acc}}, mockChainID, true)
	require.NoError(t, err)

	// A recorded signer with no contract behind it makes the after_tx call fail.
	app.AbstractAccountKeeper.SetSignerAddress(ctx, acc.GetAddress())

	_, err = makeAfterTxDecorator(app).PostHandle(ctx, tx, false, true, postTerminator)
	require.ErrorContains(t, err, "no such contract")
}
