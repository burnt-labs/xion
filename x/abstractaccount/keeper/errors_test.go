package keeper_test

import (
	"encoding/binary"
	"errors"
	"testing"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"

	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	xionapp "github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/x/abstractaccount/keeper"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

// The tests in this file reach the keeper's defensive branches: missing or
// undecodable module params, malformed registry keys, and contract keeper
// failures during registration.

func moduleStore(app *xionapp.WasmApp, ctx sdk.Context) storetypes.KVStore {
	return ctx.KVStore(app.GetKey(types.StoreKey))
}

func deleteParams(app *xionapp.WasmApp, ctx sdk.Context) {
	moduleStore(app, ctx).Delete(types.KeyParams)
}

// corruptParams stores bytes that are not a valid protobuf encoding of Params.
func corruptParams(app *xionapp.WasmApp, ctx sdk.Context) {
	moduleStore(app, ctx).Set(types.KeyParams, []byte{0xff, 0xff, 0xff})
}

func configuredParams(t *testing.T) *types.Params {
	t.Helper()
	params, err := types.NewParamsWithAddressDerivationHash(true, nil, types.DefaultMaxGas, types.DefaultMaxGas, addressDerivationHash)
	require.NoError(t, err)
	return params
}

// ------------------------------- Params ---------------------------------

func TestGetParamsMissing(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)
	deleteParams(app, ctx)

	params, err := app.AbstractAccountKeeper.GetParams(ctx)
	require.ErrorIs(t, err, sdkerrors.ErrNotFound)
	require.Nil(t, params)
}

func TestGetParamsUndecodable(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)
	corruptParams(app, ctx)

	params, err := app.AbstractAccountKeeper.GetParams(ctx)
	require.ErrorIs(t, err, types.ErrParsingParams)
	require.Nil(t, params)
}

// --------------------------- Account addresses --------------------------

func TestParseAccountAddressKey(t *testing.T) {
	sender := xionapp.RandomAccAddress()
	salt := []byte("salt")

	validKey := make([]byte, 1+8+len(sender)+len(salt))
	validKey[0] = types.KeyAccountAddressPrefix[0]
	binary.BigEndian.PutUint64(validKey[1:9], uint64(len(sender)))
	copy(validKey[9:], sender)
	copy(validKey[9+len(sender):], salt)

	zeroLength := make([]byte, 9+len(sender))
	zeroLength[0] = types.KeyAccountAddressPrefix[0]
	copy(zeroLength[9:], sender)

	overlongLength := make([]byte, 9+len(sender))
	overlongLength[0] = types.KeyAccountAddressPrefix[0]
	binary.BigEndian.PutUint64(overlongLength[1:9], uint64(len(sender)+1))
	copy(overlongLength[9:], sender)

	wrongPrefix := append([]byte(nil), validKey...)
	wrongPrefix[0] = types.KeyNextAccountID[0]

	for _, tc := range []struct {
		desc   string
		key    []byte
		expErr string
	}{
		{desc: "too short", key: types.KeyAccountAddressPrefix, expErr: "invalid account address registry key"},
		{desc: "wrong prefix", key: wrongPrefix, expErr: "invalid account address registry key"},
		{desc: "zero sender length", key: zeroLength, expErr: "invalid account address registry sender length"},
		{desc: "sender length beyond key", key: overlongLength, expErr: "invalid account address registry sender length"},
		{desc: "well formed", key: validKey},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			gotSender, gotSalt, err := keeper.ParseAccountAddressKey(tc.key)
			if tc.expErr != "" {
				require.EqualError(t, err, tc.expErr)
				require.Nil(t, gotSender)
				require.Nil(t, gotSalt)
				return
			}
			require.NoError(t, err)
			require.Equal(t, sender, gotSender)
			require.Equal(t, salt, gotSalt)
		})
	}
}

func TestIterateAccountAddressesStopsWhenAsked(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)
	k := app.AbstractAccountKeeper

	sender := xionapp.RandomAccAddress()
	k.SetAccountAddress(ctx, sender, []byte("first"), xionapp.RandomAccAddress())
	k.SetAccountAddress(ctx, sender, []byte("second"), xionapp.RandomAccAddress())

	visited := 0
	err := k.IterateAccountAddresses(ctx, func(sdk.AccAddress, []byte, sdk.AccAddress) bool {
		visited++
		return true
	})
	require.NoError(t, err)
	require.Equal(t, 1, visited)
}

