package types_test

import (
	"context"
	"testing"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/feegrant"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	xiontypes "github.com/burnt-labs/xion/x/xion/types"
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
				allowance, err = xiontypes.NewContractsAllowance(tc.allowance, tc.allowedContracts)
				require.NoError(t, err)
			} else {
				allowance = tc.allowance
			}
			authzAllowance, err := xiontypes.NewAuthzAllowance(allowance, tc.authzGrantee)
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
			allowance, err := xiontypes.NewMultiAnyAllowance(allowances)
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

func TestAuthzAllowance_GetAllowance_Error(t *testing.T) {
	// Create an invalid Any that will fail type assertion - use a string which doesn't implement FeeAllowanceI
	invalidAny, err := codectypes.NewAnyWithValue(&banktypes.MsgSend{}) // A type that doesn't implement FeeAllowanceI
	require.NoError(t, err)

	authzAllowance := &xiontypes.AuthzAllowance{
		Allowance: invalidAny,
	}

	// Should return error when type assertion fails
	allowance, err := authzAllowance.GetAllowance()
	require.Error(t, err)
	require.Nil(t, allowance)
	require.Contains(t, err.Error(), "failed to get allowance")
}

func TestContractsAllowance_GetAllowance_Error(t *testing.T) {
	// Create an invalid Any that will fail type assertion - use a type that doesn't implement FeeAllowanceI
	invalidAny, err := codectypes.NewAnyWithValue(&banktypes.MsgSend{}) // A type that doesn't implement FeeAllowanceI
	require.NoError(t, err)

	contractsAllowance := &xiontypes.ContractsAllowance{
		Allowance: invalidAny,
	}

	// Should return error when type assertion fails
	allowance, err := contractsAllowance.GetAllowance()
	require.Error(t, err)
	require.Nil(t, allowance)
	require.Contains(t, err.Error(), "failed to get allowance")
}

func TestMultiAnyAllowance_GetAllowance_Error(t *testing.T) {
	// Create an invalid Any that will fail type assertion - use a type that doesn't implement FeeAllowanceI
	invalidAny, err := codectypes.NewAnyWithValue(&banktypes.MsgSend{}) // A type that doesn't implement FeeAllowanceI
	require.NoError(t, err)

	multiAllowance := &xiontypes.MultiAnyAllowance{
		Allowances: []*codectypes.Any{invalidAny},
	}

	// Should return error when type assertion fails
	allowance, err := multiAllowance.GetAllowance(0)
	require.Error(t, err)
	require.Nil(t, allowance)
	require.Contains(t, err.Error(), "failed to get allowance")
}

// MockNonProtoAllowance is a mock that implements FeeAllowanceI but not proto.Message
type MockNonProtoAllowance struct{}

func (m *MockNonProtoAllowance) Accept(ctx context.Context, fee sdk.Coins, msgs []sdk.Msg) (bool, error) {
	return true, nil
}

func (m *MockNonProtoAllowance) ValidateBasic() error {
	return nil
}

func (m *MockNonProtoAllowance) ExpiresAt() (*time.Time, error) {
	return nil, nil
}

func TestNewAuthzAllowance_NonProtoMessage_Error(t *testing.T) {
	// Create an allowance that implements FeeAllowanceI but not proto.Message
	mockAllowance := &MockNonProtoAllowance{}

	// Should return error when allowance doesn't implement proto.Message
	authzGrantee, _ := sdk.AccAddressFromBech32("xion1qg5ega6dykkxc307y25pecuufrjkxkaggkkxh7")

	result, err := xiontypes.NewAuthzAllowance(mockAllowance, authzGrantee)
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "cannot proto marshal")
}

func TestEqTime(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)

	// Test case 1: Both times are non-nil and equal
	require.True(t, xiontypes.EqTime(&now, &now))

	// Test case 2: Both times are non-nil but different
	require.False(t, xiontypes.EqTime(&now, &future))

	// Test case 3: Both times are nil
	require.True(t, xiontypes.EqTime(nil, nil))

	// Test case 4: First is nil, second is not
	require.False(t, xiontypes.EqTime(nil, &now))

	// Test case 5: First is not nil, second is nil
	require.False(t, xiontypes.EqTime(&now, nil))
}

