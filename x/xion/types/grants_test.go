package types_test

import (
	"math"
	"math/rand"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	xiontypes "github.com/burnt-labs/xion/x/xion/types"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	storetypes "github.com/cosmos/cosmos-sdk/store/rootmulti"
	"github.com/cosmos/cosmos-sdk/store/types"
)

// validatable is an optional interface that can be implemented by an ContractInfoExtension to enable validation
type validatable interface {
	ValidateBasic() error
}

var wasmKey = sdk.NewKVStoreKey(wasmtypes.StoreKey)

func TestContractAuthzFilterValidate(t *testing.T) {
	specs := map[string]struct {
		src    wasmtypes.ContractAuthzFilterX
		expErr bool
	}{
		"allow all": {
			src: &wasmtypes.AllowAllMessagesFilter{},
		},
		"allow keys - single": {
			src: wasmtypes.NewAcceptedMessageKeysFilter("foo"),
		},
		"allow keys - multi": {
			src: wasmtypes.NewAcceptedMessageKeysFilter("foo", "bar"),
		},
		"allow keys - empty": {
			src:    wasmtypes.NewAcceptedMessageKeysFilter(),
			expErr: true,
		},
		"allow keys - duplicates": {
			src:    wasmtypes.NewAcceptedMessageKeysFilter("foo", "foo"),
			expErr: true,
		},
		"allow keys - whitespaces": {
			src:    wasmtypes.NewAcceptedMessageKeysFilter(" foo"),
			expErr: true,
		},
		"allow keys - empty key": {
			src:    wasmtypes.NewAcceptedMessageKeysFilter("", "bar"),
			expErr: true,
		},
		"allow keys - whitespace key": {
			src:    wasmtypes.NewAcceptedMessageKeysFilter(" ", "bar"),
			expErr: true,
		},
		"allow message - single": {
			src: wasmtypes.NewAcceptedMessagesFilter([]byte(`{}`)),
		},
		"allow message - multiple": {
			src: wasmtypes.NewAcceptedMessagesFilter([]byte(`{}`), []byte(`{"foo":"bar"}`)),
		},
		"allow message - multiple with empty": {
			src:    wasmtypes.NewAcceptedMessagesFilter([]byte(`{}`), nil),
			expErr: true,
		},
		"allow message - duplicate": {
			src:    wasmtypes.NewAcceptedMessagesFilter([]byte(`{}`), []byte(`{}`)),
			expErr: true,
		},
		"allow message - non json": {
			src:    wasmtypes.NewAcceptedMessagesFilter([]byte("non-json")),
			expErr: true,
		},
		"allow message - empty": {
			src:    wasmtypes.NewAcceptedMessagesFilter(),
			expErr: true,
		},
		"allow all message - always valid": {
			src: wasmtypes.NewAllowAllMessagesFilter(),
		},
		"undefined - always invalid": {
			src:    &wasmtypes.UndefinedFilter{},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gotErr := spec.src.ValidateBasic()
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
		})
	}
}

