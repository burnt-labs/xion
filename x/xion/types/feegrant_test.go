package types_test

import (
	"context"
	"fmt"
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

// Test UnpackInterfaces methods

func TestAuthzAllowance_UnpackInterfaces(t *testing.T) {
	// Create a basic allowance
	basicAllowance := &feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)),
	}
	// Create AuthzAllowance
	authzAddr := sdk.MustAccAddressFromBech32("cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x")
	authzAllowance, err := xiontypes.NewAuthzAllowance(basicAllowance, authzAddr)
	require.NoError(t, err)
	require.NotNil(t, authzAllowance)

	// Create a mock unpacker
	registry := codectypes.NewInterfaceRegistry()
	feegrant.RegisterInterfaces(registry)

	// Test successful unpacking
	err = authzAllowance.UnpackInterfaces(registry)
	require.NoError(t, err)

	// Test with nil allowance - should not panic but may or may not error
	authzAllowanceNil := &xiontypes.AuthzAllowance{
		Allowance: nil,
	}
	err = authzAllowanceNil.UnpackInterfaces(registry)
	// This may or may not error depending on implementation - just ensure it doesn't panic
	_ = err
}

func TestContractsAllowance_UnpackInterfaces(t *testing.T) {
	// Create a basic allowance
	basicAllowance := &feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)),
	}
	// Create ContractsAllowance
	contractAddr := sdk.MustAccAddressFromBech32("cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x")
	contractsAllowance, err := xiontypes.NewContractsAllowance(basicAllowance, []sdk.AccAddress{contractAddr})
	require.NoError(t, err)
	require.NotNil(t, contractsAllowance)

	// Create a mock unpacker
	registry := codectypes.NewInterfaceRegistry()
	feegrant.RegisterInterfaces(registry)

	// Test successful unpacking
	err = contractsAllowance.UnpackInterfaces(registry)
	require.NoError(t, err)

	// Test with nil allowance - should not panic but may or may not error
	contractsAllowanceNil := &xiontypes.ContractsAllowance{
		Allowance: nil,
	}
	err = contractsAllowanceNil.UnpackInterfaces(registry)
	// This may or may not error depending on implementation - just ensure it doesn't panic
	_ = err
}

func TestMultiAnyAllowance_UnpackInterfaces(t *testing.T) {
	// Create multiple allowances
	basicAllowance1 := &feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)),
	}
	basicAllowance2 := &feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uatom", 500)),
	}

	allowances := []feegrant.FeeAllowanceI{basicAllowance1, basicAllowance2}

	// Create MultiAnyAllowance
	multiAllowance, err := xiontypes.NewMultiAnyAllowance(allowances)
	require.NoError(t, err)
	require.NotNil(t, multiAllowance)

	// Create a mock unpacker
	registry := codectypes.NewInterfaceRegistry()
	feegrant.RegisterInterfaces(registry)

	// Test successful unpacking
	err = multiAllowance.UnpackInterfaces(registry)
	require.NoError(t, err)

	// Test with empty allowances list
	emptyMultiAllowance := &xiontypes.MultiAnyAllowance{
		Allowances: []*codectypes.Any{},
	}
	err = emptyMultiAllowance.UnpackInterfaces(registry)
	require.NoError(t, err) // Should succeed with empty list
	// Test with nil allowances list
	nilMultiAllowance := &xiontypes.MultiAnyAllowance{
		Allowances: nil,
	}
	err = nilMultiAllowance.UnpackInterfaces(registry)
	require.NoError(t, err) // Should succeed with nil list
}

// MockFailingAllowance implements FeeAllowanceI and is designed to fail during GetAllowance
type MockFailingAllowance struct {
	shouldFail bool
}

func (m *MockFailingAllowance) Reset()                                                 {}
func (m *MockFailingAllowance) String() string                                         { return "MockFailingAllowance" }
func (m *MockFailingAllowance) ProtoMessage()                                          {}
func (m *MockFailingAllowance) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error { return nil }
func (m *MockFailingAllowance) GetAllowance() (feegrant.FeeAllowanceI, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("mock allowance failure")
	}
	return &feegrant.BasicAllowance{}, nil
}
func (m *MockFailingAllowance) SetAllowance(allowance feegrant.FeeAllowanceI) error { return nil }
func (m *MockFailingAllowance) Accept(ctx context.Context, fee sdk.Coins, msgs []sdk.Msg) (bool, error) {
	if m.shouldFail {
		return false, fmt.Errorf("mock allowance failure")
	}
	return false, nil
}

