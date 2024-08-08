package types_test

import (
	"testing"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/feegrant"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

func TestXionAllowanceValidAllow(t *testing.T) {
	key := storetypes.NewKVStoreKey(feegrant.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

	// msg we will call in the all cases
	sendMsg := banktypes.MsgSend{}

	cases := map[string]struct {
		allowance        *feegrant.BasicAllowance
		testGrantee      sdk.AccAddress
		authzGrantee     sdk.AccAddress
		contract         sdk.AccAddress
		allowedContracts []sdk.AccAddress
		fee              sdk.Coins
		blockTime        time.Time
		accept           bool
	}{
		"correct granter": {
			allowance:    &feegrant.BasicAllowance{},
			authzGrantee: sdk.MustAccAddressFromBech32("cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x"),
			testGrantee:  sdk.MustAccAddressFromBech32("cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x"),
			accept:       true,
		},
		"incorrect granter": {
			allowance:    &feegrant.BasicAllowance{},
			authzGrantee: sdk.MustAccAddressFromBech32("cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x"),
			testGrantee:  sdk.MustAccAddressFromBech32("cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr"),
			accept:       false,
		},
		"authz for valid contract": {
			allowance:        &feegrant.BasicAllowance{},
			authzGrantee:     sdk.MustAccAddressFromBech32("cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x"),
			testGrantee:      sdk.MustAccAddressFromBech32("cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x"),
			contract:         sdk.MustAccAddressFromBech32("cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr"),
			allowedContracts: []sdk.AccAddress{sdk.MustAccAddressFromBech32("cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr")},
			accept:           true,
		},
		"authz for invalid contract": {
			allowance:        &feegrant.BasicAllowance{},
			authzGrantee:     sdk.MustAccAddressFromBech32("cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x"),
			testGrantee:      sdk.MustAccAddressFromBech32("cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x"),
			contract:         sdk.MustAccAddressFromBech32("cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr"),
			allowedContracts: []sdk.AccAddress{sdk.MustAccAddressFromBech32("cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x")},
			accept:           false,
		},
	}

	for name, stc := range cases {
		tc := stc // to make scopelint happy
		t.Run(name, func(t *testing.T) {
			err := tc.allowance.ValidateBasic()
			require.NoError(t, err)

			ctx := testCtx.Ctx.WithBlockTime(tc.blockTime)

			// create grant
			var granter, grantee sdk.AccAddress
			var allowance feegrant.FeeAllowanceI
			if len(tc.allowedContracts) > 0 {
				allowance, err = types.NewContractsAllowance(tc.allowance, tc.allowedContracts)
				require.NoError(t, err)
			} else {
				allowance = tc.allowance
			}
			authzAllowance, err := types.NewAuthzAllowance(allowance, tc.authzGrantee)
			require.NoError(t, err)
			_, err = feegrant.NewGrant(granter, grantee, authzAllowance)
			require.NoError(t, err)

			// now try to deduct
			var msgs []sdk.Msg
			if tc.contract != nil {
				msgs = []sdk.Msg{&wasmtypes.MsgExecuteContract{Contract: tc.contract.String()}}
			} else {
				msgs = []sdk.Msg{&sendMsg}
			}
			authzExecMsg := authz.NewMsgExec(tc.testGrantee, msgs)
			_, err = authzAllowance.Accept(ctx, tc.fee, []sdk.Msg{&authzExecMsg})
			if !tc.accept {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestXionMultiAllowance(t *testing.T) {
	key := storetypes.NewKVStoreKey(feegrant.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

	// msg we will call in the all cases
	sendMsg := banktypes.MsgSend{}

	now := time.Now()
	inFive := time.Now().Add(time.Minute * 5)

	cases := map[string]struct {
		allowanceOne feegrant.FeeAllowanceI
		allowanceTwo feegrant.FeeAllowanceI
		fee          sdk.Coins
		validate     bool
		accept       bool
	}{
		"no allowances deny": {
			allowanceOne: nil,
			allowanceTwo: nil,
			fee:          sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(10)}},
			validate:     false,
			accept:       false,
		},
		"one allowance accept": {
			allowanceOne: &feegrant.BasicAllowance{SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(20)}}},
			allowanceTwo: nil,
			fee:          sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(10)}},
			validate:     true,
			accept:       true,
		},
		"two allowance accept": {
			allowanceOne: &feegrant.BasicAllowance{SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(20)}}},
			allowanceTwo: &feegrant.BasicAllowance{SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(20)}}},
			fee:          sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(10)}},
			validate:     true,
			accept:       true,
		},
		"one allowance deny": {
			allowanceOne: &feegrant.BasicAllowance{SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(20)}}},
			allowanceTwo: nil,
			fee:          sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(100)}},
			validate:     true,
			accept:       false,
		},
		"two allowance deny": {
			allowanceOne: &feegrant.BasicAllowance{SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(20)}}},
			allowanceTwo: &feegrant.BasicAllowance{SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(20)}}},
			fee:          sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(100)}},
			validate:     true,
			accept:       false,
		},
		"basic and periodic accept": {
			allowanceOne: &feegrant.PeriodicAllowance{
				Basic:            feegrant.BasicAllowance{SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(200)}}},
				Period:           86400,
				PeriodSpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(200)}},
				PeriodCanSpend:   sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(200)}},
				PeriodReset:      time.Time{},
			},
			allowanceTwo: &feegrant.BasicAllowance{SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(20)}}},
			fee:          sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(100)}},
			validate:     true,
			accept:       true,
		},
		"mismatched expiry deny": {
			allowanceTwo: &feegrant.PeriodicAllowance{
				Basic:            feegrant.BasicAllowance{SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(200)}}, Expiration: &inFive},
				Period:           86400,
				PeriodSpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(200)}},
				PeriodCanSpend:   sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(200)}},
				PeriodReset:      time.Time{},
			},
			allowanceOne: &feegrant.BasicAllowance{SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(20)}}, Expiration: &now},
			fee:          sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(100)}},
			validate:     false,
			accept:       true,
		},
	}

	for name, stc := range cases {
		tc := stc // to make scopelint happy
		t.Run(name, func(t *testing.T) {
			var allowances []feegrant.FeeAllowanceI
			if tc.allowanceOne != nil {
				allowances = append(allowances, tc.allowanceOne)
			}
			if tc.allowanceTwo != nil {
				allowances = append(allowances, tc.allowanceTwo)
			}
			allowance, err := types.NewMultiAnyAllowance(allowances)
			require.NoError(t, err)

			err = allowance.ValidateBasic()
			if tc.validate {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			ctx := testCtx.Ctx
			_, err = allowance.Accept(ctx, tc.fee, []sdk.Msg{&sendMsg})
			if tc.accept {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