func TestIterateAccountAddressesRejectsMalformedKey(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)
	k := app.AbstractAccountKeeper

	// A registry key with a zero sender length cannot have been written by
	// SetAccountAddress; the iterator must surface it instead of skipping it.
	malformed := append(append([]byte(nil), types.KeyAccountAddressPrefix...), make([]byte, 8)...)
	moduleStore(app, ctx).Set(malformed, xionapp.RandomAccAddress())

	visited := 0
	err := k.IterateAccountAddresses(ctx, func(sdk.AccAddress, []byte, sdk.AccAddress) bool {
		visited++
		return false
	})
	require.EqualError(t, err, "invalid account address registry sender length")
	require.Zero(t, visited)

	require.PanicsWithError(t, "invalid account address registry sender length", func() {
		k.ExportGenesis(ctx)
	})
}

func TestPredictAccountAddressErrors(t *testing.T) {
	sender := xionapp.RandomAccAddress()

	t.Run("params missing", func(t *testing.T) {
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)
		deleteParams(app, ctx)

		_, err := app.AbstractAccountKeeper.PredictAccountAddress(ctx, sender, []byte("salt"))
		require.ErrorIs(t, err, sdkerrors.ErrNotFound)
	})

	t.Run("registration not configured", func(t *testing.T) {
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)

		_, err := app.AbstractAccountKeeper.PredictAccountAddress(ctx, sender, []byte("salt"))
		require.ErrorIs(t, err, types.ErrRegistrationNotConfigured)
	})

	t.Run("invalid salt", func(t *testing.T) {
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)
		require.NoError(t, app.AbstractAccountKeeper.SetParams(ctx, configuredParams(t)))

		_, err := app.AbstractAccountKeeper.PredictAccountAddress(ctx, sender, nil)
		require.ErrorIs(t, err, wasmtypes.ErrEmpty)
	})
}

// -------------------------------- Genesis -------------------------------

func TestExportGenesisPanicsWithoutParams(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)
	deleteParams(app, ctx)

	require.PanicsWithError(t, "x/abstractaccount module params: not found", func() {
		app.AbstractAccountKeeper.ExportGenesis(ctx)
	})
}

func TestInitGenesisRejectsMalformedRegistryEntries(t *testing.T) {
	sender := xionapp.RandomAccAddress()

	t.Run("sender is not bech32", func(t *testing.T) {
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)
		gs := types.NewGenesisState(1, configuredParams(t))
		gs.AccountAddresses = []*types.AccountAddress{{
			Sender:  "not-bech32",
			Salt:    []byte("salt"),
			Address: xionapp.RandomAccAddress().String(),
		}}

		require.PanicsWithError(t, "decoding bech32 failed: invalid separator index -1", func() {
			app.AbstractAccountKeeper.InitGenesis(ctx, gs)
		})
	})

	t.Run("address is not bech32", func(t *testing.T) {
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)
		gs := types.NewGenesisState(1, configuredParams(t))
		gs.AccountAddresses = []*types.AccountAddress{{
			Sender:  sender.String(),
			Salt:    []byte("salt"),
			Address: "not-bech32",
		}}

		require.PanicsWithError(t, "decoding bech32 failed: invalid separator index -1", func() {
			app.AbstractAccountKeeper.InitGenesis(ctx, gs)
		})
	})

	t.Run("address cannot be derived", func(t *testing.T) {
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)

		// The genesis params leave address derivation unconfigured, so the
		// registry entry cannot be checked against a predicted address.
		address := xionapp.RandomAccAddress()
		app.AccountKeeper.SetAccount(ctx, app.AccountKeeper.NewAccount(ctx, types.NewAbstractAccount(address.String(), 0, 0)))
		gs := types.NewGenesisState(1, types.DefaultParams())
		gs.AccountAddresses = []*types.AccountAddress{{
			Sender:  sender.String(),
			Salt:    []byte("salt"),
			Address: address.String(),
		}}

		require.PanicsWithError(t, types.ErrRegistrationNotConfigured.Error(), func() {
			app.AbstractAccountKeeper.InitGenesis(ctx, gs)
		})
	})
}

// ------------------------------ MsgServer -------------------------------