func TestContractAuthzFilterAccept(t *testing.T) {
	specs := map[string]struct {
		filter         wasmtypes.ContractAuthzFilterX
		src            wasmtypes.RawContractMessage
		exp            bool
		expGasConsumed sdk.Gas
		expErr         bool
	}{
		"allow all - accepts json obj": {
			filter: &wasmtypes.AllowAllMessagesFilter{},
			src:    []byte(`{}`),
			exp:    true,
		},
		"allow all - accepts json array": {
			filter: &wasmtypes.AllowAllMessagesFilter{},
			src:    []byte(`[{},{}]`),
			exp:    true,
		},
		"allow all - rejects non json msg": {
			filter: &wasmtypes.AllowAllMessagesFilter{},
			src:    []byte(``),
			expErr: true,
		},
		"allowed key - single": {
			filter:         wasmtypes.NewAcceptedMessageKeysFilter("foo"),
			src:            []byte(`{"foo": "bar"}`),
			exp:            true,
			expGasConsumed: sdk.Gas(len(`{"foo": "bar"}`)),
		},
		"allowed key - multiple": {
			filter:         wasmtypes.NewAcceptedMessageKeysFilter("foo", "other"),
			src:            []byte(`{"other": "value"}`),
			exp:            true,
			expGasConsumed: sdk.Gas(len(`{"other": "value"}`)),
		},
		"allowed key - non accepted key": {
			filter:         wasmtypes.NewAcceptedMessageKeysFilter("foo"),
			src:            []byte(`{"bar": "value"}`),
			exp:            false,
			expGasConsumed: sdk.Gas(len(`{"bar": "value"}`)),
		},
		"allowed key - unsupported array msg": {
			filter:         wasmtypes.NewAcceptedMessageKeysFilter("foo", "other"),
			src:            []byte(`[{"foo":"bar"}]`),
			expErr:         false,
			expGasConsumed: sdk.Gas(len(`[{"foo":"bar"}]`)),
		},
		"allowed key - invalid msg": {
			filter: wasmtypes.NewAcceptedMessageKeysFilter("foo", "other"),
			src:    []byte(`not a json msg`),
			expErr: true,
		},
		"allow message - single": {
			filter: wasmtypes.NewAcceptedMessagesFilter([]byte(`{}`)),
			src:    []byte(`{}`),
			exp:    true,
		},
		"allow message - multiple": {
			filter: wasmtypes.NewAcceptedMessagesFilter([]byte(`[{"foo":"bar"}]`), []byte(`{"other":"value"}`)),
			src:    []byte(`[{"foo":"bar"}]`),
			exp:    true,
		},
		"allow message - no match": {
			filter: wasmtypes.NewAcceptedMessagesFilter([]byte(`{"foo":"bar"}`)),
			src:    []byte(`{"other":"value"}`),
			exp:    false,
		},
		"allow all message - always accept valid": {
			filter: wasmtypes.NewAllowAllMessagesFilter(),
			src:    []byte(`{"other":"value"}`),
			exp:    true,
		},
		"allow all message - always reject invalid json": {
			filter: wasmtypes.NewAllowAllMessagesFilter(),
			src:    []byte(`not json`),
			expErr: true,
		},
		"undefined - always errors": {
			filter: &wasmtypes.UndefinedFilter{},
			src:    []byte(`{"foo":"bar"}`),
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gm := sdk.NewGasMeter(1_000_000)
			allowed, gotErr := spec.filter.Accept(sdk.Context{}.WithGasMeter(gm), spec.src)

			// then
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.exp, allowed)
			assert.Equal(t, spec.expGasConsumed, gm.GasConsumed())
		})
	}
}

func TestContractAuthzLimitValidate(t *testing.T) {
	oneToken := sdk.NewCoin(sdk.DefaultBondDenom, sdk.OneInt())
	specs := map[string]struct {
		src    wasmtypes.ContractAuthzLimitX
		expErr bool
	}{
		"max calls": {
			src: wasmtypes.NewMaxCallsLimit(1),
		},
		"max calls - max uint64": {
			src: wasmtypes.NewMaxCallsLimit(math.MaxUint64),
		},
		"max calls - empty": {
			src:    wasmtypes.NewMaxCallsLimit(0),
			expErr: true,
		},
		"max funds": {
			src: wasmtypes.NewMaxFundsLimit(oneToken),
		},
		"max funds - empty coins": {
			src:    wasmtypes.NewMaxFundsLimit(),
			expErr: true,
		},
		"max funds - duplicates": {
			src:    &wasmtypes.MaxFundsLimit{Amounts: sdk.Coins{oneToken, oneToken}},
			expErr: true,
		},
		"max funds - contains empty value": {
			src:    &wasmtypes.MaxFundsLimit{Amounts: sdk.Coins{oneToken, sdk.NewCoin("other", sdk.ZeroInt())}.Sort()},
			expErr: true,
		},
		"max funds - unsorted": {
			src:    &wasmtypes.MaxFundsLimit{Amounts: sdk.Coins{oneToken, sdk.NewCoin("other", sdk.OneInt())}},
			expErr: true,
		},
		"combined": {
			src: wasmtypes.NewCombinedLimit(1, oneToken),
		},
		"combined - empty calls": {
			src:    wasmtypes.NewCombinedLimit(0, oneToken),
			expErr: true,
		},
		"combined - empty amounts": {
			src:    wasmtypes.NewCombinedLimit(1),
			expErr: true,
		},
		"combined - invalid amounts": {
			src:    &wasmtypes.CombinedLimit{CallsRemaining: 1, Amounts: sdk.Coins{oneToken, oneToken}},
			expErr: true,
		},
		"undefined": {
			src:    &wasmtypes.UndefinedLimit{},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gotErr := spec.src.ValidateBasic()
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
		})
	}
}

