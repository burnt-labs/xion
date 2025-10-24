package keeper_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	module "github.com/burnt-labs/xion/x/zk"
	"github.com/burnt-labs/xion/x/zk/keeper"
	"github.com/burnt-labs/xion/x/zk/types"
)

type TestFixture struct {
	suite.Suite

	ctx         sdk.Context
	k           keeper.Keeper
	msgServer   types.MsgServer
	queryServer types.QueryServer
	appModule   *module.AppModule

	addrs      []sdk.AccAddress
	govModAddr string
}

func SetupTest(t *testing.T) *TestFixture {
	t.Helper()
	f := new(TestFixture)
	require := require.New(t)

	// Base setup
	logger := log.NewTestLogger(t)
	encCfg := moduletestutil.MakeTestEncodingConfig()

	f.govModAddr = authtypes.NewModuleAddress(govtypes.ModuleName).String()
	f.addrs = simtestutil.CreateIncrementalAccounts(3)

	key := storetypes.NewKVStoreKey(types.ModuleName)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

	f.ctx = testCtx.Ctx

	// Register SDK modules.
	registerBaseSDKModules(f, encCfg, storeService, logger, require)

	// Setup Keeper.
	f.k = keeper.NewKeeper(encCfg.Codec, storeService, logger, f.govModAddr)
	f.msgServer = keeper.NewMsgServerImpl(f.k)
	f.queryServer = keeper.NewQuerier(f.k)
	f.appModule = module.NewAppModule(encCfg.Codec, f.k)
	_, err := f.k.NextVKeyID.Next(f.ctx)
	require.NoError(err)
	return f
}

func registerModuleInterfaces(encCfg moduletestutil.TestEncodingConfig) {
	authtypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	types.RegisterInterfaces(encCfg.InterfaceRegistry)
}

func registerBaseSDKModules(
	_ *TestFixture,
	encCfg moduletestutil.TestEncodingConfig,
	_ store.KVStoreService,
	_ log.Logger,
	_ *require.Assertions,
) {
	registerModuleInterfaces(encCfg)
}

// ============================================================================
// Helper Functions
// ============================================================================

// createTestVKeyBytes creates test verification key bytes
func createTestVKeyBytes(name string) []byte {
	vkeyJSON := map[string]interface{}{
		"protocol": "groth16",
		"curve":    "bn128",
		"nPublic":  34,
		"vk_alpha_1": []string{
			"20491192805390485299153009773594534940189261866228447918068658471970481763042",
			"9383485363053290200918347156157836566562967994039712273449902621266178545958",
			"1",
		},
		"vk_beta_2": [][]string{
			{"6375614351688725206403948262868962793625744043794305715222011528459656738731", "4252822878758300859123897981450591353533073413197771768651442665752259397132"},
			{"10505242626370262277552901082094356697409835680220590971873171140371331206856", "21847035105528745403288232691147584728191162732299865338377159692350059136679"},
			{"1", "0"},
		},
		"vk_gamma_2": [][]string{
			{"10857046999023057135944570762232829481370756359578518086990519993285655852781", "11559732032986387107991004021392285783925812861821192530917403151452391805634"},
			{"8495653923123431417604973247489272438418190587263600148770280649306958101930", "4082367875863433681332203403145435568316851327593401208105741076214120093531"},
			{"1", "0"},
		},
		"vk_delta_2": [][]string{
			{"7408543996799841808823674318962923691422846694508104677211507255777183761346", "17378314708652486082434193052153411074104970941065581812653446685054220492752"},
			{"20934765493363178521480199624017210946632719146191129233788277268880988392769", "9933248257943163684434361179172132751107201169345727211797322171844177096469"},
			{"1", "0"},
		},
		"IC": [][]string{
			{"5449013234494434531196202102845211237542489505716355090765771488165044993949", "4910919431725277797191489997138444712176878647014509270723700672161925471159", "1"},
		},
	}

	bytes, _ := json.Marshal(vkeyJSON)
	return bytes
}