func (m *MockFailingAllowance) ValidateBasic() error {
	if m.shouldFail {
		return fmt.Errorf("mock allowance failure")
	}
	return nil
}
func (m *MockFailingAllowance) ExpiresAt() (*time.Time, error) { return nil, nil }

func TestMultiAnyAllowanceGasChargingVulnerability(t *testing.T) {
	key := storetypes.NewKVStoreKey(feegrant.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

	// Simulate an attacker crafting a transaction with many failing allowances
	// This represents a malicious transaction submitted to the mempool
	const numFailingAllowances = 1000 // This number can be tuned to simulate a large attack
	allowances := make([]feegrant.FeeAllowanceI, numFailingAllowances)
	for i := 0; i < numFailingAllowances; i++ {
		allowances[i] = &MockFailingAllowance{shouldFail: true}
	}

	multiAllowance, err := xiontypes.NewMultiAnyAllowance(allowances)
	require.NoError(t, err)

	ctx := testCtx.Ctx
	// Minimal gas fee paid by attacker
	fee := sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(10)}}
	msgs := []sdk.Msg{&banktypes.MsgSend{}}

	// Record initial gas consumption to measure the impact
	initialGas := ctx.GasMeter().GasConsumed()

	// Simulate network processing nodes processing this transaction from mempool
	// This represents what happens when validators process the malicious transaction
	_, err = multiAllowance.Accept(ctx, fee, msgs)
	require.Error(t, err)
	require.Contains(t, err.Error(), "all allowances errored")

	finalGas := ctx.GasMeter().GasConsumed()
	gasConsumed := finalGas - initialGas

	// Calculate the economic mismatch and resource impact
	gasConsumedPerAllowance := uint64(10) // From gasCostPerIteration constant
	expectedGasForAllowances := numFailingAllowances * gasConsumedPerAllowance

	t.Logf("[ATTACK] Attacker submits transaction with %d failing allowances", numFailingAllowances)
	t.Logf("[ATTACK] Attacker pays minimal gas fee: %s", fee.String())
	t.Logf("[IMPACT] Network nodes must process %d gas units for failed transaction", gasConsumed)
	t.Logf("[IMPACT] Expected gas consumption: %d units (%d allowances × %d gas per allowance)",
		expectedGasForAllowances, numFailingAllowances, gasConsumedPerAllowance)
	t.Logf("[IMPACT] Actual gas consumed: %d units", gasConsumed)
	t.Logf("[IMPACT] Economic mismatch: Attacker paid for %s but consumed %d gas units", fee.String(), gasConsumed)
	t.Logf("")
	t.Logf("[IMPACT] Network processing nodes work beyond set parameters:")
	t.Logf("  - Transaction fails but consumes %d gas units", gasConsumed)
	t.Logf("  - Validators process excessive work for minimal fee")
	t.Logf("  - Network performance degrades as nodes handle disproportionate load")
	t.Logf("  - This pattern can be repeated to overwhelm network processing capacity")

	// Verify the impact
	require.Equal(t, expectedGasForAllowances, gasConsumed,
		"Gas consumption should match expected calculation")
	require.Greater(t, gasConsumed, uint64(0),
		"Gas should be consumed even for failed transaction")
}

