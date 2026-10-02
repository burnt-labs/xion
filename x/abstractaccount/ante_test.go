package abstractaccount_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txsigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	xionapp "github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/x/abstractaccount"
	"github.com/burnt-labs/xion/x/abstractaccount/testdata"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

func TestIsAbstractAccountTx(t *testing.T) {
	var (
		app     = xionapp.Setup(t)
		ctx     = app.NewContext(false)
		keybase = keyring.NewInMemory(app.AppCodec())
	)

	// we create two mock accounts: 1, a BaseAccount, 2, an AbstractAccount
	acc1, err := makeMockAccount(app, ctx, keybase, "test1")
	require.NoError(t, err)

	acc2, err := makeMockAccount(app, ctx, keybase, "test2")
	acc2 = types.NewAbstractAccountFromAccount(acc2)
	require.NoError(t, err)

	app.AccountKeeper.SetAccount(ctx, acc1)
	app.AccountKeeper.SetAccount(ctx, acc2)

	signer1 := Signer{
		keyName:        "test1",
		acc:            acc1,
		overrideAccNum: nil,
		overrideSeq:    nil,
	}
	signer2 := Signer{
		keyName:        "test2",
		acc:            acc2,
		overrideAccNum: nil,
		overrideSeq:    nil,
	}

	for _, tc := range []struct {
		desc    string
		msgs    []sdk.Msg
		signers []Signer
		expIs   bool
	}{
		{
			desc: "tx has one signer and it is an AbstractAccount",
			msgs: []sdk.Msg{
				banktypes.NewMsgSend(acc2.GetAddress(), acc1.GetAddress(), sdk.NewCoins()),
			},
			signers: []Signer{signer2},
			expIs:   true,
		},
		{
			desc: "tx has one signer but it's not an AbstractAccount",
			msgs: []sdk.Msg{
				banktypes.NewMsgSend(acc1.GetAddress(), acc2.GetAddress(), sdk.NewCoins()),
			},
			signers: []Signer{signer1},
			expIs:   false,
		},
		{
			desc: "tx has more than one signers",
			msgs: []sdk.Msg{
				banktypes.NewMsgSend(acc1.GetAddress(), acc2.GetAddress(), sdk.NewCoins()),
				banktypes.NewMsgSend(acc2.GetAddress(), acc1.GetAddress(), sdk.NewCoins()),
			},
			signers: []Signer{signer1, signer2},
			expIs:   false,
		},
	} {
		sigTx, err := prepareTx(ctx, app, keybase, tc.msgs, tc.signers, mockChainID, true)
		require.NoError(t, err)

		is, _, _, err := abstractaccount.IsAbstractAccountTx(ctx, sigTx, app.AccountKeeper)
		require.NoError(t, err)
		require.Equal(t, tc.expIs, is)
	}
}

type BaseInstantiateMsg struct {
	PubKey []byte `json:"pubkey"`
}