func TestNewContractsAllowance_NonProtoMessage_Error(t *testing.T) {
	// Create an allowance that implements FeeAllowanceI but not proto.Message
	mockAllowance := &MockNonProtoAllowance{}

	// Should return error when allowance doesn't implement proto.Message
	addresses := []sdk.AccAddress{
		sdk.AccAddress("address1"),
	}

	result, err := xiontypes.NewContractsAllowance(mockAllowance, addresses)
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "cannot proto marshal")
}

func TestNewMultiAnyAllowance_NonProtoMessage_Error(t *testing.T) {
	// Create an allowance that implements FeeAllowanceI but not proto.Message
	mockAllowance := &MockNonProtoAllowance{}

	// Should return error when allowance doesn't implement proto.Message
	allowances := []feegrant.FeeAllowanceI{mockAllowance}

	result, err := xiontypes.NewMultiAnyAllowance(allowances)
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "cannot proto marshal")
}

func TestAuthzAllowance_ValidateBasic(t *testing.T) {
	validAllowance := &feegrant.BasicAllowance{
		SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(100)}},
	}
	validGrantee := sdk.AccAddress("validgrantee123456789012345")

	tests := map[string]struct {
		allowance     *xiontypes.AuthzAllowance
		expectError   bool
		errorContains string
	}{
		"valid allowance": {
			allowance: func() *xiontypes.AuthzAllowance {
				authz, _ := xiontypes.NewAuthzAllowance(validAllowance, validGrantee)
				return authz
			}(),
			expectError: false,
		},
		"nil allowance": {
			allowance: &xiontypes.AuthzAllowance{
				Allowance:    nil,
				AuthzGrantee: validGrantee.String(),
			},
			expectError:   true,
			errorContains: "allowance should not be empty",
		},
		"invalid grantee address": {
			allowance: func() *xiontypes.AuthzAllowance {
				authz, _ := xiontypes.NewAuthzAllowance(validAllowance, validGrantee)
				authz.AuthzGrantee = "invalid-address"
				return authz
			}(),
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.allowance.ValidateBasic()
			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					require.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestContractsAllowance_ValidateBasic(t *testing.T) {
	validAllowance := &feegrant.BasicAllowance{
		SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(100)}},
	}
	validAddress := sdk.AccAddress("validcontract123456789012345")

	tests := map[string]struct {
		allowance     *xiontypes.ContractsAllowance
		expectError   bool
		errorContains string
	}{
		"valid allowance": {
			allowance: func() *xiontypes.ContractsAllowance {
				contracts, _ := xiontypes.NewContractsAllowance(validAllowance, []sdk.AccAddress{validAddress})
				return contracts
			}(),
			expectError: false,
		},
		"nil allowance": {
			allowance: &xiontypes.ContractsAllowance{
				Allowance:         nil,
				ContractAddresses: []string{validAddress.String()},
			},
			expectError:   true,
			errorContains: "allowance should not be empty",
		},
		"no contract addresses": {
			allowance: func() *xiontypes.ContractsAllowance {
				contracts, _ := xiontypes.NewContractsAllowance(validAllowance, []sdk.AccAddress{})
				return contracts
			}(),
			expectError:   true,
			errorContains: "must set contracts for feegrant",
		},
		"invalid contract address": {
			allowance: func() *xiontypes.ContractsAllowance {
				contracts, _ := xiontypes.NewContractsAllowance(validAllowance, []sdk.AccAddress{validAddress})
				contracts.ContractAddresses = []string{"invalid-address"}
				return contracts
			}(),
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.allowance.ValidateBasic()
			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					require.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMultiAnyAllowance_ValidateBasic(t *testing.T) {
	validAllowance := &feegrant.BasicAllowance{
		SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(100)}},
	}

	tests := map[string]struct {
		allowance     *xiontypes.MultiAnyAllowance
		expectError   bool
		errorContains string
	}{
		"valid allowance": {
			allowance: func() *xiontypes.MultiAnyAllowance {
				multi, _ := xiontypes.NewMultiAnyAllowance([]feegrant.FeeAllowanceI{validAllowance})
				return multi
			}(),
			expectError: false,
		},
		"empty allowances": {
			allowance: &xiontypes.MultiAnyAllowance{
				Allowances: []*codectypes.Any{},
			},
			expectError:   true,
			errorContains: "allowance list should contain at least one",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.allowance.ValidateBasic()
			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					require.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