func TestContractAuthzLimitAccept(t *testing.T) {
	oneToken := sdk.NewCoin(sdk.DefaultBondDenom, sdk.OneInt())
	otherToken := sdk.NewCoin("other", sdk.OneInt())
	specs := map[string]struct {
		limit  wasmtypes.ContractAuthzLimitX
		src    wasmtypes.AuthzableWasmMsg
		exp    *wasmtypes.ContractAuthzLimitAcceptResult
		expErr bool
	}{
		"max calls - updated": {
			limit: wasmtypes.NewMaxCallsLimit(2),
			src:   &wasmtypes.MsgExecuteContract{},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: true, UpdateLimit: wasmtypes.NewMaxCallsLimit(1)},
		},
		"max calls - removed": {
			limit: wasmtypes.NewMaxCallsLimit(1),
			src:   &wasmtypes.MsgExecuteContract{},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: true, DeleteLimit: true},
		},
		"max calls - accepted with zero fund set": {
			limit: wasmtypes.NewMaxCallsLimit(1),
			src:   &wasmtypes.MsgExecuteContract{Funds: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.ZeroInt()))},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: true, DeleteLimit: true},
		},
		"max calls - rejected with some fund transfer": {
			limit: wasmtypes.NewMaxCallsLimit(1),
			src:   &wasmtypes.MsgExecuteContract{Funds: sdk.NewCoins(oneToken)},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: false},
		},
		"max calls - invalid": {
			limit:  &wasmtypes.MaxCallsLimit{},
			src:    &wasmtypes.MsgExecuteContract{},
			expErr: true,
		},
		"max funds - single updated": {
			limit: wasmtypes.NewMaxFundsLimit(oneToken.Add(oneToken)),
			src:   &wasmtypes.MsgExecuteContract{Funds: sdk.NewCoins(oneToken)},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: true, UpdateLimit: wasmtypes.NewMaxFundsLimit(oneToken)},
		},
		"max funds - single removed": {
			limit: wasmtypes.NewMaxFundsLimit(oneToken),
			src:   &wasmtypes.MsgExecuteContract{Funds: sdk.NewCoins(oneToken)},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: true, DeleteLimit: true},
		},
		"max funds - single with unknown token": {
			limit: wasmtypes.NewMaxFundsLimit(oneToken),
			src:   &wasmtypes.MsgExecuteContract{Funds: sdk.NewCoins(otherToken)},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: false},
		},
		"max funds - single exceeds limit": {
			limit: wasmtypes.NewMaxFundsLimit(oneToken),
			src:   &wasmtypes.MsgExecuteContract{Funds: sdk.NewCoins(oneToken.Add(oneToken))},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: false},
		},
		"max funds - single with additional token send": {
			limit: wasmtypes.NewMaxFundsLimit(oneToken),
			src:   &wasmtypes.MsgExecuteContract{Funds: sdk.NewCoins(oneToken, otherToken)},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: false},
		},
		"max funds - multi with other left": {
			limit: wasmtypes.NewMaxFundsLimit(oneToken, otherToken),
			src:   &wasmtypes.MsgExecuteContract{Funds: sdk.NewCoins(oneToken)},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: true, UpdateLimit: wasmtypes.NewMaxFundsLimit(otherToken)},
		},
		"max funds - multi with all used": {
			limit: wasmtypes.NewMaxFundsLimit(oneToken, otherToken),
			src:   &wasmtypes.MsgExecuteContract{Funds: sdk.NewCoins(oneToken, otherToken)},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: true, DeleteLimit: true},
		},
		"max funds - multi with no tokens sent": {
			limit: wasmtypes.NewMaxFundsLimit(oneToken, otherToken),
			src:   &wasmtypes.MsgExecuteContract{},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: true},
		},
		"max funds - multi with other exceeds limit": {
			limit: wasmtypes.NewMaxFundsLimit(oneToken, otherToken),
			src:   &wasmtypes.MsgExecuteContract{Funds: sdk.NewCoins(oneToken, otherToken.Add(otherToken))},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: false},
		},
		"max combined - multi amounts one consumed": {
			limit: wasmtypes.NewCombinedLimit(2, oneToken, otherToken),
			src:   &wasmtypes.MsgExecuteContract{Funds: sdk.NewCoins(oneToken)},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: true, UpdateLimit: wasmtypes.NewCombinedLimit(1, otherToken)},
		},
		"max combined - multi amounts none consumed": {
			limit: wasmtypes.NewCombinedLimit(2, oneToken, otherToken),
			src:   &wasmtypes.MsgExecuteContract{},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: true, UpdateLimit: wasmtypes.NewCombinedLimit(1, oneToken, otherToken)},
		},
		"max combined - removed on last execution": {
			limit: wasmtypes.NewCombinedLimit(1, oneToken, otherToken),
			src:   &wasmtypes.MsgExecuteContract{Funds: sdk.NewCoins(oneToken)},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: true, DeleteLimit: true},
		},
		"max combined - removed on last token": {
			limit: wasmtypes.NewCombinedLimit(2, oneToken),
			src:   &wasmtypes.MsgExecuteContract{Funds: sdk.NewCoins(oneToken)},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: true, DeleteLimit: true},
		},
		"max combined - update with token and calls remaining": {
			limit: wasmtypes.NewCombinedLimit(2, oneToken, otherToken),
			src:   &wasmtypes.MsgExecuteContract{Funds: sdk.NewCoins(oneToken)},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: true, UpdateLimit: wasmtypes.NewCombinedLimit(1, otherToken)},
		},
		"max combined - multi with other exceeds limit": {
			limit: wasmtypes.NewCombinedLimit(2, oneToken, otherToken),
			src:   &wasmtypes.MsgExecuteContract{Funds: sdk.NewCoins(oneToken, otherToken.Add(otherToken))},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: false},
		},
		"max combined - with unknown token": {
			limit: wasmtypes.NewCombinedLimit(2, oneToken),
			src:   &wasmtypes.MsgExecuteContract{Funds: sdk.NewCoins(otherToken)},
			exp:   &wasmtypes.ContractAuthzLimitAcceptResult{Accepted: false},
		},
		"undefined": {
			limit:  &wasmtypes.UndefinedLimit{},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gotResult, gotErr := spec.limit.Accept(sdk.Context{}, spec.src)
			// then
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.exp, gotResult)
		})
	}
}

