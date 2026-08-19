package keeper_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	wasmtestdata "github.com/CosmWasm/wasmd/x/wasm/keeper/testdata"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/burnt-labs/xion/x/abstractaccount/simapp"
	simapptesting "github.com/burnt-labs/xion/x/abstractaccount/simapp/testing"
	"github.com/burnt-labs/xion/x/abstractaccount/keeper"
	"github.com/burnt-labs/xion/x/abstractaccount/testdata"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

type AccountInitMsg struct {
	PubKey []byte `json:"pubkey"`
}

var (
	user                  = simapptesting.MakeRandomAddress()
	userInitialBalance    = sdk.NewCoins(sdk.NewCoin(simapptesting.DefaultBondDenom, math.NewInt(123456)))
	acctRegisterFunds     = sdk.NewCoins(sdk.NewCoin(simapptesting.DefaultBondDenom, math.NewInt(88888)))
	addressDerivationHash = bytes.Repeat([]byte{0xA5}, wasmtypes.ContractAddrLen)
)

// ------------------------------- UpdateParams --------------------------------

func TestUpdateParams(t *testing.T) {
	for _, tc := range []struct {
		desc      string
		sender    string
		newParams *types.Params
		expErr    bool
	}{
		{
			desc:      "sender is not authority",
			sender:    user.String(),
			newParams: types.DefaultParams(),
			expErr:    true,
		},
		{
			desc:      "invalid params",
			sender:    simapp.Authority,
			newParams: &types.Params{MaxGasBefore: 88888, MaxGasAfter: 0},
			expErr:    true,
		},
		{
			desc:      "sender is authority and params are valid",
			sender:    simapp.Authority,
			newParams: &types.Params{MaxGasBefore: 88888, MaxGasAfter: 99999},
			expErr:    false,
		},
	} {
		app := simapptesting.MakeMockApp([]banktypes.Balance{})
		ctx := app.NewContext(false)

		msgServer := keeper.NewMsgServerImpl(app.AbstractAccountKeeper)

		paramsBefore, err1 := app.AbstractAccountKeeper.GetParams(ctx)
		require.NoError(t, err1)

		_, err2 := msgServer.UpdateParams(ctx, &types.MsgUpdateParams{
			Sender: tc.sender,
			Params: tc.newParams,
		})

		paramsAfter, err3 := app.AbstractAccountKeeper.GetParams(ctx)
		require.NoError(t, err3)

		if tc.expErr {
			require.Error(t, err2)
			require.Equal(t, paramsBefore, paramsAfter)
		} else {
			require.NoError(t, err2)
			require.Equal(t, tc.newParams, paramsAfter)
		}
	}
}

func TestUpdateParamsAddressDerivationHash(t *testing.T) {
	app := simapptesting.MakeSimpleMockApp()
	ctx := app.NewContext(false)
	k := app.AbstractAccountKeeper

	msgServer := keeper.NewMsgServerImpl(k)

	invalidHash := types.DefaultParams()
	invalidHash.AddressDerivationHash = []byte("too-short")
	_, err := msgServer.UpdateParams(ctx, &types.MsgUpdateParams{Sender: simapp.Authority, Params: invalidHash})
	require.ErrorIs(t, err, types.ErrInvalidAddressDerivationHash)

	configured, err := types.NewParamsWithAddressDerivationHash(
		false,
		nil,
		types.DefaultMaxGas,
		types.DefaultMaxGas,
		addressDerivationHash,
	)
	require.NoError(t, err)
	_, err = msgServer.UpdateParams(ctx, &types.MsgUpdateParams{Sender: simapp.Authority, Params: configured})
	require.NoError(t, err)

	changedHash := *configured
	changedHash.AddressDerivationHash = bytes.Repeat([]byte{0xB6}, wasmtypes.ContractAddrLen)
	_, err = msgServer.UpdateParams(ctx, &types.MsgUpdateParams{Sender: simapp.Authority, Params: &changedHash})
	require.ErrorIs(t, err, types.ErrImmutableAddressHash)
}