func TestBeforeTx(t *testing.T) {
	var (
		app        = xionapp.Setup(t)
		keybase    = keyring.NewInMemory(app.AppCodec())
		mockAccNum = uint64(12345)
		mockSeq    = uint64(88888)
	)

	ctx := app.NewContext(false).WithBlockTime(time.Now()).WithChainID(mockChainID)

	// create two mock accounts
	acc1, err := makeMockAccount(app, ctx, keybase, "test1")
	require.NoError(t, err)

	acc2, err := makeMockAccount(app, ctx, keybase, "test2")
	require.NoError(t, err)

	// register the AbstractAccount
	absAcc, err := storeCodeAndRegisterAccount(
		ctx,
		app,
		// use the pubkey of acc1 as the AbstractAccount's pubkey
		acc1.GetAddress(),
		testdata.AccountWasm,
		&BaseInstantiateMsg{PubKey: acc1.GetPubKey().Bytes()},
		sdk.NewCoins(),
	)
	require.NoError(t, err)

	// change the AbstractAccount's account number and sequence to some non-zero
	// numbers to make the tests harder
	app.AccountKeeper.RemoveAccount(ctx, absAcc)
	err = absAcc.SetAccountNumber(mockAccNum)
	require.NoError(t, err)
	err = absAcc.SetSequence(mockSeq)
	require.NoError(t, err)
	app.AccountKeeper.SetAccount(ctx, absAcc)

	for _, tc := range []struct {
		desc     string
		simulate bool   // whether to run the AnteHandler in simulation mode
		sign     bool   // whether a signature is to be included with this tx
		signWith string // if a sig is to be included, which key to use to sign it
		chainID  string
		accNum   uint64
		seq      uint64
		maxGas   uint64
		expOk    bool
		expPanic bool
	}{
		{
			desc:     "tx signed with the correct key",
			simulate: false,
			sign:     true,
			signWith: "test1",
			chainID:  mockChainID,
			accNum:   mockAccNum,
			seq:      mockSeq,
			maxGas:   types.DefaultMaxGas,
			expOk:    true,
			expPanic: false,
		},
		{
			desc:     "tx signed with an incorrect key",
			simulate: false,
			sign:     true,
			signWith: "test2",
			chainID:  mockChainID,
			accNum:   mockAccNum,
			seq:      mockSeq,
			maxGas:   types.DefaultMaxGas,
			expOk:    false,
			expPanic: false,
		},
		{
			desc:     "tx signed with an incorrect chain id",
			simulate: false,
			sign:     true,
			signWith: "test1",
			chainID:  "wrong-chain-id",
			accNum:   mockAccNum,
			seq:      mockSeq,
			maxGas:   types.DefaultMaxGas,
			expOk:    false,
			expPanic: false,
		},
		{
			desc:     "tx signed with an incorrect account number",
			simulate: false,
			sign:     true,
			signWith: "test1",
			chainID:  mockChainID,
			accNum:   4524455,
			seq:      mockSeq,
			maxGas:   types.DefaultMaxGas,
			expOk:    false,
			expPanic: false,
		},
		{
			desc:     "tx signed with an incorrect sequence",
			simulate: false,
			sign:     true,
			signWith: "test1",
			chainID:  mockChainID,
			accNum:   mockAccNum,
			seq:      5786786,
			maxGas:   types.DefaultMaxGas,
			expOk:    false,
			expPanic: false,
		},
		{
			desc:     "contract call exceeds gas limit",
			simulate: false,
			sign:     true,
			signWith: "test1",
			chainID:  mockChainID,
			accNum:   mockAccNum,
			seq:      mockSeq,
			maxGas:   1, // the call for sure will use more than 1 gas
			expOk:    false,
			expPanic: true, // attempting to consume above the gas limit results in panicking
		},
		{
			desc:     "not in simulation mode, but tx isn't signed",
			simulate: false,
			sign:     false,
			signWith: "",
			chainID:  mockChainID,
			accNum:   mockAccNum,
			seq:      mockSeq,
			maxGas:   types.DefaultMaxGas,
			expOk:    false,
			expPanic: false,
		},
		{
			desc:     "in simulation, tx is signed",
			simulate: true,
			sign:     true,
			signWith: "test1",
			chainID:  mockChainID,
			accNum:   mockAccNum,
			seq:      mockSeq,
			maxGas:   types.DefaultMaxGas,
			expOk:    true, // we accept it
			expPanic: false,
		},
		{
			desc:     "in simulation, tx is not signed",
			simulate: true,
			sign:     false,
			signWith: "test1",
			chainID:  mockChainID,
			accNum:   mockAccNum,
			seq:      mockSeq,
			maxGas:   types.DefaultMaxGas,
			expOk:    true, // in simulation mode, for this particular account type, the credential can be omitted
			expPanic: false,
		},
	} {
		// set max gas
		err := app.AbstractAccountKeeper.SetParams(ctx, &types.Params{
			MaxGasBefore: tc.maxGas,
			MaxGasAfter:  types.DefaultMaxGas,
		})
		require.NoError(t, err)

		msg := banktypes.NewMsgSend(absAcc.GetAddress(), acc2.GetAddress(), sdk.NewCoins())

		signer := Signer{
			keyName:        tc.signWith,
			acc:            absAcc,
			overrideAccNum: &tc.accNum,
			overrideSeq:    &tc.seq,
		}

		tx, err := prepareTx(
			ctx,
			app,
			keybase,
			[]sdk.Msg{msg},
			[]Signer{signer},
			tc.chainID,
			tc.sign,
		)
		require.NoError(t, err)

		if tc.expPanic {
			require.Panics(t, func() {
				decorator := makeBeforeTxDecorator(app)
				_, _ = decorator.AnteHandle(ctx, tx, tc.simulate, anteTerminator)
			})

			continue
		}

		decorator := makeBeforeTxDecorator(app)
		_, err = decorator.AnteHandle(ctx, tx, tc.simulate, anteTerminator)

		if tc.expOk {
			require.NoError(t, err)

			// the signer address should have been stored for use by the PostHandler
			signerAddr := app.AbstractAccountKeeper.GetSignerAddress(ctx)
			require.Equal(t, absAcc.GetAddress(), signerAddr)

			// delete the stored signer address so that we start from a clean state
			// for the next test case
			app.AbstractAccountKeeper.DeleteSignerAddress(ctx)
		} else {
			require.Error(t, err)
		}
	}
}