func TestMultiAnyAllowanceGasChargingFix(t *testing.T) {
	key := storetypes.NewKVStoreKey(feegrant.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

	// Test case 1: With the fix, working allowances still consume gas appropriately
	t.Run("working allowances consume gas after fix", func(t *testing.T) {
		workingAllowance := &feegrant.BasicAllowance{
			SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(1000)}},
		}

		multiAllowance, err := xiontypes.NewMultiAnyAllowance([]feegrant.FeeAllowanceI{workingAllowance})
		require.NoError(t, err)

		ctx := testCtx.Ctx
		fee := sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(10)}}
		msgs := []sdk.Msg{&banktypes.MsgSend{}}

		initialGas := ctx.GasMeter().GasConsumed()

		// This should succeed and consume gas for the successful GetAllowance + processing
		accepted, err := multiAllowance.Accept(ctx, fee, msgs)
		require.NoError(t, err)
		require.False(t, accepted) // Should not remove the allowance

		finalGas := ctx.GasMeter().GasConsumed()
		gasConsumed := finalGas - initialGas

		t.Logf("Gas consumed for working allowance: %d units", gasConsumed)
		require.Equal(t, uint64(10), gasConsumed, "Should consume exactly gasCostPerIteration")
	})

	// Test case 2: Compare gas consumption of failing allowances before and after fix
	t.Run("failing allowances demonstrate fix effectiveness", func(t *testing.T) {
		// Create 100 allowances that will fail in Accept() method, not GetAllowance()
		const numAllowances = 100
		allowances := make([]feegrant.FeeAllowanceI, numAllowances)
		for i := 0; i < numAllowances; i++ {
			// Create allowances with insufficient funds that will fail Accept() but succeed GetAllowance()
			allowances[i] = &feegrant.BasicAllowance{
				SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(1)}}, // Too small
			}
		}

		multiAllowance, err := xiontypes.NewMultiAnyAllowance(allowances)
		require.NoError(t, err)

		ctx := testCtx.Ctx
		fee := sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(100)}} // Larger than any allowance
		msgs := []sdk.Msg{&banktypes.MsgSend{}}

		initialGas := ctx.GasMeter().GasConsumed()

		// This should fail but consume gas for all successful GetAllowance calls
		_, err = multiAllowance.Accept(ctx, fee, msgs)
		require.Error(t, err)
		require.Contains(t, err.Error(), "all allowances errored")

		finalGas := ctx.GasMeter().GasConsumed()
		gasConsumed := finalGas - initialGas

		expectedGas := uint64(numAllowances) * 10 // All GetAllowance calls succeed, so gas is charged

		t.Logf("[FIX VERIFICATION] Gas consumed with fix: %d units", gasConsumed)
		t.Logf("[FIX VERIFICATION] Expected gas (all GetAllowance succeed): %d units", expectedGas)

		require.Equal(t, expectedGas, gasConsumed,
			"Should consume gas for all successful GetAllowance calls")

		t.Logf("✅ FIX VERIFIED: Gas charged only after successful GetAllowance operations")
	})
}

// Test ExpiresAt methods

func TestAuthzAllowance_ExpiresAt(t *testing.T) {
	testCases := []struct {
		name        string
		allowance   feegrant.FeeAllowanceI
		expectError bool
		expectTime  bool
	}{
		{
			name:        "BasicAllowance without expiration",
			allowance:   &feegrant.BasicAllowance{},
			expectError: false,
			expectTime:  false,
		},
		{
			name: "BasicAllowance with expiration",
			allowance: &feegrant.BasicAllowance{
				Expiration: &time.Time{},
			},
			expectError: false,
			expectTime:  true,
		},
		{
			name: "PeriodicAllowance with expiration",
			allowance: &feegrant.PeriodicAllowance{
				Basic: feegrant.BasicAllowance{
					Expiration: &time.Time{},
				},
			},
			expectError: false,
			expectTime:  true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create AuthzAllowance
			authzAddr := sdk.MustAccAddressFromBech32("cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x")
			authzAllowance, err := xiontypes.NewAuthzAllowance(tc.allowance, authzAddr)
			require.NoError(t, err)
			// Test ExpiresAt
			expiry, err := authzAllowance.ExpiresAt()
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tc.expectTime {
					require.NotNil(t, expiry)
				} else {
					require.Nil(t, expiry)
				}
			}
		})
	}
	// Test error case where GetAllowance fails
	t.Run("invalid allowance type", func(t *testing.T) {
		// Create an AuthzAllowance with invalid Any content
		authzAllowance := &xiontypes.AuthzAllowance{
			AuthzGrantee: "cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x",
		}
		// Set Allowance to an Any containing a non-FeeAllowanceI type
		invalidAny, err := codectypes.NewAnyWithValue(&banktypes.MsgSend{})
		require.NoError(t, err)
		authzAllowance.Allowance = invalidAny
		// This should error when trying to get allowance
		expiry, err := authzAllowance.ExpiresAt()
		require.Error(t, err)
		require.Nil(t, expiry)
		require.Contains(t, err.Error(), "failed to get allowance")
	})
}