func TestUpdateParamsErrors(t *testing.T) {
	t.Run("nil params", func(t *testing.T) {
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)
		msgServer := keeper.NewMsgServerImpl(app.AbstractAccountKeeper)

		_, err := msgServer.UpdateParams(ctx, &types.MsgUpdateParams{Sender: authority, Params: nil})
		require.ErrorIs(t, err, sdkerrors.ErrInvalidRequest)
	})

	t.Run("current params missing", func(t *testing.T) {
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)
		msgServer := keeper.NewMsgServerImpl(app.AbstractAccountKeeper)
		deleteParams(app, ctx)

		_, err := msgServer.UpdateParams(ctx, &types.MsgUpdateParams{Sender: authority, Params: types.DefaultParams()})
		require.ErrorIs(t, err, sdkerrors.ErrNotFound)
	})
}

// registrationFixture is an app with the account code stored and
// registration configured, ready for RegisterAccount calls.
type registrationFixture struct {
	app    *xionapp.WasmApp
	ctx    sdk.Context
	codeID uint64
}

func newRegistrationFixture(t *testing.T) registrationFixture {
	t.Helper()

	app := xionapp.SetupWithBalances(t, []banktypes.Balance{{Address: user.String(), Coins: userInitialBalance}})
	ctx := app.NewContext(false).WithBlockTime(time.Now())
	codeID, err := storeCode(ctx, app.AbstractAccountKeeper.ContractKeeper())
	require.NoError(t, err)
	require.NoError(t, app.AbstractAccountKeeper.SetParams(ctx, configuredParams(t)))

	return registrationFixture{app: app, ctx: ctx, codeID: codeID}
}

func (f registrationFixture) registerMsg(t *testing.T, salt []byte) *types.MsgRegisterAccount {
	t.Helper()
	return &types.MsgRegisterAccount{
		Sender: user.String(),
		CodeID: f.codeID,
		Msg:    mustMarshalAccountInitMsg(t),
		Funds:  acctRegisterFunds,
		Salt:   salt,
	}
}

// contractKeeperStub forwards to the real contract keeper except for the
// operations a test overrides.
type contractKeeperStub struct {
	wasmtypes.ContractOpsKeeperWithAddressHash

	instantiate2 func(ctx sdk.Context, codeID uint64, addressHash []byte, creator, admin sdk.AccAddress, initMsg []byte, label string, deposit sdk.Coins, salt []byte) (sdk.AccAddress, []byte, error)
	updateAdmin  func(ctx sdk.Context, contractAddress, caller, newAdmin sdk.AccAddress) error
}

func (s contractKeeperStub) Instantiate2WithAddressHash(
	ctx sdk.Context, codeID uint64, addressHash []byte, creator, admin sdk.AccAddress,
	initMsg []byte, label string, deposit sdk.Coins, salt []byte,
) (sdk.AccAddress, []byte, error) {
	if s.instantiate2 != nil {
		return s.instantiate2(ctx, codeID, addressHash, creator, admin, initMsg, label, deposit, salt)
	}
	return s.ContractOpsKeeperWithAddressHash.Instantiate2WithAddressHash(ctx, codeID, addressHash, creator, admin, initMsg, label, deposit, salt)
}

func (s contractKeeperStub) UpdateContractAdmin(ctx sdk.Context, contractAddress, caller, newAdmin sdk.AccAddress) error {
	if s.updateAdmin != nil {
		return s.updateAdmin(ctx, contractAddress, caller, newAdmin)
	}
	return s.ContractOpsKeeperWithAddressHash.UpdateContractAdmin(ctx, contractAddress, caller, newAdmin)
}

// keeperWithContractKeeper builds a module keeper over the app's own stores
// and account keeper, with the contract keeper replaced by stub.
func (f registrationFixture) keeperWithContractKeeper(stub contractKeeperStub) keeper.Keeper {
	stub.ContractOpsKeeperWithAddressHash = f.app.AbstractAccountKeeper.ContractKeeper()
	return keeper.NewKeeper(
		f.app.AppCodec(),
		f.app.GetKey(types.StoreKey),
		f.app.GetTKey(types.TransientStoreKey),
		f.app.AccountKeeper,
		stub,
		&f.app.WasmKeeper,
		authority,
	)
}