func TestAfterTx(t *testing.T) {
	var (
		app     = xionapp.Setup(t)
		keybase = keyring.NewInMemory(app.AppCodec())
	)

	ctx := app.NewContext(false).WithBlockTime(time.Now()).WithChainID(mockChainID)

	// create a mock account
	acc, err := makeMockAccount(app, ctx, keybase, "test1")
	require.NoError(t, err)

	// register the AbstractAccount
	absAcc, err := storeCodeAndRegisterAccount(
		ctx,
		app,
		acc.GetAddress(),
		testdata.AccountWasm,
		&BaseInstantiateMsg{PubKey: acc.GetPubKey().Bytes()},
		sdk.NewCoins(),
	)
	require.NoError(t, err)

	// save the signer address to mimic what happens in the BeforeTx hook
	app.AbstractAccountKeeper.SetSignerAddress(ctx, absAcc.GetAddress())

	tx, err := prepareTx(
		ctx,
		app,
		keybase,
		[]sdk.Msg{banktypes.NewMsgSend(absAcc.GetAddress(), acc.GetAddress(), sdk.NewCoins())},
		[]Signer{{
			keyName: "test1",
			acc:     absAcc,
		}},
		mockChainID,
		true,
	)
	require.NoError(t, err)

	decorator := makeAfterTxDecorator(app)
	_, err = decorator.PostHandle(ctx, tx, false, true, postTerminator)
	require.NoError(t, err)
}

func TestSigVerificationGasConsumer(t *testing.T) {
	var (
		app     = xionapp.Setup(t)
		ctx     = app.NewContext(false)
		keybase = keyring.NewInMemory(app.AppCodec())
	)

	// create mock account
	acc, err := makeMockAccount(app, ctx, keybase, "test1")
	require.NoError(t, err)

	// test with NilPubKey - should not consume gas
	nilPubKey := &types.NilPubKey{}
	sig := txsigning.SignatureV2{
		PubKey: nilPubKey,
		Data: &txsigning.SingleSignatureData{
			SignMode:  txsigning.SignMode_SIGN_MODE_DIRECT,
			Signature: []byte("test"),
		},
		Sequence: 0,
	}

	gasMeter := storetypes.NewGasMeter(1000000)
	err = abstractaccount.SigVerificationGasConsumer(gasMeter, sig, authtypes.DefaultParams())
	require.NoError(t, err)
	require.Equal(t, storetypes.Gas(0), gasMeter.GasConsumed())

	// test with regular pubkey - should delegate to default consumer
	sig.PubKey = acc.GetPubKey()
	gasMeter = storetypes.NewGasMeter(1000000)
	err = abstractaccount.SigVerificationGasConsumer(gasMeter, sig, authtypes.DefaultParams())
	require.NoError(t, err)
	require.Greater(t, gasMeter.GasConsumed(), storetypes.Gas(0))
}

func TestIsAbstractAccountTx_ErrorCases(t *testing.T) {
	var (
		app     = xionapp.Setup(t)
		ctx     = app.NewContext(false)
		keybase = keyring.NewInMemory(app.AppCodec())
	)

	// create mock accounts
	acc1, err := makeMockAccount(app, ctx, keybase, "test1")
	require.NoError(t, err)

	app.AccountKeeper.SetAccount(ctx, acc1)

	// test with invalid signer address (account not found)
	invalidAddr, _ := sdk.AccAddressFromBech32("cosmos1invalid000000000000000000000000000000000")
	msg := banktypes.NewMsgSend(invalidAddr, acc1.GetAddress(), sdk.NewCoins())

	signer := Signer{
		keyName: "test1",
		acc:     acc1,
	}
	// modify the signer to use invalid address
	signer.acc = &mockAccount{address: invalidAddr, pubkey: acc1.GetPubKey()}

	tx, err := prepareTx(ctx, app, keybase, []sdk.Msg{msg}, []Signer{signer}, mockChainID, true)
	require.NoError(t, err)

	is, _, _, err := abstractaccount.IsAbstractAccountTx(ctx, tx, app.AccountKeeper)
	require.Error(t, err)
	require.False(t, is)
}