func TestValidateCodeIdGrant(t *testing.T) {
	specs := map[string]struct {
		setup  func(t *testing.T) xiontypes.CodeIdGrant
		expErr bool
	}{
		"all good": {
			setup: func(t *testing.T) xiontypes.CodeIdGrant {
				return mustGrant(rand.Uint64(), wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAllowAllMessagesFilter())
			},
		},
		"invalid code id": {
			setup: func(t *testing.T) xiontypes.CodeIdGrant {
				return mustGrant(0, wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAllowAllMessagesFilter())
			},
			expErr: true,
		},
		"invalid limit": {
			setup: func(t *testing.T) xiontypes.CodeIdGrant {
				return mustGrant(rand.Uint64(), wasmtypes.NewMaxCallsLimit(0), wasmtypes.NewAllowAllMessagesFilter())
			},
			expErr: true,
		},

		"invalid filter ": {
			setup: func(t *testing.T) xiontypes.CodeIdGrant {
				return mustGrant(rand.Uint64(), wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAcceptedMessageKeysFilter())
			},
			expErr: true,
		},
		"empty limit": {
			setup: func(t *testing.T) xiontypes.CodeIdGrant {
				r := mustGrant(rand.Uint64(), wasmtypes.NewMaxCallsLimit(0), wasmtypes.NewAllowAllMessagesFilter())
				r.Limit = nil
				return r
			},
			expErr: true,
		},

		"empty filter ": {
			setup: func(t *testing.T) xiontypes.CodeIdGrant {
				r := mustGrant(rand.Uint64(), wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAcceptedMessageKeysFilter())
				r.Filter = nil
				return r
			},
			expErr: true,
		},
		"wrong limit type": {
			setup: func(t *testing.T) xiontypes.CodeIdGrant {
				r := mustGrant(rand.Uint64(), wasmtypes.NewMaxCallsLimit(0), wasmtypes.NewAllowAllMessagesFilter())
				r.Limit = r.Filter
				return r
			},
			expErr: true,
		},

		"wrong filter type": {
			setup: func(t *testing.T) xiontypes.CodeIdGrant {
				r := mustGrant(rand.Uint64(), wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAcceptedMessageKeysFilter())
				r.Filter = r.Limit
				return r
			},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gotErr := spec.setup(t).ValidateBasic()
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
		})
	}
}