func TestRegistrationPausePreservesAddressQueries(t *testing.T) {
	app := simapptesting.MakeMockApp([]banktypes.Balance{{
		Address: user.String(),
		Coins:   userInitialBalance,
	}})
	ctx := app.NewContext(false).WithBlockTime(time.Now())
	k := app.AbstractAccountKeeper

	codeID, err := storeCode(ctx, k.ContractKeeper())
	require.NoError(t, err)
	params, err := types.NewParamsWithAddressDerivationHash(
		true, nil, types.DefaultMaxGas, types.DefaultMaxGas, addressDerivationHash,
	)
	require.NoError(t, err)
	require.NoError(t, k.SetParams(ctx, params))

	msgServer := keeper.NewMsgServerImpl(k)
	registered, err := msgServer.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender: user.String(),
		CodeID: codeID,
		Msg:    mustMarshalAccountInitMsg(t),
		Salt:   []byte("registered-before-pause"),
	})
	require.NoError(t, err)

	paused := *params
	paused.RegistrationEnabled = false
	_, err = msgServer.UpdateParams(ctx, &types.MsgUpdateParams{
		Sender: simapp.Authority,
		Params: &paused,
	})
	require.NoError(t, err)

	salt := []byte("predict-during-pause")
	predicted, err := k.PredictAccountAddress(ctx, user, salt)
	require.NoError(t, err)
	queried, err := keeper.NewQueryServerImpl(k).AccountAddress(ctx, &types.QueryAccountAddressRequest{
		Sender: user.String(),
		Salt:   salt,
	})
	require.NoError(t, err)
	require.Equal(t, predicted.String(), queried.Address)
	require.False(t, queried.Registered)

	_, err = msgServer.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender: user.String(),
		CodeID: codeID,
		Msg:    mustMarshalAccountInitMsg(t),
		Salt:   salt,
	})
	require.ErrorIs(t, err, types.ErrRegistrationDisabled)

	require.NotEmpty(t, registered.Address)

	resumed := paused
	resumed.RegistrationEnabled = true
	_, err = msgServer.UpdateParams(ctx, &types.MsgUpdateParams{
		Sender: simapp.Authority,
		Params: &resumed,
	})
	require.NoError(t, err)
	res, err := msgServer.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender: user.String(),
		CodeID: codeID,
		Msg:    mustMarshalAccountInitMsg(t),
		Salt:   salt,
	})
	require.NoError(t, err)
	require.Equal(t, predicted.String(), res.Address)
}

// ------------------------------ RegisterAccount ------------------------------

func TestRegisterAccount(t *testing.T) {
	for _, tc := range []struct {
		allowAllCodeIDs bool
		allowedCodeIDs  []uint64
	}{
		{
			allowAllCodeIDs: true,
			allowedCodeIDs:  []uint64{},
		},
		{
			allowAllCodeIDs: false,
			allowedCodeIDs:  []uint64{1, 69, 420},
		},
	} {
		app := simapptesting.MakeMockApp([]banktypes.Balance{
			{
				Address: user.String(),
				Coins:   userInitialBalance,
			},
		})

		ctx := app.NewContext(false).WithBlockTime(time.Now())

		k := app.AbstractAccountKeeper

		// store code
		codeID, err := storeCode(ctx, k.ContractKeeper())
		require.NoError(t, err)
		require.Equal(t, uint64(1), codeID)

		params, err := types.NewParamsWithAddressDerivationHash(
			tc.allowAllCodeIDs,
			tc.allowedCodeIDs,
			types.DefaultMaxGas,
			types.DefaultMaxGas,
			addressDerivationHash,
		)
		require.NoError(t, err)
		require.NoError(t, k.SetParams(ctx, params))

		// register account
		accAddr, err := registerAccount(ctx, keeper.NewMsgServerImpl(k), codeID)

		require.NoError(t, err)

		// check the contract info is correct
		contractInfo := app.WasmKeeper.GetContractInfo(ctx, accAddr)
		require.Equal(t, codeID, contractInfo.CodeID)
		require.Equal(t, user.String(), contractInfo.Creator)
		require.Equal(t, accAddr.String(), contractInfo.Admin)
		require.Equal(t, fmt.Sprintf("%s/%d", types.ModuleName, k.GetNextAccountID(ctx)-1), contractInfo.Label)

		// make sure an AbstractAccount has been created
		_, ok := app.AccountKeeper.GetAccount(ctx, accAddr).(*types.AbstractAccount)
		require.True(t, ok)

		// make sure the contract has received the funds
		balance := app.BankKeeper.GetAllBalances(ctx, accAddr)
		require.Equal(t, acctRegisterFunds, balance)
	}
}