// loadVKeyFromJSON loads a vkey from a JSON file
func loadVKeyFromJSON(t *testing.T, filepath string) []byte {
	data, err := os.ReadFile(filepath)
	require.NoError(t, err)
	return data
}

// ============================================================================
// Basic Tests
// ============================================================================

func TestKeeperLogger(t *testing.T) {
	f := SetupTest(t)
	logger := f.k.Logger()
	require.NotNil(t, logger)
}

// ============================================================================
// VKey CRUD Tests
// ============================================================================

func TestAddVKey(t *testing.T) {
	f := SetupTest(t)

	tests := []struct {
		name        string
		authority   string
		vkeyName    string
		vkeyBytes   []byte
		description string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successfully add vkey",
			authority:   f.govModAddr,
			vkeyName:    "email_auth",
			vkeyBytes:   createTestVKeyBytes("email_auth"),
			description: "Email authentication circuit",
			expectError: false,
		},
		{
			name:        "successfully add second vkey",
			authority:   f.govModAddr,
			vkeyName:    "rollup_batch",
			vkeyBytes:   createTestVKeyBytes("rollup_batch"),
			description: "Rollup batch verification",
			expectError: false,
		},
		{
			name:        "fail to add duplicate vkey name",
			authority:   f.govModAddr,
			vkeyName:    "email_auth",
			vkeyBytes:   createTestVKeyBytes("email_auth"),
			description: "Duplicate",
			expectError: true,
			errorMsg:    "already exists",
		},
		{
			name:        "fail to add with incorrect authority",
			authority:   f.addrs[0].String(),
			vkeyName:    "unauthorized",
			vkeyBytes:   createTestVKeyBytes("unauthorized"),
			description: "Unauthorized",
			expectError: true,
			errorMsg:    "invalid authority",
		},
		{
			name:        "fail to add with empty vkey bytes",
			authority:   f.govModAddr,
			vkeyName:    "empty_vkey",
			vkeyBytes:   []byte{},
			description: "Empty vkey",
			expectError: true,
			errorMsg:    "invalid verification key",
		},
		{
			name:        "fail to add with invalid JSON",
			authority:   f.govModAddr,
			vkeyName:    "invalid_json",
			vkeyBytes:   []byte("not valid json"),
			description: "Invalid JSON",
			expectError: true,
			errorMsg:    "invalid verification key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := f.k.AddVKey(f.ctx, tt.authority, tt.vkeyName, tt.vkeyBytes, tt.description)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
				require.Equal(t, uint64(0), id)
			} else {
				require.NoError(t, err)
				require.GreaterOrEqual(t, id, uint64(0))

				// Verify the vkey was stored correctly
				storedVKey, err := f.k.GetVKeyByID(f.ctx, id)
				require.NoError(t, err)
				require.Equal(t, tt.vkeyName, storedVKey.Name)
				require.Equal(t, tt.description, storedVKey.Description)
				require.Equal(t, tt.vkeyBytes, storedVKey.KeyBytes)
			}
		})
	}
}

func TestGetVKeyByID(t *testing.T) {
	f := SetupTest(t)

	// Add test vkey
	vkeyBytes := createTestVKeyBytes("test_key")
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "test_key", vkeyBytes, "Test verification key")
	require.NoError(t, err)

	tests := []struct {
		name        string
		id          uint64
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successfully get existing vkey",
			id:          id,
			expectError: false,
		},
		{
			name:        "fail to get non-existent vkey",
			id:          9999,
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := f.k.GetVKeyByID(f.ctx, tt.id)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, "test_key", retrieved.Name)
				require.Equal(t, "Test verification key", retrieved.Description)
			}
		})
	}
}