func TestValidateCodeIdAuthorization(t *testing.T) {
	validGrant, err := xiontypes.NewCodeIdGrant(rand.Uint64(), wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAllowAllMessagesFilter())
	require.NoError(t, err)
	invalidGrant, err := xiontypes.NewCodeIdGrant(rand.Uint64(), wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAllowAllMessagesFilter())
	require.NoError(t, err)
	invalidGrant.Limit = nil

	specs := map[string]struct {
		setup  func(t *testing.T) validatable
		expErr bool
	}{
		"contract execution": {
			setup: func(t *testing.T) validatable {
				return xiontypes.NewCodeIdExecutionAuthorization(*validGrant)
			},
		},
		"contract execution - duplicate grants": {
			setup: func(t *testing.T) validatable {
				return xiontypes.NewCodeIdExecutionAuthorization(*validGrant, *validGrant)
			},
		},
		"contract execution - invalid grant": {
			setup: func(t *testing.T) validatable {
				return xiontypes.NewCodeIdExecutionAuthorization(*validGrant, *invalidGrant)
			},
			expErr: true,
		},
		"contract execution - empty grants": {
			setup: func(t *testing.T) validatable {
				return xiontypes.NewCodeIdExecutionAuthorization()
			},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gotErr := spec.setup(t).ValidateBasic()
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
		})
	}
}