func TestRegisterAccountUsesFixedAddressHashAndRegistry(t *testing.T) {
	app := simapptesting.MakeMockApp([]banktypes.Balance{{
		Address: user.String(),
		Coins:   userInitialBalance,
	}})
	ctx := app.NewContext(false).WithBlockTime(time.Now())
	k := app.AbstractAccountKeeper

	codeID, err := storeCode(ctx, k.ContractKeeper())
	require.NoError(t, err)

	params, err := types.NewParamsWithAddressDerivationHash(
		true,
		nil,
		types.DefaultMaxGas,
		types.DefaultMaxGas,
		addressDerivationHash,
	)
	require.NoError(t, err)
	require.NoError(t, k.SetParams(ctx, params))

	salt := []byte("chain-owned-address")
	predicted, err := k.PredictAccountAddress(ctx, user, salt)
	require.NoError(t, err)

	queryServer := keeper.NewQueryServerImpl(k)
	before, err := queryServer.AccountAddress(ctx, &types.QueryAccountAddressRequest{
		Sender: user.String(),
		Salt:   salt,
	})
	require.NoError(t, err)
	require.Equal(t, predicted.String(), before.Address)
	require.False(t, before.Registered)

	msgBytes, err := json.Marshal(&AccountInitMsg{PubKey: simapptesting.MakeRandomPubKey().Bytes()})
	require.NoError(t, err)
	res, err := keeper.NewMsgServerImpl(k).RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender: user.String(),
		CodeID: codeID,
		Msg:    msgBytes,
		Salt:   salt,
	})
	require.NoError(t, err)
	require.Equal(t, predicted.String(), res.Address)

	contractInfo := app.WasmKeeper.GetContractInfo(ctx, predicted)
	require.NotNil(t, contractInfo)
	require.Equal(t, codeID, contractInfo.CodeID)
	require.Equal(t, predicted.String(), contractInfo.Admin)

	registeredAddress, found := k.GetAccountAddress(ctx, user, salt)
	require.True(t, found)
	require.Equal(t, predicted, registeredAddress)

	after, err := queryServer.AccountAddress(ctx, &types.QueryAccountAddressRequest{
		Sender: user.String(),
		Salt:   salt,
	})
	require.NoError(t, err)
	require.True(t, after.Registered)
	require.Equal(t, predicted.String(), after.Address)

	_, err = keeper.NewMsgServerImpl(k).RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender: user.String(),
		CodeID: codeID,
		Msg:    msgBytes,
		Salt:   salt,
	})
	require.ErrorIs(t, err, types.ErrAccountAlreadyRegistered)
}