func TestGetVKeyByName(t *testing.T) {
	f := SetupTest(t)

	// Add test vkey
	vkeyBytes := createTestVKeyBytes("email_auth")
	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth", vkeyBytes, "Email authentication")
	require.NoError(t, err)

	tests := []struct {
		name        string
		vkeyName    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successfully get vkey by name",
			vkeyName:    "email_auth",
			expectError: false,
		},
		{
			name:        "fail to get non-existent vkey",
			vkeyName:    "non_existent",
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := f.k.GetVKeyByName(f.ctx, tt.vkeyName)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.vkeyName, retrieved.Name)
				require.Equal(t, "Email authentication", retrieved.Description)
			}
		})
	}
}

func TestGetCircomVKeyByName(t *testing.T) {
	f := SetupTest(t)

	// Add test vkey
	vkeyBytes := createTestVKeyBytes("email_auth")
	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth", vkeyBytes, "Email authentication")
	require.NoError(t, err)

	// Get as CircomVerificationKey
	circomVKey, err := f.k.GetCircomVKeyByName(f.ctx, "email_auth")
	require.NoError(t, err)
	require.NotNil(t, circomVKey)
	require.Equal(t, "groth16", circomVKey.Protocol)
	require.Equal(t, "bn128", circomVKey.Curve)
	require.Equal(t, 34, circomVKey.NPublic)
}

func TestHasVKey(t *testing.T) {
	f := SetupTest(t)

	// Add test vkey
	vkeyBytes := createTestVKeyBytes("test_key")
	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "test_key", vkeyBytes, "Test key")
	require.NoError(t, err)

	tests := []struct {
		name     string
		vkeyName string
		expected bool
	}{
		{
			name:     "vkey exists",
			vkeyName: "test_key",
			expected: true,
		},
		{
			name:     "vkey does not exist",
			vkeyName: "non_existent",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			has, err := f.k.HasVKey(f.ctx, tt.vkeyName)
			require.NoError(t, err)
			require.Equal(t, tt.expected, has)
		})
	}
}

func TestUpdateVKey(t *testing.T) {
	f := SetupTest(t)

	// Add initial vkey
	vkeyBytes := createTestVKeyBytes("email_auth")
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth", vkeyBytes, "Original description")
	require.NoError(t, err)

	tests := []struct {
		name        string
		authority   string
		vkeyName    string
		newBytes    []byte
		description string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successfully update vkey",
			authority:   f.govModAddr,
			vkeyName:    "email_auth",
			newBytes:    createTestVKeyBytes("email_auth"),
			description: "Updated description",
			expectError: false,
		},
		{
			name:        "fail to update with incorrect authority",
			authority:   f.addrs[0].String(),
			vkeyName:    "email_auth",
			newBytes:    createTestVKeyBytes("email_auth"),
			description: "Unauthorized",
			expectError: true,
			errorMsg:    "invalid authority",
		},
		{
			name:        "fail to update non-existent vkey",
			authority:   f.govModAddr,
			vkeyName:    "non_existent",
			newBytes:    createTestVKeyBytes("non_existent"),
			description: "Does not exist",
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name:        "fail with empty vkey bytes",
			authority:   f.govModAddr,
			vkeyName:    "email_auth",
			newBytes:    []byte{},
			description: "Empty",
			expectError: true,
			errorMsg:    "invalid verification key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := f.k.UpdateVKey(f.ctx, tt.authority, tt.vkeyName, tt.newBytes, tt.description)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)

				// Verify the update
				updated, err := f.k.GetVKeyByID(f.ctx, id)
				require.NoError(t, err)
				require.Equal(t, tt.description, updated.Description)
			}
		})
	}
}