func TestAcceptGrantedMessage(t *testing.T) {
	myContractAddr := sdk.AccAddress(randBytes(wasmtypes.SDKAddrLen))
	myCodeId := uint64(0)
	otherCodeId := uint64(1)
	ctx := sdk.Context{}.WithGasMeter(sdk.NewInfiniteGasMeter())
	store := populateStore(t, myContractAddr, myCodeId)
	ctx.WithMultiStore(store)
	specs := map[string]struct {
		auth      authztypes.Authorization
		msg       sdk.Msg
		expResult authztypes.AcceptResponse
		expErr    *errorsmod.Error
	}{
		"accepted and updated - contract execution": {
			auth: xiontypes.NewCodeIdExecutionAuthorization(mustGrant(myCodeId, wasmtypes.NewMaxCallsLimit(2), wasmtypes.NewAllowAllMessagesFilter())),
			msg: &wasmtypes.MsgExecuteContract{
				Sender:   sdk.AccAddress(randBytes(wasmtypes.SDKAddrLen)).String(),
				Contract: myContractAddr.String(),
				Msg:      []byte(`{"foo":"bar"}`),
			},
			expResult: authztypes.AcceptResponse{
				Accept:  true,
				Updated: xiontypes.NewCodeIdExecutionAuthorization(mustGrant(myCodeId, wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAllowAllMessagesFilter())),
			},
		},
		"accepted and not updated - limit not touched": {
			auth: xiontypes.NewCodeIdExecutionAuthorization(mustGrant(myCodeId, wasmtypes.NewMaxFundsLimit(sdk.NewCoin(sdk.DefaultBondDenom, sdk.OneInt())), wasmtypes.NewAllowAllMessagesFilter())),
			msg: &wasmtypes.MsgExecuteContract{
				Sender:   sdk.AccAddress(randBytes(wasmtypes.SDKAddrLen)).String(),
				Contract: myContractAddr.String(),
				Msg:      []byte(`{"foo":"bar"}`),
			},
			expResult: authztypes.AcceptResponse{Accept: true},
		},
		"accepted and removed - single": {
			auth: xiontypes.NewCodeIdExecutionAuthorization(mustGrant(myCodeId, wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAllowAllMessagesFilter())),
			msg: &wasmtypes.MsgExecuteContract{
				Sender:   sdk.AccAddress(randBytes(wasmtypes.SDKAddrLen)).String(),
				Contract: myContractAddr.String(),
				Msg:      []byte(`{"foo":"bar"}`),
			},
			expResult: authztypes.AcceptResponse{Accept: true, Delete: true},
		},
		"accepted and updated - multi, one removed": {
			auth: xiontypes.NewCodeIdExecutionAuthorization(
				mustGrant(myCodeId, wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAllowAllMessagesFilter()),
				mustGrant(myCodeId, wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAllowAllMessagesFilter()),
			),
			msg: &wasmtypes.MsgExecuteContract{
				Sender:   sdk.AccAddress(randBytes(wasmtypes.SDKAddrLen)).String(),
				Contract: myContractAddr.String(),
				Msg:      []byte(`{"foo":"bar"}`),
			},
			expResult: authztypes.AcceptResponse{
				Accept:  true,
				Updated: xiontypes.NewCodeIdExecutionAuthorization(mustGrant(myCodeId, wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAllowAllMessagesFilter())),
			},
		},
		"accepted and updated - multi, one updated": {
			auth: xiontypes.NewCodeIdExecutionAuthorization(
				mustGrant(otherCodeId, wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAllowAllMessagesFilter()),
				mustGrant(myCodeId, wasmtypes.NewMaxFundsLimit(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(2))), wasmtypes.NewAcceptedMessageKeysFilter("bar")),
				mustGrant(myCodeId, wasmtypes.NewCombinedLimit(2, sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(2))), wasmtypes.NewAcceptedMessageKeysFilter("foo")),
			),
			msg: &wasmtypes.MsgExecuteContract{
				Sender:   sdk.AccAddress(randBytes(wasmtypes.SDKAddrLen)).String(),
				Contract: myContractAddr.String(),
				Msg:      []byte(`{"foo":"bar"}`),
				Funds:    sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.OneInt())),
			},
			expResult: authztypes.AcceptResponse{
				Accept: true,
				Updated: xiontypes.NewCodeIdExecutionAuthorization(
					mustGrant(myCodeId, wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAllowAllMessagesFilter()),
					mustGrant(myCodeId, wasmtypes.NewMaxFundsLimit(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(2))), wasmtypes.NewAcceptedMessageKeysFilter("bar")),
					mustGrant(myCodeId, wasmtypes.NewCombinedLimit(1, sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(1))), wasmtypes.NewAcceptedMessageKeysFilter("foo")),
				),
			},
		},
		"not accepted - no matching contract address": {
			auth: xiontypes.NewCodeIdExecutionAuthorization(mustGrant(myCodeId, wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAllowAllMessagesFilter())),
			msg: &wasmtypes.MsgExecuteContract{
				Sender:   sdk.AccAddress(randBytes(wasmtypes.SDKAddrLen)).String(),
				Contract: sdk.AccAddress(randBytes(wasmtypes.SDKAddrLen)).String(),
				Msg:      []byte(`{"foo":"bar"}`),
			},
			expResult: authztypes.AcceptResponse{Accept: false},
		},
		"not accepted - max calls but tokens": {
			auth: xiontypes.NewCodeIdExecutionAuthorization(mustGrant(myCodeId, wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAllowAllMessagesFilter())),
			msg: &wasmtypes.MsgExecuteContract{
				Sender:   sdk.AccAddress(randBytes(wasmtypes.SDKAddrLen)).String(),
				Contract: myContractAddr.String(),
				Msg:      []byte(`{"foo":"bar"}`),
				Funds:    sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.OneInt())),
			},
			expResult: authztypes.AcceptResponse{Accept: false},
		},
		"not accepted - funds exceeds limit": {
			auth: xiontypes.NewCodeIdExecutionAuthorization(mustGrant(myCodeId, wasmtypes.NewMaxFundsLimit(sdk.NewCoin(sdk.DefaultBondDenom, sdk.OneInt())), wasmtypes.NewAllowAllMessagesFilter())),
			msg: &wasmtypes.MsgExecuteContract{
				Sender:   sdk.AccAddress(randBytes(wasmtypes.SDKAddrLen)).String(),
				Contract: myContractAddr.String(),
				Msg:      []byte(`{"foo":"bar"}`),
				Funds:    sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(2))),
			},
			expResult: authztypes.AcceptResponse{Accept: false},
		},
		"not accepted - no matching filter": {
			auth: xiontypes.NewCodeIdExecutionAuthorization(mustGrant(myCodeId, wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAcceptedMessageKeysFilter("other"))),
			msg: &wasmtypes.MsgExecuteContract{
				Sender:   sdk.AccAddress(randBytes(wasmtypes.SDKAddrLen)).String(),
				Contract: myContractAddr.String(),
				Msg:      []byte(`{"foo":"bar"}`),
				Funds:    sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.OneInt())),
			},
			expResult: authztypes.AcceptResponse{Accept: false},
		},
		"invalid msg type - contract execution": {
			auth: xiontypes.NewCodeIdExecutionAuthorization(mustGrant(myCodeId, wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAllowAllMessagesFilter())),
			msg: &wasmtypes.MsgMigrateContract{
				Sender:   sdk.AccAddress(randBytes(wasmtypes.SDKAddrLen)).String(),
				Contract: myContractAddr.String(),
				CodeID:   1,
				Msg:      []byte(`{"foo":"bar"}`),
			},
			expErr: sdkerrors.ErrInvalidType,
		},
		"payload is empty": {
			auth: xiontypes.NewCodeIdExecutionAuthorization(mustGrant(myCodeId, wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAllowAllMessagesFilter())),
			msg: &wasmtypes.MsgExecuteContract{
				Sender:   sdk.AccAddress(randBytes(wasmtypes.SDKAddrLen)).String(),
				Contract: myContractAddr.String(),
			},
			expErr: sdkerrors.ErrInvalidType,
		},
		"payload is invalid": {
			auth: xiontypes.NewCodeIdExecutionAuthorization(mustGrant(myCodeId, wasmtypes.NewMaxCallsLimit(1), wasmtypes.NewAllowAllMessagesFilter())),
			msg: &wasmtypes.MsgExecuteContract{
				Sender:   sdk.AccAddress(randBytes(wasmtypes.SDKAddrLen)).String(),
				Contract: myContractAddr.String(),
				Msg:      []byte(`not json`),
			},
			expErr: wasmtypes.ErrInvalid,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gotResult, gotErr := spec.auth.Accept(ctx, spec.msg)
			if spec.expErr != nil {
				require.ErrorIs(t, gotErr, spec.expErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.expResult, gotResult)
		})
	}
}