func TestBeforeTx_ErrorCases(t *testing.T) {
	var (
		app     = xionapp.Setup(t)
		keybase = keyring.NewInMemory(app.AppCodec())
	)

	ctx := app.NewContext(false).WithBlockTime(time.Now()).WithChainID(mockChainID)

	// create mock account
	acc1, err := makeMockAccount(app, ctx, keybase, "test1")
	require.NoError(t, err)

	// register the AbstractAccount
	absAcc, err := storeCodeAndRegisterAccount(
		ctx,
		app,
		acc1.GetAddress(),
		testdata.AccountWasm,
		&BaseInstantiateMsg{PubKey: acc1.GetPubKey().Bytes()},
		sdk.NewCoins(),
	)
	require.NoError(t, err)

	// test with zero block time
	ctxZeroTime := app.NewContext(false).WithChainID(mockChainID)
	msg := banktypes.NewMsgSend(absAcc.GetAddress(), acc1.GetAddress(), sdk.NewCoins())
	signer := Signer{
		keyName: "test1",
		acc:     absAcc,
	}

	tx, err := prepareTx(ctxZeroTime, app, keybase, []sdk.Msg{msg}, []Signer{signer}, mockChainID, true)
	require.NoError(t, err)

	decorator := makeBeforeTxDecorator(app)
	_, err = decorator.AnteHandle(ctxZeroTime, tx, false, anteTerminator)
	require.Error(t, err)
	require.Contains(t, err.Error(), "block time can not be zero")
}

func TestAfterTx_ErrorCases(t *testing.T) {
	var (
		app     = xionapp.Setup(t)
		keybase = keyring.NewInMemory(app.AppCodec())
	)

	ctx := app.NewContext(false).WithBlockTime(time.Now()).WithChainID(mockChainID)

	// create mock account
	acc, err := makeMockAccount(app, ctx, keybase, "test1")
	require.NoError(t, err)

	// test when no signer address is stored (non-AA tx)
	tx, err := prepareTx(
		ctx,
		app,
		keybase,
		[]sdk.Msg{banktypes.NewMsgSend(acc.GetAddress(), acc.GetAddress(), sdk.NewCoins())},
		[]Signer{{keyName: "test1", acc: acc}},
		mockChainID,
		true,
	)
	require.NoError(t, err)

	decorator := makeAfterTxDecorator(app)
	newCtx, err := decorator.PostHandle(ctx, tx, false, true, postTerminator)
	require.NoError(t, err)
	require.Equal(t, ctx, newCtx) // should pass through unchanged
}

func TestIsAbstractAccountTx_MoreCases(t *testing.T) {
	var (
		app     = xionapp.Setup(t)
		ctx     = app.NewContext(false)
		keybase = keyring.NewInMemory(app.AppCodec())
	)

	// create mock accounts
	acc1, err := makeMockAccount(app, ctx, keybase, "test1")
	require.NoError(t, err)

	acc2, err := makeMockAccount(app, ctx, keybase, "test2")
	require.NoError(t, err)

	app.AccountKeeper.SetAccount(ctx, acc1)
	app.AccountKeeper.SetAccount(ctx, acc2)

	// test case: zero signers
	tx, err := prepareTx(ctx, app, keybase, []sdk.Msg{
		banktypes.NewMsgSend(acc1.GetAddress(), acc2.GetAddress(), sdk.NewCoins()),
	}, []Signer{}, mockChainID, false) // no signers
	require.NoError(t, err)

	is, _, _, err := abstractaccount.IsAbstractAccountTx(ctx, tx, app.AccountKeeper)
	require.NoError(t, err)
	require.False(t, is)

	// test case: two signers with same signer (should return false)
	signer1 := Signer{
		keyName: "test1",
		acc:     acc1,
	}

	tx, err = prepareTx(ctx, app, keybase, []sdk.Msg{
		banktypes.NewMsgSend(acc1.GetAddress(), acc2.GetAddress(), sdk.NewCoins()),
		banktypes.NewMsgSend(acc1.GetAddress(), acc2.GetAddress(), sdk.NewCoins()),
	}, []Signer{signer1, signer1}, mockChainID, true)
	require.NoError(t, err)

	is, _, _, err = abstractaccount.IsAbstractAccountTx(ctx, tx, app.AccountKeeper)
	require.NoError(t, err)
	require.False(t, is) // more than one signer/signature
}

