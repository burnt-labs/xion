package module

import (
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
)

// TestAppModule_IsOnePerModuleType tests the depinject marker interface.
// This method is intentionally empty as it's a marker for the depinject framework.
// Note: Coverage will show 0% because the method body is empty (no statements to cover).
func TestAppModule_IsOnePerModuleType(t *testing.T) {
	am := &AppModule{}
	// Call the method to ensure it can be invoked without panic
	am.IsOnePerModuleType()
	// Verify the method exists and is callable
	require.NotNil(t, am)
}

// TestAppModule_IsAppModule tests the appmodule marker interface.
// This method is intentionally empty as it's a marker for the appmodule framework.
// Note: Coverage will show 0% because the method body is empty (no statements to cover).
func TestAppModule_IsAppModule(t *testing.T) {
	am := &AppModule{}
	// Call the method to ensure it can be invoked without panic
	am.IsAppModule()
	// Verify the method exists and is callable
	require.NotNil(t, am)
}

// TestProvideModule tests the depinject provider function.
func TestProvideModule(t *testing.T) {
	// Create test dependencies
	encCfg := moduletestutil.MakeTestEncodingConfig()
	key := storetypes.NewKVStoreKey("dkim")
	storeService := runtime.NewKVStoreService(key)

	inputs := Inputs{
		Cdc:          encCfg.Codec,
		StoreService: storeService,
		AddressCodec: mockAddressCodec{},
	}

	// Call ProvideModule
	outputs := ProvideModule(inputs)

	// Verify outputs
	require.NotNil(t, outputs.Module)
	require.NotNil(t, outputs.Keeper)
}

// mockAddressCodec is a simple mock for testing
type mockAddressCodec struct{}

func (m mockAddressCodec) StringToBytes(text string) ([]byte, error) {
	return sdk.AccAddressFromBech32(text)
}

func (m mockAddressCodec) BytesToString(bz []byte) (string, error) {
	return sdk.AccAddress(bz).String(), nil
}