func TestRemoveVKey(t *testing.T) {
	f := SetupTest(t)

	// Add test vkeys
	vkey1Bytes := createTestVKeyBytes("key1")
	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "key1", vkey1Bytes, "Key 1")
	require.NoError(t, err)

	vkey2Bytes := createTestVKeyBytes("key2")
	_, err = f.k.AddVKey(f.ctx, f.govModAddr, "key2", vkey2Bytes, "Key 2")
	require.NoError(t, err)

	tests := []struct {
		name        string
		authority   string
		vkeyName    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "fail to remove with incorrect authority",
			authority:   f.addrs[0].String(),
			vkeyName:    "key1",
			expectError: true,
			errorMsg:    "invalid authority",
		},
		{
			name:        "successfully remove vkey",
			authority:   f.govModAddr,
			vkeyName:    "key1",
			expectError: false,
		},
		{
			name:        "fail to remove non-existent vkey",
			authority:   f.govModAddr,
			vkeyName:    "non_existent",
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name:        "fail to remove already removed vkey",
			authority:   f.govModAddr,
			vkeyName:    "key1",
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := f.k.RemoveVKey(f.ctx, tt.authority, tt.vkeyName)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)

				// Verify the vkey was removed
				has, err := f.k.HasVKey(f.ctx, tt.vkeyName)
				require.NoError(t, err)
				require.False(t, has)
			}
		})
	}
}

func TestListVKeys(t *testing.T) {
	f := SetupTest(t)

	// Test with empty store
	vkeys, err := f.k.ListVKeys(f.ctx)
	require.NoError(t, err)
	require.Empty(t, vkeys)

	// Add multiple vkeys
	for i := 0; i < 3; i++ {
		vkeyBytes := createTestVKeyBytes(fmt.Sprintf("key%d", i))
		_, err := f.k.AddVKey(f.ctx, f.govModAddr, fmt.Sprintf("key%d", i), vkeyBytes, fmt.Sprintf("Key %d", i))
		require.NoError(t, err)
	}

	// List all vkeys
	vkeys, err = f.k.ListVKeys(f.ctx)
	require.NoError(t, err)
	require.Len(t, vkeys, 3)

	// Verify names are present
	names := make(map[string]bool)
	for _, vkey := range vkeys {
		names[vkey.Name] = true
	}
	require.True(t, names["key0"])
	require.True(t, names["key1"])
	require.True(t, names["key2"])
}

// ============================================================================
// Sequence Tests
// ============================================================================

func TestSequenceIncrement(t *testing.T) {
	f := SetupTest(t)

	// Add multiple vkeys and verify IDs increment
	vkey1Bytes := createTestVKeyBytes("key1")
	id1, err := f.k.AddVKey(f.ctx, f.govModAddr, "key1", vkey1Bytes, "Key 1")
	require.NoError(t, err)
	require.Equal(t, uint64(1), id1)

	vkey2Bytes := createTestVKeyBytes("key2")
	id2, err := f.k.AddVKey(f.ctx, f.govModAddr, "key2", vkey2Bytes, "Key 2")
	require.NoError(t, err)
	require.Equal(t, uint64(2), id2)

	vkey3Bytes := createTestVKeyBytes("key3")
	id3, err := f.k.AddVKey(f.ctx, f.govModAddr, "key3", vkey3Bytes, "Key 3")
	require.NoError(t, err)
	require.Equal(t, uint64(3), id3)
}

func TestSequencePersistence(t *testing.T) {
	f := SetupTest(t)

	// Add first vkey
	vkey1Bytes := createTestVKeyBytes("key1")
	id1, err := f.k.AddVKey(f.ctx, f.govModAddr, "key1", vkey1Bytes, "Key 1")
	require.NoError(t, err)
	require.Equal(t, uint64(1), id1)

	// Add second vkey - sequence should increment
	vkey2Bytes := createTestVKeyBytes("key2")
	id2, err := f.k.AddVKey(f.ctx, f.govModAddr, "key2", vkey2Bytes, "Key 2")
	require.NoError(t, err)
	require.Equal(t, uint64(2), id2)

	// Verify both vkeys exist
	retrieved1, err := f.k.GetVKeyByID(f.ctx, id1)
	require.NoError(t, err)
	require.Equal(t, "key1", retrieved1.Name)

	retrieved2, err := f.k.GetVKeyByID(f.ctx, id2)
	require.NoError(t, err)
	require.Equal(t, "key2", retrieved2.Name)
}

// ============================================================================
// Index Tests
// ============================================================================

