package keeper_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	xionapp "github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/x/abstractaccount/keeper"
	"github.com/burnt-labs/xion/x/abstractaccount/testdata"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

const (
	testEthAddress = "0x690E03B98D469FB04396570051B3FF31AC50A8EC"
	testPubkey     = "A08EGB7ro1ORuFhjOnZcSgwYlpe0DSFjVNUIkNNQxwKQ"
)

func secp256k1InitMsg(pubkey string) []byte {
	return []byte(`{"authenticator":{"Secp256K1":{"id":0,"pubkey":"` + pubkey + `","signature":"c2ln"}}}`)
}

func ethWalletInitMsg(address string) []byte {
	return []byte(`{"authenticator":{"EthWallet":{"id":0,"address":"` + address + `","signature":"c2ln"}}}`)
}

func jwtInitMsg(aud, sub string) []byte {
	return []byte(`{"authenticator":{"Jwt":{"id":0,"aud":"` + aud + `","sub":"` + sub + `","token":"dG9r"}}}`)
}

func sha256Of(b []byte) []byte {
	sum := sha256.Sum256(b)
	return sum[:]
}

// checksumAccountFixture is a code whose checksum stands in for code 1880's.
type checksumAccountFixture struct {
	app      *xionapp.WasmApp
	ctx      sdk.Context
	codeID   uint64
	checksum []byte
}

func newChecksumAccountFixture(t *testing.T) checksumAccountFixture {
	t.Helper()

	app := xionapp.Setup(t)
	ctx := app.NewContext(false)
	codeID, checksum, err := app.AbstractAccountKeeper.ContractKeeper().Create(ctx, user, accountWasm(), nil)
	require.NoError(t, err)

	return checksumAccountFixture{app: app, ctx: ctx, codeID: codeID, checksum: checksum}
}

// importAccount puts a contract at address on chain through wasm genesis, with
// history entries whose INIT message is initMsg, and makes it an abstract
// account when abstract is true.
func (f checksumAccountFixture) importAccount(
	t *testing.T,
	address, creator sdk.AccAddress,
	initCodeID uint64,
	initMsg []byte,
	abstract bool,
	extraHistory ...wasmtypes.ContractCodeHistoryEntry,
) {
	t.Helper()

	position := &wasmtypes.AbsoluteTxPosition{BlockHeight: 1, TxIndex: 1}
	currentCodeID := initCodeID
	if len(extraHistory) > 0 {
		currentCodeID = extraHistory[len(extraHistory)-1].CodeID
	}
	history := append([]wasmtypes.ContractCodeHistoryEntry{{
		Operation: wasmtypes.ContractCodeHistoryOperationTypeInit,
		CodeID:    initCodeID,
		Updated:   position,
		Msg:       initMsg,
	}}, extraHistory...)

	_, err := wasmkeeper.InitGenesis(f.ctx, &f.app.WasmKeeper, wasmtypes.GenesisState{
		Params: f.app.WasmKeeper.GetParams(f.ctx),
		Contracts: []wasmtypes.Contract{{
			ContractAddress: address.String(),
			ContractInfo: wasmtypes.ContractInfo{
				CodeID:  currentCodeID,
				Creator: creator.String(),
				Admin:   address.String(),
				Label:   "checksum account",
				Created: position,
			},
			ContractCodeHistory: history,
		}},
	})
	require.NoError(t, err)

	if abstract {
		f.app.AccountKeeper.SetAccount(f.ctx, types.NewAbstractAccount(address.String(), f.app.AccountKeeper.NextAccountNumber(f.ctx), 0))
	} else {
		f.app.AccountKeeper.SetAccount(f.ctx, f.app.AccountKeeper.NewAccountWithAddress(f.ctx, address))
	}
}

// importChecksumAccount imports an abstract account at the address derived from
// the fixture code's checksum, the way pre-registry registration placed it.
func (f checksumAccountFixture) importChecksumAccount(t *testing.T, creator sdk.AccAddress, salt, initMsg []byte) sdk.AccAddress {
	t.Helper()

	address := wasmkeeper.BuildContractAddressPredictable(f.checksum, creator, salt, nil)
	f.importAccount(t, address, creator, f.codeID, initMsg, true)
	return address
}