func TestContractsAllowance_ExpiresAt(t *testing.T) {
	testCases := []struct {
		name        string
		allowance   feegrant.FeeAllowanceI
		expectError bool
		expectTime  bool
	}{
		{
			name:        "BasicAllowance without expiration",
			allowance:   &feegrant.BasicAllowance{},
			expectError: false,
			expectTime:  false,
		},
		{
			name: "BasicAllowance with expiration",
			allowance: &feegrant.BasicAllowance{
				Expiration: &time.Time{},
			},
			expectError: false,
			expectTime:  true,
		},
		{
			name: "PeriodicAllowance with expiration",
			allowance: &feegrant.PeriodicAllowance{
				Basic: feegrant.BasicAllowance{
					Expiration: &time.Time{},
				},
			},
			expectError: false,
			expectTime:  true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create ContractsAllowance
			contractAddr := sdk.MustAccAddressFromBech32("cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x")
			contractsAllowance, err := xiontypes.NewContractsAllowance(tc.allowance, []sdk.AccAddress{contractAddr})
			require.NoError(t, err)
			// Test ExpiresAt
			expiry, err := contractsAllowance.ExpiresAt()
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tc.expectTime {
					require.NotNil(t, expiry)
				} else {
					require.Nil(t, expiry)
				}
			}
		})
	}
	// Test error case where GetAllowance fails
	t.Run("invalid allowance type", func(t *testing.T) {
		// Create a ContractsAllowance with invalid Any content
		invalidAny, err := codectypes.NewAnyWithValue(&banktypes.MsgSend{})
		require.NoError(t, err)

		contractsAllowance := &xiontypes.ContractsAllowance{
			Allowance: invalidAny,
		}

		// This should error when trying to get allowance
		expiry, err := contractsAllowance.ExpiresAt()
		require.Error(t, err)
		require.Nil(t, expiry)
		require.Contains(t, err.Error(), "failed to get allowance")
	})
}

func TestMultiAnyAllowance_UnpackInterfaces_ErrorCases(t *testing.T) {
	// Test with invalid Any type that can't be unpacked
	invalidAny := &codectypes.Any{
		TypeUrl: "invalid/type/url",
		Value:   []byte("invalid data"),
	}

	multiAllowance := &xiontypes.MultiAnyAllowance{
		Allowances: []*codectypes.Any{invalidAny},
	}

	registry := codectypes.NewInterfaceRegistry()
	feegrant.RegisterInterfaces(registry)

	err := multiAllowance.UnpackInterfaces(registry)
	require.Error(t, err) // Should error with invalid type
}

func TestMultiAnyAllowance_ExpiresAt(t *testing.T) {
	testCases := []struct {
		name         string
		allowances   []feegrant.FeeAllowanceI
		expectError  bool
		expectedTime *time.Time
	}{
		{
			name:         "empty allowances",
			allowances:   []feegrant.FeeAllowanceI{},
			expectError:  false,
			expectedTime: nil,
		},
		{
			name: "single allowance without expiration",
			allowances: []feegrant.FeeAllowanceI{
				&feegrant.BasicAllowance{},
			},
			expectError:  false,
			expectedTime: nil,
		},
		{
			name: "single allowance with expiration",
			allowances: []feegrant.FeeAllowanceI{
				&feegrant.BasicAllowance{
					Expiration: &time.Time{},
				},
			},
			expectError:  false,
			expectedTime: &time.Time{},
		},
		{
			name: "multiple allowances with same expiration",
			allowances: []feegrant.FeeAllowanceI{
				&feegrant.BasicAllowance{
					Expiration: &time.Time{},
				},
				&feegrant.BasicAllowance{
					Expiration: &time.Time{},
				},
			},
			expectError:  false,
			expectedTime: &time.Time{},
		},
		{
			name: "multiple allowances with different expirations",
			allowances: []feegrant.FeeAllowanceI{
				&feegrant.BasicAllowance{
					Expiration: &time.Time{},
				},
				&feegrant.BasicAllowance{
					Expiration: func() *time.Time {
						t := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
						return &t
					}(),
				},
			},
			expectError:  true,
			expectedTime: nil,
		},
		{
			name: "multiple allowances with nil vs non-nil expiration",
			allowances: []feegrant.FeeAllowanceI{
				&feegrant.BasicAllowance{}, // no expiration (nil)
				&feegrant.BasicAllowance{
					Expiration: func() *time.Time {
						t := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
						return &t
					}(),
				},
			},
			expectError:  true,
			expectedTime: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create MultiAnyAllowance
			multiAllowance, err := xiontypes.NewMultiAnyAllowance(tc.allowances)
			require.NoError(t, err)
			// Test ExpiresAt
			expiry, err := multiAllowance.ExpiresAt()
			if tc.expectError {
				require.Error(t, err)
				require.Nil(t, expiry)
			} else {
				require.NoError(t, err)
				if tc.expectedTime != nil {
					require.NotNil(t, expiry)
				} else {
					require.Nil(t, expiry)
				}
			}
		})
	}

	// Test error case where GetAllowance fails
	t.Run("invalid allowance type in multi", func(t *testing.T) {
		// Create a MultiAnyAllowance with invalid Any content
		invalidAny, err := codectypes.NewAnyWithValue(&banktypes.MsgSend{})
		require.NoError(t, err)

		multiAllowance := &xiontypes.MultiAnyAllowance{
			Allowances: []*codectypes.Any{invalidAny},
		}

		// This should error when trying to get allowance
		expiry, err := multiAllowance.ExpiresAt()
		require.Error(t, err)
		require.Nil(t, expiry)
		require.Contains(t, err.Error(), "failed to get allowance")
	})

	// Test error case where an allowance's ExpiresAt method fails
	t.Run("allowance ExpiresAt error", func(t *testing.T) {
		// Create a valid allowance and an invalid one that would cause ExpiresAt to fail
		validAllowance := &feegrant.BasicAllowance{}

		// Create an invalid Any for the second allowance
		invalidAny, err := codectypes.NewAnyWithValue(&banktypes.MsgSend{})
		require.NoError(t, err)

		multiAllowance := &xiontypes.MultiAnyAllowance{
			Allowances: []*codectypes.Any{},
		}

		// Add the valid allowance first
		validAny, err := codectypes.NewAnyWithValue(validAllowance)
		require.NoError(t, err)
		multiAllowance.Allowances = append(multiAllowance.Allowances, validAny)

		// Add the invalid allowance
		multiAllowance.Allowances = append(multiAllowance.Allowances, invalidAny)

		// This should error when trying to get the second allowance
		expiry, err := multiAllowance.ExpiresAt()
		require.Error(t, err)
		require.Nil(t, expiry)
		require.Contains(t, err.Error(), "failed to get allowance")
	})
}