func TestNameIndexConsistency(t *testing.T) {
	f := SetupTest(t)

	// Add vkey
	vkeyBytes := createTestVKeyBytes("test_key")
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "test_key", vkeyBytes, "Test")
	require.NoError(t, err)

	// Verify both ID and name access return the same vkey
	vkeyByID, err := f.k.GetVKeyByID(f.ctx, id)
	require.NoError(t, err)

	vkeyByName, err := f.k.GetVKeyByName(f.ctx, "test_key")
	require.NoError(t, err)

	require.Equal(t, vkeyByID.Name, vkeyByName.Name)
	require.Equal(t, vkeyByID.Description, vkeyByName.Description)
	require.Equal(t, vkeyByID.KeyBytes, vkeyByName.KeyBytes)
}

func TestNameIndexAfterRemoval(t *testing.T) {
	f := SetupTest(t)

	// Add vkey
	vkeyBytes := createTestVKeyBytes("test_key")
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "test_key", vkeyBytes, "Test")
	require.NoError(t, err)

	// Remove vkey
	err = f.k.RemoveVKey(f.ctx, f.govModAddr, "test_key")
	require.NoError(t, err)

	// Verify both ID and name access fail
	_, err = f.k.GetVKeyByID(f.ctx, id)
	require.Error(t, err)

	_, err = f.k.GetVKeyByName(f.ctx, "test_key")
	require.Error(t, err)

	// Verify name is not in index
	has, err := f.k.HasVKey(f.ctx, "test_key")
	require.NoError(t, err)
	require.False(t, has)
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestEmptyName(t *testing.T) {
	f := SetupTest(t)

	vkeyBytes := createTestVKeyBytes("")
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "", vkeyBytes, "Empty name test")

	// Empty names should be allowed at keeper level
	// Validation should happen at message level
	require.NoError(t, err)
	require.GreaterOrEqual(t, id, uint64(0))

	retrieved, err := f.k.GetVKeyByName(f.ctx, "")
	require.NoError(t, err)
	require.Equal(t, "", retrieved.Name)
}

func TestVeryLongName(t *testing.T) {
	f := SetupTest(t)

	longName := string(make([]byte, 1000))
	for i := 0; i < 1000; i++ {
		longName = longName[:i] + "a" + longName[i+1:]
	}

	vkeyBytes := createTestVKeyBytes(longName)
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, longName, vkeyBytes, "Long name test")
	require.NoError(t, err)
	require.GreaterOrEqual(t, id, uint64(0))

	// Verify retrieval works
	retrieved, err := f.k.GetVKeyByName(f.ctx, longName)
	require.NoError(t, err)
	require.Equal(t, longName, retrieved.Name)
}

func TestConcurrentAccess(t *testing.T) {
	f := SetupTest(t)

	// Add multiple vkeys
	for i := 1; i < 11; i++ {
		vkeyBytes := createTestVKeyBytes(fmt.Sprintf("key%d", i))
		_, err := f.k.AddVKey(f.ctx, f.govModAddr, fmt.Sprintf("key%d", i), vkeyBytes, fmt.Sprintf("Key %d", i))
		require.NoError(t, err)
	}

	// Access all vkeys by both ID and name
	for i := 1; i < 11; i++ {
		vkeyByID, err := f.k.GetVKeyByID(f.ctx, uint64(i))
		require.NoError(t, err)

		vkeyByName, err := f.k.GetVKeyByName(f.ctx, fmt.Sprintf("key%d", i))
		require.NoError(t, err)

		require.Equal(t, vkeyByID.Name, vkeyByName.Name)
	}
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestFullVKeyLifecycle(t *testing.T) {
	f := SetupTest(t)

	// 1. Add vkey
	vkeyBytes := createTestVKeyBytes("lifecycle_key")
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "lifecycle_key", vkeyBytes, "Initial version")
	require.NoError(t, err)
	require.Equal(t, uint64(1), id)

	// 2. Verify it exists
	has, err := f.k.HasVKey(f.ctx, "lifecycle_key")
	require.NoError(t, err)
	require.True(t, has)

	// 3. Get by ID
	retrieved, err := f.k.GetVKeyByID(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, "Initial version", retrieved.Description)

	// 4. Get by name
	retrievedByName, err := f.k.GetVKeyByName(f.ctx, "lifecycle_key")
	require.NoError(t, err)
	require.Equal(t, retrieved.Name, retrievedByName.Name)

	// 5. Update
	updatedBytes := createTestVKeyBytes("lifecycle_key")
	err = f.k.UpdateVKey(f.ctx, f.govModAddr, "lifecycle_key", updatedBytes, "Updated version")
	require.NoError(t, err)

	// 6. Verify update
	retrieved, err = f.k.GetVKeyByID(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, "Updated version", retrieved.Description)

	// 7. List all keys
	vkeys, err := f.k.ListVKeys(f.ctx)
	require.NoError(t, err)
	require.Len(t, vkeys, 1)

	// 8. Remove
	err = f.k.RemoveVKey(f.ctx, f.govModAddr, "lifecycle_key")
	require.NoError(t, err)

	// 9. Verify removal
	has, err = f.k.HasVKey(f.ctx, "lifecycle_key")
	require.NoError(t, err)
	require.False(t, has)

	// 10. List should be empty
	vkeys, err = f.k.ListVKeys(f.ctx)
	require.NoError(t, err)
	require.Empty(t, vkeys)
}