func TestBackfillChecksumAccountAddressesRegistersEachAuthenticatorKind(t *testing.T) {
	f := newChecksumAccountFixture(t)
	k := f.app.AbstractAccountKeeper
	creator := xionapp.RandomAccAddress()

	accounts := []struct {
		name    string
		salt    []byte
		initMsg []byte
	}{
		{"Secp256K1", sha256Of([]byte(testPubkey)), secp256k1InitMsg(testPubkey)},
		{"EthWallet", sha256Of(mustHex(t, "690e03b98d469fb04396570051b3ff31ac50a8ec")), ethWalletInitMsg(testEthAddress)},
		{"Jwt", sha256Of([]byte("project-test.user-test")), jwtInitMsg("project-test", "user-test")},
	}
	addresses := make([]sdk.AccAddress, len(accounts))
	for i, account := range accounts {
		addresses[i] = f.importChecksumAccount(t, creator, account.salt, account.initMsg)
	}

	result, err := k.BackfillChecksumAccountAddresses(f.ctx, f.codeID, f.checksum)
	require.NoError(t, err)
	require.Equal(t, keeper.ChecksumAccountBackfill{Registered: 3}, result)

	params, err := types.NewParamsWithAddressDerivationHash(true, nil, types.DefaultMaxGas, types.DefaultMaxGas, addressDerivationHash)
	require.NoError(t, err)
	require.NoError(t, k.SetParams(f.ctx, params))
	queryServer := keeper.NewQueryServerImpl(k)
	msgServer := keeper.NewMsgServerImpl(k)

	for i, account := range accounts {
		t.Run(account.name, func(t *testing.T) {
			registered, found := k.GetAccountAddress(f.ctx, creator, account.salt)
			require.True(t, found)
			require.Equal(t, addresses[i], registered)

			res, err := queryServer.AccountAddress(f.ctx, &types.QueryAccountAddressRequest{Sender: creator.String(), Salt: account.salt})
			require.NoError(t, err)
			require.True(t, res.Registered)
			require.Equal(t, addresses[i].String(), res.Address)

			_, err = msgServer.RegisterAccount(f.ctx, &types.MsgRegisterAccount{
				Sender: creator.String(),
				CodeID: f.codeID,
				Msg:    mustMarshalAccountInitMsg(t),
				Salt:   account.salt,
			})
			require.ErrorIs(t, err, types.ErrAccountAlreadyRegistered)
		})
	}
}