func TestPrepareCredentials_ErrorCases(t *testing.T) {
	var (
		app     = xionapp.Setup(t)
		keybase = keyring.NewInMemory(app.AppCodec())
	)

	ctx := app.NewContext(false).WithBlockTime(time.Now()).WithChainID(mockChainID)

	// create mock account
	acc, err := makeMockAccount(app, ctx, keybase, "test1")
	require.NoError(t, err)

	// create a tx with multi-signature data (not single signature)
	multiSigData := &txsigning.MultiSignatureData{
		BitArray: nil,
		Signatures: []txsigning.SignatureData{
			&txsigning.SingleSignatureData{
				SignMode:  txsigning.SignMode_SIGN_MODE_DIRECT,
				Signature: []byte("sig1"),
			},
		},
	}

	// This will test the error case in prepareCredentials when sigData is not SingleSignatureData
	msg := banktypes.NewMsgSend(acc.GetAddress(), acc.GetAddress(), sdk.NewCoins())
	txBuilder := app.TxConfig().NewTxBuilder()
	err = txBuilder.SetMsgs(msg)
	require.NoError(t, err)

	// Set multi-signature to trigger the error
	sig := txsigning.SignatureV2{
		PubKey:   acc.GetPubKey(),
		Data:     multiSigData,
		Sequence: acc.GetSequence(),
	}
	err = txBuilder.SetSignatures(sig)
	require.NoError(t, err)

	// Create AbstractAccount to trigger the prepareCredentials path
	absAcc := types.NewAbstractAccountFromAccount(acc)
	app.AccountKeeper.SetAccount(ctx, absAcc)

	decorator := makeBeforeTxDecorator(app)
	_, err = decorator.AnteHandle(ctx, txBuilder.GetTx(), false, anteTerminator)
	require.Error(t, err)
	require.Contains(t, err.Error(), "signature is not a txsigning.SingleSignatureData")
}

func TestSdkMsgsToAnys(t *testing.T) {
	// Test with valid messages
	msg1 := banktypes.NewMsgSend(sdk.AccAddress("addr1"), sdk.AccAddress("addr2"), sdk.NewCoins())
	msg2 := banktypes.NewMsgSend(sdk.AccAddress("addr3"), sdk.AccAddress("addr4"), sdk.NewCoins())

	anys, err := abstractaccount.SdkMsgsToAnys([]sdk.Msg{msg1, msg2})
	require.NoError(t, err)
	require.Len(t, anys, 2)
	require.NotNil(t, anys[0])
	require.NotNil(t, anys[1])

	// Test with empty slice
	anys, err = abstractaccount.SdkMsgsToAnys([]sdk.Msg{})
	require.NoError(t, err)
	require.Len(t, anys, 0)
}