func mustGrant(codeId uint64, limit wasmtypes.ContractAuthzLimitX, filter wasmtypes.ContractAuthzFilterX) xiontypes.CodeIdGrant {
	g, err := xiontypes.NewCodeIdGrant(codeId, limit, filter)
	if err != nil {
		panic(err)
	}
	return *g
}

func populateStore(t *testing.T, contractAdress sdk.AccAddress, codeId uint64) *storetypes.Store {
	var db dbm.DB = dbm.NewMemDB()
	store := newMultiStoreWithMounts(db)
	err := store.LoadLatestVersion()
	require.Nil(t, err)

	contract := wasmtypes.ContractInfo{
		CodeID: codeId,
	}
	contractBz, err := contract.Marshal()
	require.NoError(t, err)
	require.NotNil(t, contractBz)
	// write some data in all stores
	k1, v1 := []byte(wasmtypes.GetContractAddressKey(contractAdress)), contractBz
	s1 := store.GetKVStore(wasmKey)
	require.NotNil(t, s1)
	s1.Set(k1, v1)
	return store
}

func newMultiStoreWithMounts(db dbm.DB) *storetypes.Store {
	store := storetypes.NewStore(db, log.NewNopLogger())

	store.MountStoreWithDB(wasmKey, types.StoreTypeIAVL, db)

	return store
}

func randBytes(n int) []byte {
	r := make([]byte, n)
	rand.Read(r) //nolint:staticcheck
	return r
}