func TestBackfillChecksumAccountAddressesSkipsWhatItCannotVerify(t *testing.T) {
	f := newChecksumAccountFixture(t)
	k := f.app.AbstractAccountKeeper
	creator := xionapp.RandomAccAddress()

	// Not checksum-derived: an account registered at the fixed-hash address.
	fixedHashSalt := sha256Of([]byte("fixed-hash"))
	fixedHashAddress := wasmkeeper.BuildContractAddressPredictable(addressDerivationHash, creator, fixedHashSalt, nil)
	f.importAccount(t, fixedHashAddress, creator, f.codeID, secp256k1InitMsg("fixed-hash"), true)

	// Not an abstract account.
	plainSalt := sha256Of([]byte("plain"))
	plainAddress := wasmkeeper.BuildContractAddressPredictable(f.checksum, creator, plainSalt, nil)
	f.importAccount(t, plainAddress, creator, f.codeID, secp256k1InitMsg("plain"), false)

	// No recognised authenticator in the INIT message.
	f.importChecksumAccount(t, creator, sha256Of([]byte("pubkey-only")), []byte(`{"pubkey":"cHVia2V5LW9ubHk="}`))

	// Slot already taken by another account.
	takenSalt := sha256Of([]byte("taken"))
	f.importChecksumAccount(t, creator, takenSalt, secp256k1InitMsg("taken"))
	other := xionapp.RandomAccAddress()
	k.SetAccountAddress(f.ctx, creator, takenSalt, other)

	// Instantiated with another code, then migrated to this one.
	otherCodeID, _, err := k.ContractKeeper().Create(f.ctx, user, accountWasm(), nil)
	require.NoError(t, err)
	migratedSalt := sha256Of([]byte("migrated"))
	migratedAddress := wasmkeeper.BuildContractAddressPredictable(f.checksum, creator, migratedSalt, nil)
	f.importAccount(t, migratedAddress, creator, otherCodeID, secp256k1InitMsg("migrated"), true, wasmtypes.ContractCodeHistoryEntry{
		Operation: wasmtypes.ContractCodeHistoryOperationTypeMigrate,
		CodeID:    f.codeID,
		Updated:   &wasmtypes.AbsoluteTxPosition{BlockHeight: 2, TxIndex: 1},
		Msg:       []byte(`{}`),
	})

	result, err := k.BackfillChecksumAccountAddresses(f.ctx, f.codeID, f.checksum)
	require.NoError(t, err)
	require.Equal(t, keeper.ChecksumAccountBackfill{Skipped: 5}, result)

	for _, salt := range [][]byte{fixedHashSalt, plainSalt, sha256Of([]byte("pubkey-only")), migratedSalt} {
		_, found := k.GetAccountAddress(f.ctx, creator, salt)
		require.False(t, found)
	}
	taken, found := k.GetAccountAddress(f.ctx, creator, takenSalt)
	require.True(t, found)
	require.Equal(t, other, taken)
}

func TestBackfillChecksumAccountAddressesIsIdempotent(t *testing.T) {
	f := newChecksumAccountFixture(t)
	f.importChecksumAccount(t, xionapp.RandomAccAddress(), sha256Of([]byte(testPubkey)), secp256k1InitMsg(testPubkey))

	first, err := f.app.AbstractAccountKeeper.BackfillChecksumAccountAddresses(f.ctx, f.codeID, f.checksum)
	require.NoError(t, err)
	require.Equal(t, keeper.ChecksumAccountBackfill{Registered: 1}, first)

	second, err := f.app.AbstractAccountKeeper.BackfillChecksumAccountAddresses(f.ctx, f.codeID, f.checksum)
	require.NoError(t, err)
	require.Equal(t, keeper.ChecksumAccountBackfill{Skipped: 1}, second)
}

func TestBackfillChecksumAccountAddressesRefusesAnUnexpectedCode(t *testing.T) {
	f := newChecksumAccountFixture(t)
	k := f.app.AbstractAccountKeeper

	_, err := k.BackfillChecksumAccountAddresses(f.ctx, f.codeID+1, f.checksum)
	require.ErrorIs(t, err, types.ErrCodeIDNotFound)

	wrongChecksum := sha256Of([]byte("another code"))
	_, err = k.BackfillChecksumAccountAddresses(f.ctx, f.codeID, wrongChecksum)
	require.ErrorIs(t, err, types.ErrInvalidAccountAddressRegistry)
}