// TestBeforeTx_UnorderedTx tests handleUnorderedTx which enforces
// replay protection for unordered AbstractAccount transactions.
func TestBeforeTx_UnorderedTx(t *testing.T) {
	var (
		app     = xionapp.Setup(t)
		keybase = keyring.NewInMemory(app.AppCodec())
	)

	blockTime := time.Now()
	ctx := app.NewContext(false).WithBlockTime(blockTime).WithChainID(mockChainID)

	// create a signing key and register it as an AbstractAccount
	acc1, err := makeMockAccount(app, ctx, keybase, "test1")
	require.NoError(t, err)

	absAcc, err := storeCodeAndRegisterAccount(
		ctx,
		app,
		acc1.GetAddress(),
		testdata.AccountWasm,
		&BaseInstantiateMsg{PubKey: acc1.GetPubKey().Bytes()},
		sdk.NewCoins(),
	)
	require.NoError(t, err)

	acc2, err := makeMockAccount(app, ctx, keybase, "test2")
	require.NoError(t, err)

	msg := banktypes.NewMsgSend(absAcc.GetAddress(), acc2.GetAddress(), sdk.NewCoins())

	// helper: build an ordered signer (sequence override = 0 for unordered mode)
	zeroSeq := uint64(0)
	unorderedSigner := Signer{
		keyName:        "test1",
		acc:            absAcc,
		overrideAccNum: nil,
		overrideSeq:    &zeroSeq,
	}

	for _, tc := range []struct {
		desc             string
		timeoutTimestamp time.Time
		simulate         bool
		expOk            bool
		expErrContains   string
	}{
		{
			desc:             "valid unordered tx",
			timeoutTimestamp: blockTime.Add(5 * time.Minute),
			simulate:         false,
			expOk:            true,
		},
		{
			desc:             "valid unordered tx in simulation mode",
			timeoutTimestamp: blockTime.Add(5 * time.Minute),
			simulate:         true,
			expOk:            true,
		},
		{
			desc:             "zero timeout_timestamp (not set)",
			timeoutTimestamp: time.Time{},
			simulate:         false,
			expOk:            false,
			expErrContains:   "timeout_timestamp",
		},
		{
			desc:             "expired timeout_timestamp",
			timeoutTimestamp: blockTime.Add(-1 * time.Second),
			simulate:         false,
			expOk:            false,
			expErrContains:   "timeout_timestamp that has already passed",
		},
		{
			desc:             "timeout_timestamp exceeds max TTL",
			timeoutTimestamp: blockTime.Add(abstractaccount.DefaultMaxUnorderedTTL + time.Minute),
			simulate:         false,
			expOk:            false,
			expErrContains:   "ttl exceeds",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			txBuilder := app.TxConfig().NewTxBuilder()
			require.NoError(t, txBuilder.SetMsgs(msg))
			txBuilder.SetUnordered(true)
			txBuilder.SetTimeoutTimestamp(tc.timeoutTimestamp)

			// round 1: empty sigs
			emptySig := txsigning.SignatureV2{
				PubKey: absAcc.GetPubKey(),
				Data: &txsigning.SingleSignatureData{
					SignMode:  signMode,
					Signature: nil,
				},
				Sequence: 0,
			}
			require.NoError(t, txBuilder.SetSignatures(emptySig))

			// round 2: sign
			signerData := authsigning.SignerData{
				Address:       absAcc.GetAddress().String(),
				ChainID:       mockChainID,
				AccountNumber: unorderedSigner.AccountNumber(),
				Sequence:      0,
				PubKey:        absAcc.GetPubKey(),
			}
			signBytes, err := authsigning.GetSignBytesAdapter(ctx, app.TxConfig().SignModeHandler(), signMode, signerData, txBuilder.GetTx())
			require.NoError(t, err)
			sigBytes, _, err := keybase.Sign("test1", signBytes, signMode)
			require.NoError(t, err)
			finalSig := txsigning.SignatureV2{
				PubKey: absAcc.GetPubKey(),
				Data: &txsigning.SingleSignatureData{
					SignMode:  signMode,
					Signature: sigBytes,
				},
				Sequence: 0,
			}
			require.NoError(t, txBuilder.SetSignatures(finalSig))

			tx := txBuilder.GetTx()
			decorator := makeBeforeTxDecorator(app)
			_, err = decorator.AnteHandle(ctx, tx, tc.simulate, anteTerminator)

			if tc.expOk {
				require.NoError(t, err, tc.desc)
				app.AbstractAccountKeeper.DeleteSignerAddress(ctx)
			} else {
				require.Error(t, err, tc.desc)
				if tc.expErrContains != "" {
					require.Contains(t, err.Error(), tc.expErrContains, tc.desc)
				}
			}
		})
	}
}