// ============================================================================
// Circom Integration Tests
// ============================================================================

func TestCircomVKeyConversion(t *testing.T) {
	f := SetupTest(t)

	// Add vkey
	vkeyBytes := createTestVKeyBytes("circom_test")
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "circom_test", vkeyBytes, "Circom test")
	require.NoError(t, err)

	// Get as standard VKey
	vkey, err := f.k.GetVKeyByID(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, "circom_test", vkey.Name)

	// Get as CircomVerificationKey
	circomVKey, err := f.k.GetCircomVKeyByID(f.ctx, id)
	require.NoError(t, err)
	require.NotNil(t, circomVKey)
	require.Equal(t, "groth16", circomVKey.Protocol)
	require.Equal(t, "bn128", circomVKey.Curve)

	// Verify it can be used with the parser
	require.Equal(t, 34, circomVKey.NPublic)
	require.Len(t, circomVKey.VkAlpha1, 3)
	require.Len(t, circomVKey.IC, 1)
}

func TestLoadActualVKeyFile(t *testing.T) {
	// Skip if test file doesn't exist
	if _, err := os.Stat("testdata/email_auth_vkey.json"); os.IsNotExist(err) {
		t.Skip("testdata/email_auth_vkey.json not found")
	}

	f := SetupTest(t)

	// Load actual vkey from file
	vkeyBytes := loadVKeyFromJSON(t, "testdata/email_auth_vkey.json")

	// Add to keeper
	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth", vkeyBytes, "Email authentication from file")
	require.NoError(t, err)

	// Retrieve and verify
	circomVKey, err := f.k.GetCircomVKeyByName(f.ctx, "email_auth")
	require.NoError(t, err)
	require.NotNil(t, circomVKey)
	require.Equal(t, "groth16", circomVKey.Protocol)
	require.Equal(t, "bn128", circomVKey.Curve)
	require.Equal(t, 34, circomVKey.NPublic)
	require.Len(t, circomVKey.IC, 35) // 34 public inputs + 1
}

// ============================================================================
// Validation Tests
// ============================================================================

func TestValidateVKeyBytes(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid vkey",
			data:        createTestVKeyBytes("test"),
			expectError: false,
		},
		{
			name:        "empty data",
			data:        []byte{},
			expectError: true,
			errorMsg:    "empty vkey data",
		},
		{
			name:        "invalid json",
			data:        []byte("not json"),
			expectError: true,
			errorMsg:    "invalid verification key JSON",
		},
		{
			name: "missing required fields",
			data: []byte(`{
				"protocol": "groth16",
				"curve": "bn128"
			}`),
			expectError: true,
			errorMsg:    "invalid verification key JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.ValidateVKeyBytes(tt.data)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