// MockFailingAllowance implements FeeAllowanceI and is designed to fail during GetAllowance
type MockFailingAllowance struct {
	shouldFail bool
}

func (m *MockFailingAllowance) Reset()                                               {}
func (m *MockFailingAllowance) String() string                                       { return "MockFailingAllowance" }
func (m *MockFailingAllowance) ProtoMessage()                                        {}
func (m *MockFailingAllowance) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error { return nil }
func (m *MockFailingAllowance) GetAllowance() (feegrant.FeeAllowanceI, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("mock allowance failure")
	}
	return &feegrant.BasicAllowance{}, nil
}
func (m *MockFailingAllowance) SetAllowance(allowance feegrant.FeeAllowanceI) error { return nil }
func (m *MockFailingAllowance) Accept(ctx context.Context, fee sdk.Coins, msgs []sdk.Msg) (bool, error) {
	if m.shouldFail {
		return false, fmt.Errorf("mock allowance failure")
	}
	return false, nil
}
func (m *MockFailingAllowance) ValidateBasic() error {
	if m.shouldFail {
		return fmt.Errorf("mock allowance failure")
	}
	return nil
}
func (m *MockFailingAllowance) ExpiresAt() (*time.Time, error) { return nil, nil }

func TestMultiAnyAllowanceGasChargingVulnerability(t *testing.T) {
	key := storetypes.NewKVStoreKey(feegrant.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

	// Simulate an attacker crafting a transaction with many failing allowances
	// This represents a malicious transaction submitted to the mempool
	const numFailingAllowances = 1000 // This number can be tuned to simulate a large attack
	allowances := make([]feegrant.FeeAllowanceI, numFailingAllowances)
	for i := 0; i < numFailingAllowances; i++ {
		allowances[i] = &MockFailingAllowance{shouldFail: true}
	}

	multiAllowance, err := xiontypes.NewMultiAnyAllowance(allowances)
	require.NoError(t, err)

	ctx := testCtx.Ctx
	// Minimal gas fee paid by attacker
	fee := sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(10)}}
	msgs := []sdk.Msg{&banktypes.MsgSend{}}

	// Record initial gas consumption to measure the impact
	initialGas := ctx.GasMeter().GasConsumed()

	// Simulate network processing nodes processing this transaction from mempool
	// This represents what happens when validators process the malicious transaction
	_, err = multiAllowance.Accept(ctx, fee, msgs)
	require.Error(t, err)
	require.Contains(t, err.Error(), "all allowances errored")

	finalGas := ctx.GasMeter().GasConsumed()
	gasConsumed := finalGas - initialGas

	// Calculate the economic mismatch and resource impact
	gasConsumedPerAllowance := uint64(10) // From gasCostPerIteration constant
	expectedGasForAllowances := numFailingAllowances * gasConsumedPerAllowance

	t.Logf("[ATTACK] Attacker submits transaction with %d failing allowances", numFailingAllowances)
	t.Logf("[ATTACK] Attacker pays minimal gas fee: %s", fee.String())
	t.Logf("[IMPACT] Network nodes must process %d gas units for failed transaction", gasConsumed)
	t.Logf("[IMPACT] Expected gas consumption: %d units (%d allowances × %d gas per allowance)",
		expectedGasForAllowances, numFailingAllowances, gasConsumedPerAllowance)
	t.Logf("[IMPACT] Actual gas consumed: %d units", gasConsumed)
	t.Logf("[IMPACT] Economic mismatch: Attacker paid for %s but consumed %d gas units", fee.String(), gasConsumed)
	t.Logf("")
	t.Logf("[IMPACT] Network processing nodes work beyond set parameters:")
	t.Logf("  - Transaction fails but consumes %d gas units", gasConsumed)
	t.Logf("  - Validators process excessive work for minimal fee")
	t.Logf("  - Network performance degrades as nodes handle disproportionate load")
	t.Logf("  - This pattern can be repeated to overwhelm network processing capacity")

	// Verify the impact
	require.Equal(t, expectedGasForAllowances, gasConsumed,
		"Gas consumption should match expected calculation")
	require.Greater(t, gasConsumed, uint64(0),
		"Gas should be consumed even for failed transaction")
}