func TestRegisterAccountInstantiationFailureIsAtomic(t *testing.T) {
	app := simapptesting.MakeMockApp([]banktypes.Balance{{
		Address: user.String(),
		Coins:   userInitialBalance,
	}})
	ctx := app.NewContext(false).WithBlockTime(time.Now())
	k := app.AbstractAccountKeeper

	codeID, err := storeCode(ctx, k.ContractKeeper())
	require.NoError(t, err)
	params, err := types.NewParamsWithAddressDerivationHash(
		true, nil, types.DefaultMaxGas, types.DefaultMaxGas, addressDerivationHash,
	)
	require.NoError(t, err)
	require.NoError(t, k.SetParams(ctx, params))

	salt := []byte("instantiation-must-succeed")
	predicted, err := k.PredictAccountAddress(ctx, user, salt)
	require.NoError(t, err)

	cacheCtx, _ := ctx.CacheContext()
	_, err = keeper.NewMsgServerImpl(k).RegisterAccount(cacheCtx, &types.MsgRegisterAccount{
		Sender: user.String(),
		CodeID: codeID,
		Msg:    []byte(`{"invalid":true}`),
		Salt:   salt,
	})
	require.Error(t, err)

	_, found := k.GetAccountAddress(ctx, user, salt)
	require.False(t, found)
	require.Nil(t, app.WasmKeeper.GetContractInfo(ctx, predicted))
	require.Nil(t, app.AccountKeeper.GetAccount(ctx, predicted))
}

func TestRegisterAccountDirectlyInstantiatesCallerSelectedAllowedCodeID(t *testing.T) {
	app := simapptesting.MakeMockApp([]banktypes.Balance{{
		Address: user.String(),
		Coins:   userInitialBalance,
	}})
	ctx := app.NewContext(false).WithBlockTime(time.Now())
	k := app.AbstractAccountKeeper

	implementationCodeID, _, err := k.ContractKeeper().Create(
		ctx,
		user,
		wasmtestdata.IBCReflectContractWasm(),
		nil,
	)
	require.NoError(t, err)
	params, err := types.NewParamsWithAddressDerivationHash(
		false,
		[]uint64{implementationCodeID},
		types.DefaultMaxGas,
		types.DefaultMaxGas,
		addressDerivationHash,
	)
	require.NoError(t, err)
	require.NoError(t, k.SetParams(ctx, params))

	salt := []byte("caller-selected-implementation")
	predicted, err := k.PredictAccountAddress(ctx, user, salt)
	require.NoError(t, err)
	res, err := keeper.NewMsgServerImpl(k).RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender: user.String(),
		CodeID: implementationCodeID,
		Msg:    mustMarshalReflectInitMsg(t, implementationCodeID),
		Salt:   salt,
	})
	require.NoError(t, err)
	require.Equal(t, predicted.String(), res.Address)

	contractInfo := app.WasmKeeper.GetContractInfo(ctx, predicted)
	require.NotNil(t, contractInfo)
	require.Equal(t, implementationCodeID, contractInfo.CodeID)
	require.Equal(t, predicted.String(), contractInfo.Admin)
	_, ok := app.AccountKeeper.GetAccount(ctx, predicted).(*types.AbstractAccount)
	require.True(t, ok)
}

func TestRegisterAccountAddressDoesNotDependOnCodeID(t *testing.T) {
	type result struct {
		address sdk.AccAddress
		codeID  uint64
	}
	register := func(t *testing.T, useReflect bool) result {
		t.Helper()
		app := simapptesting.MakeMockApp([]banktypes.Balance{{
			Address: user.String(),
			Coins:   userInitialBalance,
		}})
		ctx := app.NewContext(false).WithBlockTime(time.Now())
		k := app.AbstractAccountKeeper
		accountCodeID, err := storeCode(ctx, k.ContractKeeper())
		require.NoError(t, err)
		reflectCodeID, _, err := k.ContractKeeper().Create(ctx, user, wasmtestdata.IBCReflectContractWasm(), nil)
		require.NoError(t, err)

		params, err := types.NewParamsWithAddressDerivationHash(
			true, nil, types.DefaultMaxGas, types.DefaultMaxGas, addressDerivationHash,
		)
		require.NoError(t, err)
		require.NoError(t, k.SetParams(ctx, params))

		codeID := accountCodeID
		msg := mustMarshalAccountInitMsg(t)
		if useReflect {
			codeID = reflectCodeID
			msg = mustMarshalReflectInitMsg(t, reflectCodeID)
		}
		res, err := keeper.NewMsgServerImpl(k).RegisterAccount(ctx, &types.MsgRegisterAccount{
			Sender: user.String(),
			CodeID: codeID,
			Msg:    msg,
			Salt:   []byte("same-address-across-code-ids"),
		})
		require.NoError(t, err)
		address, err := sdk.AccAddressFromBech32(res.Address)
		require.NoError(t, err)
		require.Equal(t, codeID, app.WasmKeeper.GetContractInfo(ctx, address).CodeID)

		return result{address: address, codeID: codeID}
	}

	account := register(t, false)
	reflect := register(t, true)
	require.NotEqual(t, account.codeID, reflect.codeID)
	require.Equal(t, account.address, reflect.address)
}