func TestChecksumAccountSalt(t *testing.T) {
	ethAddressBytes := mustHex(t, "690e03b98d469fb04396570051b3ff31ac50a8ec")

	cases := []struct {
		name string
		msg  string
		salt []byte
	}{
		{"Secp256K1 hashes the pubkey string", string(secp256k1InitMsg(testPubkey)), sha256Of([]byte(testPubkey))},
		{"EthWallet hashes the address bytes", string(ethWalletInitMsg(testEthAddress)), sha256Of(ethAddressBytes)},
		{"EthWallet without 0x", string(ethWalletInitMsg("690e03b98d469fb04396570051b3ff31ac50a8ec")), sha256Of(ethAddressBytes)},
		{"Jwt hashes aud.sub", string(jwtInitMsg("aud", "sub")), sha256Of([]byte("aud.sub"))},
		{"EthWallet with bad hex", string(ethWalletInitMsg("0xzz")), nil},
		{"EthWallet of the wrong length", string(ethWalletInitMsg("0x6900")), nil},
		{"Secp256K1 without a pubkey", `{"authenticator":{"Secp256K1":{"id":0}}}`, nil},
		{"Jwt without a sub", `{"authenticator":{"Jwt":{"id":0,"aud":"aud"}}}`, nil},
		{"Jwt without an aud", `{"authenticator":{"Jwt":{"id":0,"sub":"sub"}}}`, nil},
		{"unknown authenticator kind", `{"authenticator":{"Passkey":{"id":0,"url":"u"}}}`, nil},
		{"two authenticators", `{"authenticator":{"Jwt":{"aud":"a","sub":"s"},"Secp256K1":{"pubkey":"p"}}}`, nil},
		{"no authenticator", `{"pubkey":"cA=="}`, nil},
		{"authenticator is not an object", `{"authenticator":{"Secp256K1":"p"}}`, nil},
		{"not JSON", `not json`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			salt, ok := keeper.ChecksumAccountSalt([]byte(tc.msg))
			require.Equal(t, tc.salt != nil, ok)
			require.Equal(t, tc.salt, salt)
		})
	}
}

func TestGenesisRoundTripsABackfilledChecksumAccount(t *testing.T) {
	creator := xionapp.RandomAccAddress()
	salt := sha256Of([]byte(testPubkey))
	initMsg := secp256k1InitMsg(testPubkey)
	params, err := types.NewParamsWithAddressDerivationHash(true, nil, types.DefaultMaxGas, types.DefaultMaxGas, addressDerivationHash)
	require.NoError(t, err)

	source := newChecksumAccountFixture(t)
	require.NoError(t, source.app.AbstractAccountKeeper.SetParams(source.ctx, params))
	address := source.importChecksumAccount(t, creator, salt, initMsg)
	_, err = source.app.AbstractAccountKeeper.BackfillChecksumAccountAddresses(source.ctx, source.codeID, source.checksum)
	require.NoError(t, err)
	exported := source.app.AbstractAccountKeeper.ExportGenesis(source.ctx)
	require.Len(t, exported.AccountAddresses, 1)

	target := newChecksumAccountFixture(t)
	require.Equal(t, source.checksum, target.checksum)
	target.importAccount(t, address, creator, target.codeID, initMsg, true)
	require.NotPanics(t, func() { target.app.AbstractAccountKeeper.InitGenesis(target.ctx, exported) })

	registered, found := target.app.AbstractAccountKeeper.GetAccountAddress(target.ctx, creator, salt)
	require.True(t, found)
	require.Equal(t, address, registered)
	require.Equal(t, exported, target.app.AbstractAccountKeeper.ExportGenesis(target.ctx))
}

func TestInitGenesisRejectsAnEntryNotDerivedFromItsInitCode(t *testing.T) {
	f := newChecksumAccountFixture(t)
	params, err := types.NewParamsWithAddressDerivationHash(true, nil, types.DefaultMaxGas, types.DefaultMaxGas, addressDerivationHash)
	require.NoError(t, err)
	require.NoError(t, f.app.AbstractAccountKeeper.SetParams(f.ctx, params))

	creator := xionapp.RandomAccAddress()
	address := f.importChecksumAccount(t, creator, sha256Of([]byte(testPubkey)), secp256k1InitMsg(testPubkey))
	otherSalt := sha256Of([]byte("other"))
	predicted, err := f.app.AbstractAccountKeeper.PredictAccountAddress(f.ctx, creator, otherSalt)
	require.NoError(t, err)

	gs := types.NewGenesisState(1, params)
	gs.AccountAddresses = []*types.AccountAddress{{Sender: creator.String(), Salt: otherSalt, Address: address.String()}}
	require.PanicsWithError(t,
		"genesis account address "+address.String()+" does not match derived address "+predicted.String()+": invalid account address registry entry",
		func() { f.app.AbstractAccountKeeper.InitGenesis(f.ctx, gs) },
	)
}

func accountWasm() []byte {
	return testdata.AccountWasm
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}