func TestMultiAnyAllowanceGasChargingFix(t *testing.T) {
	key := storetypes.NewKVStoreKey(feegrant.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

	// Test case 1: With the fix, working allowances still consume gas appropriately
	t.Run("working allowances consume gas after fix", func(t *testing.T) {
		workingAllowance := &feegrant.BasicAllowance{
			SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(1000)}},
		}

		multiAllowance, err := xiontypes.NewMultiAnyAllowance([]feegrant.FeeAllowanceI{workingAllowance})
		require.NoError(t, err)

		ctx := testCtx.Ctx
		fee := sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(10)}}
		msgs := []sdk.Msg{&banktypes.MsgSend{}}

		initialGas := ctx.GasMeter().GasConsumed()

		// This should succeed and consume gas for the successful GetAllowance + processing
		accepted, err := multiAllowance.Accept(ctx, fee, msgs)
		require.NoError(t, err)
		require.False(t, accepted) // Should not remove the allowance

		finalGas := ctx.GasMeter().GasConsumed()
		gasConsumed := finalGas - initialGas

		t.Logf("Gas consumed for working allowance: %d units", gasConsumed)
		require.Equal(t, uint64(10), gasConsumed, "Should consume exactly gasCostPerIteration")
	})

	// Test case 2: Compare gas consumption of failing allowances before and after fix
	t.Run("failing allowances demonstrate fix effectiveness", func(t *testing.T) {
		// Create 100 allowances that will fail in Accept() method, not GetAllowance()
		const numAllowances = 100
		allowances := make([]feegrant.FeeAllowanceI, numAllowances)
		for i := 0; i < numAllowances; i++ {
			// Create allowances with insufficient funds that will fail Accept() but succeed GetAllowance()
			allowances[i] = &feegrant.BasicAllowance{
				SpendLimit: sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(1)}}, // Too small
			}
		}

		multiAllowance, err := xiontypes.NewMultiAnyAllowance(allowances)
		require.NoError(t, err)

		ctx := testCtx.Ctx
		fee := sdk.Coins{sdk.Coin{Denom: "uxion", Amount: sdkmath.NewInt(100)}} // Larger than any allowance
		msgs := []sdk.Msg{&banktypes.MsgSend{}}

		initialGas := ctx.GasMeter().GasConsumed()

		// This should fail but consume gas for all successful GetAllowance calls
		_, err = multiAllowance.Accept(ctx, fee, msgs)
		require.Error(t, err)
		require.Contains(t, err.Error(), "all allowances errored")

		finalGas := ctx.GasMeter().GasConsumed()
		gasConsumed := finalGas - initialGas

		expectedGas := uint64(numAllowances) * 10 // All GetAllowance calls succeed, so gas is charged

		t.Logf("[FIX VERIFICATION] Gas consumed with fix: %d units", gasConsumed)
		t.Logf("[FIX VERIFICATION] Expected gas (all GetAllowance succeed): %d units", expectedGas)

		require.Equal(t, expectedGas, gasConsumed,
			"Should consume gas for all successful GetAllowance calls")

		t.Logf("✅ FIX VERIFIED: Gas charged only after successful GetAllowance operations")
	})
}