func TestRegisterAccountWithoutParams(t *testing.T) {
	f := newRegistrationFixture(t)
	msgServer := keeper.NewMsgServerImpl(f.app.AbstractAccountKeeper)
	deleteParams(f.app, f.ctx)

	_, err := msgServer.RegisterAccount(f.ctx, f.registerMsg(t, []byte("salt")))
	require.ErrorIs(t, err, sdkerrors.ErrNotFound)
}

func TestRegisterAccountRejectsEmptySalt(t *testing.T) {
	f := newRegistrationFixture(t)
	msgServer := keeper.NewMsgServerImpl(f.app.AbstractAccountKeeper)

	_, err := msgServer.RegisterAccount(f.ctx, f.registerMsg(t, nil))
	require.ErrorIs(t, err, wasmtypes.ErrEmpty)
}

func TestRegisterAccountRejectsOccupiedPredictedAddress(t *testing.T) {
	f := newRegistrationFixture(t)
	k := f.app.AbstractAccountKeeper
	msgServer := keeper.NewMsgServerImpl(k)
	salt := []byte("occupied")

	// An abstract account already lives at the predicted address, although
	// the (sender, salt) slot was never written to the registry.
	predicted, err := k.PredictAccountAddress(f.ctx, user, salt)
	require.NoError(t, err)
	f.app.AccountKeeper.SetAccount(f.ctx, f.app.AccountKeeper.NewAccount(f.ctx, types.NewAbstractAccount(predicted.String(), 0, 0)))

	_, err = msgServer.RegisterAccount(f.ctx, f.registerMsg(t, salt))
	require.ErrorIs(t, err, types.ErrAccountAlreadyRegistered)
	require.Contains(t, err.Error(), predicted.String())
}

func TestRegisterAccountRejectsUnexpectedInstantiatedAddress(t *testing.T) {
	f := newRegistrationFixture(t)
	elsewhere := xionapp.RandomAccAddress()
	k := f.keeperWithContractKeeper(contractKeeperStub{
		instantiate2: func(sdk.Context, uint64, []byte, sdk.AccAddress, sdk.AccAddress, []byte, string, sdk.Coins, []byte) (sdk.AccAddress, []byte, error) {
			return elsewhere, nil, nil
		},
	})
	msgServer := keeper.NewMsgServerImpl(k)

	_, err := msgServer.RegisterAccount(f.ctx, f.registerMsg(t, []byte("salt")))
	require.Error(t, err)
	require.Contains(t, err.Error(), "instantiated account address "+elsewhere.String()+" does not match predicted address")
}

func TestRegisterAccountAdminUpdateFailure(t *testing.T) {
	f := newRegistrationFixture(t)
	boom := errors.New("admin update refused")
	k := f.keeperWithContractKeeper(contractKeeperStub{
		updateAdmin: func(sdk.Context, sdk.AccAddress, sdk.AccAddress, sdk.AccAddress) error { return boom },
	})
	msgServer := keeper.NewMsgServerImpl(k)

	_, err := msgServer.RegisterAccount(f.ctx, f.registerMsg(t, []byte("salt")))
	require.ErrorIs(t, err, boom)
}

func TestRegisterAccountRequiresBaseAccountAfterInstantiation(t *testing.T) {
	f := newRegistrationFixture(t)
	salt := []byte("salt")
	predicted, err := f.app.AbstractAccountKeeper.PredictAccountAddress(f.ctx, user, salt)
	require.NoError(t, err)

	// Instantiation reports the predicted address without creating any
	// account there, so the base account the module expects is missing.
	k := f.keeperWithContractKeeper(contractKeeperStub{
		instantiate2: func(sdk.Context, uint64, []byte, sdk.AccAddress, sdk.AccAddress, []byte, string, sdk.Coins, []byte) (sdk.AccAddress, []byte, error) {
			return predicted, nil, nil
		},
		updateAdmin: func(sdk.Context, sdk.AccAddress, sdk.AccAddress, sdk.AccAddress) error { return nil },
	})
	msgServer := keeper.NewMsgServerImpl(k)

	_, err = msgServer.RegisterAccount(f.ctx, f.registerMsg(t, salt))
	require.ErrorIs(t, err, types.ErrNotBaseAccount)
}

// ------------------------------ QueryServer -----------------------------