func TestRegisterAccountRejectsInvalidRequestedImplementationsBeforeInstantiation(t *testing.T) {
	app := simapptesting.MakeMockApp([]banktypes.Balance{{
		Address: user.String(),
		Coins:   userInitialBalance,
	}})
	ctx := app.NewContext(false).WithBlockTime(time.Now())
	k := app.AbstractAccountKeeper

	codeID, err := storeCode(ctx, k.ContractKeeper())
	require.NoError(t, err)
	params, err := types.NewParamsWithAddressDerivationHash(
		false,
		[]uint64{codeID},
		types.DefaultMaxGas,
		types.DefaultMaxGas,
		addressDerivationHash,
	)
	require.NoError(t, err)
	require.NoError(t, k.SetParams(ctx, params))
	msgServer := keeper.NewMsgServerImpl(k)

	unallowedSalt := []byte("unallowed-implementation")
	_, err = msgServer.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender: user.String(),
		CodeID: codeID + 1,
		Msg:    mustMarshalAccountInitMsg(t),
		Salt:   unallowedSalt,
	})
	require.ErrorIs(t, err, types.ErrNotAllowedCodeID)
	_, found := k.GetAccountAddress(ctx, user, unallowedSalt)
	require.False(t, found)

	allowAll := *params
	allowAll.AllowAllCodeIDs = true
	allowAll.AllowedCodeIDs = nil
	require.NoError(t, k.SetParams(ctx, &allowAll))
	missingSalt := []byte("missing-implementation")
	_, err = msgServer.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender: user.String(),
		CodeID: 999,
		Msg:    mustMarshalAccountInitMsg(t),
		Salt:   missingSalt,
	})
	require.ErrorIs(t, err, types.ErrCodeIDNotFound)
	_, found = k.GetAccountAddress(ctx, user, missingSalt)
	require.False(t, found)
}

func TestAccountAddressRecognizesCanonicalPreRegistryAccount(t *testing.T) {
	app := simapptesting.MakeMockApp([]banktypes.Balance{{
		Address: user.String(),
		Coins:   userInitialBalance,
	}})
	ctx := app.NewContext(false).WithBlockTime(time.Now())
	k := app.AbstractAccountKeeper

	codeID, err := storeCode(ctx, k.ContractKeeper())
	require.NoError(t, err)
	params, err := types.NewParamsWithAddressDerivationHash(
		true, nil, types.DefaultMaxGas, types.DefaultMaxGas, addressDerivationHash,
	)
	require.NoError(t, err)
	require.NoError(t, k.SetParams(ctx, params))

	salt := []byte("historical-account")
	msgBytes, err := json.Marshal(&AccountInitMsg{PubKey: simapptesting.MakeRandomPubKey().Bytes()})
	require.NoError(t, err)
	address, _, err := k.ContractKeeper().Instantiate2WithAddressHash(
		ctx,
		codeID,
		addressDerivationHash,
		user,
		user,
		msgBytes,
		"historical",
		nil,
		salt,
	)
	require.NoError(t, err)
	baseAccount := app.AccountKeeper.GetAccount(ctx, address)
	require.NotNil(t, baseAccount)
	app.AccountKeeper.SetAccount(ctx, types.NewAbstractAccountFromAccount(baseAccount))

	_, found := k.GetAccountAddress(ctx, user, salt)
	require.False(t, found)
	res, err := keeper.NewQueryServerImpl(k).AccountAddress(ctx, &types.QueryAccountAddressRequest{
		Sender: user.String(),
		Salt:   salt,
	})
	require.NoError(t, err)
	require.True(t, res.Registered)
	require.Equal(t, address.String(), res.Address)
}