// TestBeforeTx_UnorderedTx_ReplayRejection verifies that replaying the same
// unordered nonce (sender + timeout_timestamp) is rejected.
func TestBeforeTx_UnorderedTx_ReplayRejection(t *testing.T) {
	app := xionapp.Setup(t)
	keybase := keyring.NewInMemory(app.AppCodec())

	blockTime := time.Now()
	ctx := app.NewContext(false).WithBlockTime(blockTime).WithChainID(mockChainID)

	acc1, err := makeMockAccount(app, ctx, keybase, "test1")
	require.NoError(t, err)

	absAcc, err := storeCodeAndRegisterAccount(
		ctx, app, acc1.GetAddress(), testdata.AccountWasm,
		&BaseInstantiateMsg{PubKey: acc1.GetPubKey().Bytes()}, sdk.NewCoins(),
	)
	require.NoError(t, err)

	timeoutTS := blockTime.Add(5 * time.Minute)

	buildTx := func() sdk.Tx {
		txBuilder := app.TxConfig().NewTxBuilder()
		require.NoError(t, txBuilder.SetMsgs(banktypes.NewMsgSend(absAcc.GetAddress(), acc1.GetAddress(), sdk.NewCoins())))
		txBuilder.SetUnordered(true)
		txBuilder.SetTimeoutTimestamp(timeoutTS)

		emptySig := txsigning.SignatureV2{
			PubKey:   absAcc.GetPubKey(),
			Data:     &txsigning.SingleSignatureData{SignMode: signMode, Signature: nil},
			Sequence: 0,
		}
		require.NoError(t, txBuilder.SetSignatures(emptySig))

		signerData := authsigning.SignerData{
			Address:       absAcc.GetAddress().String(),
			ChainID:       mockChainID,
			AccountNumber: absAcc.GetAccountNumber(),
			Sequence:      0,
			PubKey:        absAcc.GetPubKey(),
		}
		signBytes, err := authsigning.GetSignBytesAdapter(ctx, app.TxConfig().SignModeHandler(), signMode, signerData, txBuilder.GetTx())
		require.NoError(t, err)
		sigBytes, _, err := keybase.Sign("test1", signBytes, signMode)
		require.NoError(t, err)
		require.NoError(t, txBuilder.SetSignatures(txsigning.SignatureV2{
			PubKey:   absAcc.GetPubKey(),
			Data:     &txsigning.SingleSignatureData{SignMode: signMode, Signature: sigBytes},
			Sequence: 0,
		}))
		return txBuilder.GetTx()
	}

	// First submission should succeed
	decorator := makeBeforeTxDecorator(app)
	_, err = decorator.AnteHandle(ctx, buildTx(), false, anteTerminator)
	require.NoError(t, err)
	app.AbstractAccountKeeper.DeleteSignerAddress(ctx)

	// Second submission with same nonce must be rejected
	decorator = makeBeforeTxDecorator(app)
	_, err = decorator.AnteHandle(ctx, buildTx(), false, anteTerminator)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unordered nonce")
}

// TestBeforeTx_UnorderedTx_NonZeroSequenceRejected verifies that unordered
// transactions with a non-zero sequence field are rejected.
func TestBeforeTx_UnorderedTx_NonZeroSequenceRejected(t *testing.T) {
	app := xionapp.Setup(t)
	keybase := keyring.NewInMemory(app.AppCodec())

	blockTime := time.Now()
	ctx := app.NewContext(false).WithBlockTime(blockTime).WithChainID(mockChainID)

	acc1, err := makeMockAccount(app, ctx, keybase, "test1")
	require.NoError(t, err)

	absAcc, err := storeCodeAndRegisterAccount(
		ctx, app, acc1.GetAddress(), testdata.AccountWasm,
		&BaseInstantiateMsg{PubKey: acc1.GetPubKey().Bytes()}, sdk.NewCoins(),
	)
	require.NoError(t, err)

	timeoutTS := blockTime.Add(5 * time.Minute)

	// Build unordered tx with non-zero sequence
	txBuilder := app.TxConfig().NewTxBuilder()
	require.NoError(t, txBuilder.SetMsgs(banktypes.NewMsgSend(absAcc.GetAddress(), acc1.GetAddress(), sdk.NewCoins())))
	txBuilder.SetUnordered(true)
	txBuilder.SetTimeoutTimestamp(timeoutTS)

	// Use sequence = 1 (should be rejected for unordered txs)
	nonZeroSeq := uint64(1)
	emptySig := txsigning.SignatureV2{
		PubKey:   absAcc.GetPubKey(),
		Data:     &txsigning.SingleSignatureData{SignMode: signMode, Signature: nil},
		Sequence: nonZeroSeq,
	}
	require.NoError(t, txBuilder.SetSignatures(emptySig))

	signerData := authsigning.SignerData{
		Address:       absAcc.GetAddress().String(),
		ChainID:       mockChainID,
		AccountNumber: absAcc.GetAccountNumber(),
		Sequence:      nonZeroSeq,
		PubKey:        absAcc.GetPubKey(),
	}
	signBytes, err := authsigning.GetSignBytesAdapter(ctx, app.TxConfig().SignModeHandler(), signMode, signerData, txBuilder.GetTx())
	require.NoError(t, err)
	sigBytes, _, err := keybase.Sign("test1", signBytes, signMode)
	require.NoError(t, err)
	require.NoError(t, txBuilder.SetSignatures(txsigning.SignatureV2{
		PubKey:   absAcc.GetPubKey(),
		Data:     &txsigning.SingleSignatureData{SignMode: signMode, Signature: sigBytes},
		Sequence: nonZeroSeq,
	}))

	decorator := makeBeforeTxDecorator(app)
	_, err = decorator.AnteHandle(ctx, txBuilder.GetTx(), false, anteTerminator)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sequence is not allowed for unordered transactions")
}