func TestAccountAddressQueryErrors(t *testing.T) {
	t.Run("invalid sender", func(t *testing.T) {
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)
		queryServer := keeper.NewQueryServerImpl(app.AbstractAccountKeeper)

		_, err := queryServer.AccountAddress(ctx, &types.QueryAccountAddressRequest{Sender: "not-bech32", Salt: []byte("salt")})
		require.ErrorContains(t, err, "decoding bech32")
	})

	t.Run("invalid salt", func(t *testing.T) {
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)
		queryServer := keeper.NewQueryServerImpl(app.AbstractAccountKeeper)

		_, err := queryServer.AccountAddress(ctx, &types.QueryAccountAddressRequest{Sender: user.String(), Salt: nil})
		require.ErrorIs(t, err, wasmtypes.ErrEmpty)
	})

	t.Run("address cannot be predicted", func(t *testing.T) {
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)
		queryServer := keeper.NewQueryServerImpl(app.AbstractAccountKeeper)

		_, err := queryServer.AccountAddress(ctx, &types.QueryAccountAddressRequest{Sender: user.String(), Salt: []byte("salt")})
		require.ErrorIs(t, err, types.ErrRegistrationNotConfigured)
	})
}

func TestParamsQueryWithoutParams(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)
	queryServer := keeper.NewQueryServerImpl(app.AbstractAccountKeeper)
	deleteParams(app, ctx)

	_, err := queryServer.Params(ctx, &types.QueryParamsRequest{})
	require.ErrorIs(t, err, sdkerrors.ErrNotFound)
}

// --------------------------- Checksum accounts --------------------------

func TestBackfillChecksumAccountAddressesSkipsUndecodableCreator(t *testing.T) {
	f := newChecksumAccountFixture(t)
	creator := xionapp.RandomAccAddress()
	salt := sha256Of([]byte(testPubkey))
	address := f.importChecksumAccount(t, creator, salt, secp256k1InitMsg(testPubkey))

	// Contract info whose creator does not decode cannot be produced through
	// the wasm keeper; write it to the wasm store directly.
	info := f.app.WasmKeeper.GetContractInfo(f.ctx, address)
	require.NotNil(t, info)
	info.Creator = "not-bech32"
	f.ctx.KVStore(f.app.GetKey(wasmtypes.StoreKey)).Set(wasmtypes.GetContractAddressKey(address), f.app.AppCodec().MustMarshal(info))

	result, err := f.app.AbstractAccountKeeper.BackfillChecksumAccountAddresses(f.ctx, f.codeID, f.checksum)
	require.NoError(t, err)
	require.Equal(t, keeper.ChecksumAccountBackfill{Skipped: 1}, result)

	_, found := f.app.AbstractAccountKeeper.GetAccountAddress(f.ctx, creator, salt)
	require.False(t, found)
}

func TestInitGenesisRejectsAnEntryWhoseInitCodeIsUnknown(t *testing.T) {
	f := newChecksumAccountFixture(t)
	params := configuredParams(t)
	require.NoError(t, f.app.AbstractAccountKeeper.SetParams(f.ctx, params))

	// The contract's INIT history names a code that is not stored, so the
	// checksum the address would have to derive from is unavailable.
	creator := xionapp.RandomAccAddress()
	salt := sha256Of([]byte(testPubkey))
	address := wasmkeeper.BuildContractAddressPredictable(f.checksum, creator, salt, nil)
	unknownCodeID := f.codeID + 1000
	f.importAccount(t, address, creator, unknownCodeID, secp256k1InitMsg(testPubkey), true, wasmtypes.ContractCodeHistoryEntry{
		Operation: wasmtypes.ContractCodeHistoryOperationTypeMigrate,
		CodeID:    f.codeID,
		Updated:   &wasmtypes.AbsoluteTxPosition{BlockHeight: 2, TxIndex: 1},
		Msg:       []byte(`{}`),
	})
	require.Nil(t, f.app.WasmKeeper.GetCodeInfo(f.ctx, unknownCodeID))
	predicted, err := f.app.AbstractAccountKeeper.PredictAccountAddress(f.ctx, creator, salt)
	require.NoError(t, err)

	gs := types.NewGenesisState(1, params)
	gs.AccountAddresses = []*types.AccountAddress{{Sender: creator.String(), Salt: salt, Address: address.String()}}
	require.PanicsWithError(t,
		"genesis account address "+address.String()+" does not match derived address "+predicted.String()+": invalid account address registry entry",
		func() { f.app.AbstractAccountKeeper.InitGenesis(f.ctx, gs) },
	)
}