// ---------------------------------- Helpers ----------------------------------

func storeCode(ctx sdk.Context, contractKeeper wasmtypes.ContractOpsKeeper) (uint64, error) {
	codeID, _, err := contractKeeper.Create(ctx, user, testdata.AccountWasm, nil)

	return codeID, err
}

func registerAccount(ctx sdk.Context, msgServer types.MsgServer, codeID uint64) (sdk.AccAddress, error) {
	msgBytes, err := json.Marshal(&AccountInitMsg{
		PubKey: simapptesting.MakeRandomPubKey().Bytes(),
	})
	if err != nil {
		return nil, err
	}

	res, err := msgServer.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender: user.String(),
		CodeID: codeID,
		Msg:    msgBytes,
		Funds:  acctRegisterFunds,
		Salt:   []byte("hello"),
	})
	if err != nil {
		return nil, err
	}

	return sdk.AccAddressFromBech32(res.Address)
}

func mustMarshalAccountInitMsg(t *testing.T) []byte {
	t.Helper()
	msgBytes, err := json.Marshal(&AccountInitMsg{PubKey: simapptesting.MakeRandomPubKey().Bytes()})
	require.NoError(t, err)

	return msgBytes
}

func mustMarshalReflectInitMsg(t *testing.T, reflectCodeID uint64) []byte {
	t.Helper()
	msgBytes, err := json.Marshal(struct {
		ReflectCodeID uint64 `json:"reflect_code_id"`
	}{ReflectCodeID: reflectCodeID})
	require.NoError(t, err)

	return msgBytes
}

// ----------------------------- Additional Tests for 100% Coverage -----------------------------

func TestRegisterAccountErrors(t *testing.T) {
	app := simapptesting.MakeMockApp([]banktypes.Balance{
		{
			Address: user.String(),
			Coins:   userInitialBalance,
		},
	})

	ctx := app.NewContext(false).WithBlockTime(time.Now())

	// Set up params allowing code ID 1
	params, err := types.NewParamsWithAddressDerivationHash(false, []uint64{1}, types.DefaultMaxGas, types.DefaultMaxGas, addressDerivationHash)
	require.NoError(t, err)

	k := app.AbstractAccountKeeper
	err = k.SetParams(ctx, params)
	require.NoError(t, err)

	msgServer := keeper.NewMsgServerImpl(k)

	// Test case 1: Invalid sender address
	t.Run("invalid sender address", func(t *testing.T) {
		msgBytes, err := json.Marshal(&AccountInitMsg{
			PubKey: simapptesting.MakeRandomPubKey().Bytes(),
		})
		require.NoError(t, err)

		_, err = msgServer.RegisterAccount(ctx, &types.MsgRegisterAccount{
			Sender: "invalid-address",
			CodeID: 1,
			Msg:    msgBytes,
			Funds:  acctRegisterFunds,
			Salt:   []byte("test"),
		})
		require.Error(t, err)
	})

	// Test case 2: Contract instantiation failure (using invalid code ID that doesn't exist)
	t.Run("contract instantiation failure", func(t *testing.T) {
		msgBytes, err := json.Marshal(&AccountInitMsg{
			PubKey: simapptesting.MakeRandomPubKey().Bytes(),
		})
		require.NoError(t, err)

		_, err = msgServer.RegisterAccount(ctx, &types.MsgRegisterAccount{
			Sender: user.String(),
			CodeID: 1,
			Msg:    msgBytes,
			Funds:  acctRegisterFunds,
			Salt:   []byte("test"),
		})
		require.Error(t, err)
	})
}