// TestBeforeTx_UnorderedTx_AdvancedAccountSequence verifies that an unordered
// transaction still verifies after the account's sequence has advanced past
// zero. Unordered txs are signed with sequence 0 regardless of the account
// state, so the sign bytes handed to the account contract must be computed
// with the signature's sequence, not the account's.
func TestBeforeTx_UnorderedTx_AdvancedAccountSequence(t *testing.T) {
	app := xionapp.Setup(t)
	keybase := keyring.NewInMemory(app.AppCodec())

	blockTime := time.Now()
	ctx := app.NewContext(false).WithBlockTime(blockTime).WithChainID(mockChainID)

	acc1, err := makeMockAccount(app, ctx, keybase, "test1")
	require.NoError(t, err)

	absAcc, err := storeCodeAndRegisterAccount(
		ctx, app, acc1.GetAddress(), testdata.AccountWasm,
		&BaseInstantiateMsg{PubKey: acc1.GetPubKey().Bytes()}, sdk.NewCoins(),
	)
	require.NoError(t, err)

	// Advance the account sequence past zero, as any ordered tx would have.
	require.NoError(t, absAcc.SetSequence(7))
	app.AccountKeeper.SetAccount(ctx, absAcc)

	timeoutTS := blockTime.Add(5 * time.Minute)

	txBuilder := app.TxConfig().NewTxBuilder()
	require.NoError(t, txBuilder.SetMsgs(banktypes.NewMsgSend(absAcc.GetAddress(), acc1.GetAddress(), sdk.NewCoins())))
	txBuilder.SetUnordered(true)
	txBuilder.SetTimeoutTimestamp(timeoutTS)

	emptySig := txsigning.SignatureV2{
		PubKey:   absAcc.GetPubKey(),
		Data:     &txsigning.SingleSignatureData{SignMode: signMode, Signature: nil},
		Sequence: 0,
	}
	require.NoError(t, txBuilder.SetSignatures(emptySig))

	// The client signs with sequence 0, the only value permitted for
	// unordered transactions.
	signerData := authsigning.SignerData{
		Address:       absAcc.GetAddress().String(),
		ChainID:       mockChainID,
		AccountNumber: absAcc.GetAccountNumber(),
		Sequence:      0,
		PubKey:        absAcc.GetPubKey(),
	}
	signBytes, err := authsigning.GetSignBytesAdapter(ctx, app.TxConfig().SignModeHandler(), signMode, signerData, txBuilder.GetTx())
	require.NoError(t, err)
	sigBytes, _, err := keybase.Sign("test1", signBytes, signMode)
	require.NoError(t, err)
	require.NoError(t, txBuilder.SetSignatures(txsigning.SignatureV2{
		PubKey:   absAcc.GetPubKey(),
		Data:     &txsigning.SingleSignatureData{SignMode: signMode, Signature: sigBytes},
		Sequence: 0,
	}))

	decorator := makeBeforeTxDecorator(app)
	_, err = decorator.AnteHandle(ctx, txBuilder.GetTx(), false, anteTerminator)
	require.NoError(t, err)
}

// Mock types for testing error cases

type mockAccount struct {
	address sdk.AccAddress
	pubkey  cryptotypes.PubKey
}

func (m *mockAccount) GetAddress() sdk.AccAddress         { return m.address }
func (m *mockAccount) SetAddress(sdk.AccAddress) error    { return nil }
func (m *mockAccount) GetPubKey() cryptotypes.PubKey      { return m.pubkey }
func (m *mockAccount) SetPubKey(cryptotypes.PubKey) error { return nil }
func (m *mockAccount) GetAccountNumber() uint64           { return 1 }
func (m *mockAccount) SetAccountNumber(uint64) error      { return nil }
func (m *mockAccount) GetSequence() uint64                { return 0 }
func (m *mockAccount) SetSequence(uint64) error           { return nil }
func (m *mockAccount) String() string                     { return "" }
func (m *mockAccount) ProtoMessage()                      {}
func (m *mockAccount) Reset()                             {}
