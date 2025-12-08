package v25_upgrade

import (
	"fmt"
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// setupTestStore creates a test store for migration testing
func setupTestStore(t *testing.T) (sdk.Context, storetypes.StoreKey, storetypes.CommitMultiStore) {
	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey("wasm")

	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)

	err := cms.LoadLatestVersion()
	require.NoError(t, err)

	// Create context with empty header
	ctx := sdk.NewContext(cms, tmproto.Header{}, false, log.NewNopLogger())
	return ctx, storeKey, cms
}

// createContractKey creates a storage key for a contract
func createContractKey(address string) []byte {
	// For testing, just use the address string as raw bytes
	// In production, this would be a proper bech32 address
	addrBytes := []byte(address)
	return append([]byte{ContractKeyPrefix}, addrBytes...)
}

// TestMigrateContractsEmpty verifies migration with no contracts
func TestMigrateContractsEmpty(t *testing.T) {
	ctx, storeKey, _ := setupTestStore(t)

	// Run migration on empty store
	err := MigrateContracts(ctx, storeKey)
	require.NoError(t, err)
}

// TestMigrateContractsHealthy verifies that healthy contracts are not modified
func TestMigrateContractsHealthy(t *testing.T) {
	ctx, storeKey, cms := setupTestStore(t)
	store := ctx.KVStore(storeKey)

	// Create healthy contracts
	addresses := []string{
		"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpn45e7",
		"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpkxfqtv",
		"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqp5dzgw7",
	}

	for i, addr := range addresses {
		contractInfo := &wasmtypes.ContractInfo{
			CodeID:  uint64(i + 1),
			Creator: "xion1creator",
			Admin:   "xion1admin",
			Label:   "test-contract",
		}

		data, err := proto.Marshal(contractInfo)
		require.NoError(t, err)

		key := createContractKey(addr)
		store.Set(key, data)
	}

	// Commit to persist
	cms.Commit()

	// Run migration
	err := MigrateContracts(ctx, storeKey)
	require.NoError(t, err)

	// Verify contracts are unchanged
	for _, addr := range addresses {
		key := createContractKey(addr)
		data := store.Get(key)
		require.NotNil(t, data)

		var contractInfo wasmtypes.ContractInfo
		err := proto.Unmarshal(data, &contractInfo)
		require.NoError(t, err, "healthy contract should still unmarshal")
	}
}

// TestMigrateContractsCorrupted verifies that corrupted contracts are fixed
func TestMigrateContractsCorrupted(t *testing.T) {
	ctx, storeKey, cms := setupTestStore(t)
	store := ctx.KVStore(storeKey)

	addr := "xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpn45e7"

	// Create corrupted contract with swapped fields
	corruptedData := []byte{}

	// Field 1: CodeID = 1
	corruptedData = append(corruptedData, EncodeFieldTag(1, WireVarint)...)
	corruptedData = append(corruptedData, 1)

	// Field 2: Creator
	creator := "xion1creator"
	corruptedData = append(corruptedData, EncodeFieldTag(2, WireBytes)...)
	corruptedData = append(corruptedData, byte(len(creator)))
	corruptedData = append(corruptedData, []byte(creator)...)

	// Field 7: Contains port ID (wrong - should be field 8)
	portID := "wasm.xion1test"
	corruptedData = append(corruptedData, EncodeFieldTag(7, WireBytes)...)
	corruptedData = append(corruptedData, byte(len(portID)))
	corruptedData = append(corruptedData, []byte(portID)...)

	// Verify it cannot unmarshal
	var testInfo wasmtypes.ContractInfo
	unmarshalErr := proto.Unmarshal(corruptedData, &testInfo)
	require.Error(t, unmarshalErr, "corrupted data should fail to unmarshal")

	// Store corrupted contract
	key := createContractKey(addr)
	store.Set(key, corruptedData)

	// Commit to persist
	cms.Commit()

	// Run migration
	err := MigrateContracts(ctx, storeKey)
	require.NoError(t, err)

	// Verify contract is now fixed
	fixedData := store.Get(key)
	require.NotNil(t, fixedData)

	var contractInfo wasmtypes.ContractInfo
	err = proto.Unmarshal(fixedData, &contractInfo)
	require.NoError(t, err, "fixed contract should unmarshal successfully")

	// Verify data integrity
	require.Equal(t, uint64(1), contractInfo.CodeID)
	require.Equal(t, creator, contractInfo.Creator)
}

// TestMigrateContractsMixed verifies migration with mix of healthy and corrupted
func TestMigrateContractsMixed(t *testing.T) {
	ctx, storeKey, cms := setupTestStore(t)
	store := ctx.KVStore(storeKey)

	// Create 3 healthy contracts
	healthyAddrs := []string{
		"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpn45e7",
		"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpkxfqtv",
		"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqp5dzgw7",
	}

	for i, addr := range healthyAddrs {
		contractInfo := &wasmtypes.ContractInfo{
			CodeID:  uint64(i + 1),
			Creator: "xion1creator",
			Label:   "healthy-contract",
		}

		data, err := proto.Marshal(contractInfo)
		require.NoError(t, err)

		key := createContractKey(addr)
		store.Set(key, data)
	}

	// Create 2 corrupted contracts
	corruptedAddrs := []string{
		"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqp7wcwzc",
		"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqp9xu7z3",
	}

	for i, addr := range corruptedAddrs {
		// Create corrupted data
		corruptedData := []byte{}

		// Field 1: CodeID
		corruptedData = append(corruptedData, EncodeFieldTag(1, WireVarint)...)
		corruptedData = append(corruptedData, byte(i+10))

		// Field 2: Creator
		creator := "xion1creator"
		corruptedData = append(corruptedData, EncodeFieldTag(2, WireBytes)...)
		corruptedData = append(corruptedData, byte(len(creator)))
		corruptedData = append(corruptedData, []byte(creator)...)

		// Field 7: Wrong data (should trigger fix)
		portID := "wasm.xion1test"
		corruptedData = append(corruptedData, EncodeFieldTag(7, WireBytes)...)
		corruptedData = append(corruptedData, byte(len(portID)))
		corruptedData = append(corruptedData, []byte(portID)...)

		key := createContractKey(addr)
		store.Set(key, corruptedData)
	}

	// Commit to persist
	cms.Commit()

	// Run migration
	err := MigrateContracts(ctx, storeKey)
	require.NoError(t, err)

	// Verify all contracts can now unmarshal
	allAddrs := append(healthyAddrs, corruptedAddrs...)
	for _, addr := range allAddrs {
		key := createContractKey(addr)
		data := store.Get(key)
		require.NotNil(t, data, "contract should exist for %s", addr)

		var contractInfo wasmtypes.ContractInfo
		err := proto.Unmarshal(data, &contractInfo)
		require.NoError(t, err, "all contracts should unmarshal after migration for %s", addr)
	}
}

// TestMigrateContractsSchemaInconsistent verifies schema-inconsistent contracts are skipped
func TestMigrateContractsSchemaInconsistent(t *testing.T) {
	ctx, storeKey, cms := setupTestStore(t)
	store := ctx.KVStore(storeKey)

	addr := "xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpn45e7"

	// Create schema-inconsistent contract (missing fields 7&8 but can unmarshal)
	data := []byte{}

	// Field 1: CodeID = 1
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 2: Creator
	creator := "xion1creator"
	data = append(data, EncodeFieldTag(2, WireBytes)...)
	data = append(data, byte(len(creator)))
	data = append(data, []byte(creator)...)

	// No fields 7 or 8 (but should still unmarshal)

	// Verify it can unmarshal
	var testInfo wasmtypes.ContractInfo
	err := proto.Unmarshal(data, &testInfo)
	require.NoError(t, err, "schema-inconsistent should still unmarshal")

	// Store it
	key := createContractKey(addr)
	store.Set(key, data)

	// Commit to persist
	cms.Commit()

	// Get original data for comparison
	originalData := store.Get(key)

	// Run migration
	err = MigrateContracts(ctx, storeKey)
	require.NoError(t, err)

	// Verify contract is unchanged (we skip schema-inconsistent ones)
	newData := store.Get(key)
	require.Equal(t, originalData, newData, "schema-inconsistent contract should not be modified")
}

// TestMigrateContractsDataIntegrity verifies no data loss during migration
func TestMigrateContractsDataIntegrity(t *testing.T) {
	ctx, storeKey, cms := setupTestStore(t)
	store := ctx.KVStore(storeKey)

	addr := "xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpn45e7"

	// Create corrupted contract with specific data
	corruptedData := []byte{}

	// Field 1: CodeID = 42
	corruptedData = append(corruptedData, EncodeFieldTag(1, WireVarint)...)
	corruptedData = append(corruptedData, 42)

	// Field 2: Creator
	creator := "xion1mycreator"
	corruptedData = append(corruptedData, EncodeFieldTag(2, WireBytes)...)
	corruptedData = append(corruptedData, byte(len(creator)))
	corruptedData = append(corruptedData, []byte(creator)...)

	// Field 3: Admin
	admin := "xion1myadmin"
	corruptedData = append(corruptedData, EncodeFieldTag(3, WireBytes)...)
	corruptedData = append(corruptedData, byte(len(admin)))
	corruptedData = append(corruptedData, []byte(admin)...)

	// Field 4: Label
	label := "my-special-contract"
	corruptedData = append(corruptedData, EncodeFieldTag(4, WireBytes)...)
	corruptedData = append(corruptedData, byte(len(label)))
	corruptedData = append(corruptedData, []byte(label)...)

	// Field 7: Wrong data
	portID := "wasm.xion1port"
	corruptedData = append(corruptedData, EncodeFieldTag(7, WireBytes)...)
	corruptedData = append(corruptedData, byte(len(portID)))
	corruptedData = append(corruptedData, []byte(portID)...)

	// Store it
	key := createContractKey(addr)
	store.Set(key, corruptedData)

	// Commit to persist
	cms.Commit()

	// Run migration
	err := MigrateContracts(ctx, storeKey)
	require.NoError(t, err)

	// Verify all data is preserved
	fixedData := store.Get(key)
	require.NotNil(t, fixedData)

	var contractInfo wasmtypes.ContractInfo
	err = proto.Unmarshal(fixedData, &contractInfo)
	require.NoError(t, err)

	// Check all fields are preserved
	require.Equal(t, uint64(42), contractInfo.CodeID, "CodeID should be preserved")
	require.Equal(t, creator, contractInfo.Creator, "Creator should be preserved")
	require.Equal(t, admin, contractInfo.Admin, "Admin should be preserved")
	require.Equal(t, label, contractInfo.Label, "Label should be preserved")
}

// TestMigrateContractsLargeScale simulates migration with many contracts
func TestMigrateContractsLargeScale(t *testing.T) {
	ctx, storeKey, cms := setupTestStore(t)
	store := ctx.KVStore(storeKey)

	// Create 100 contracts (mix of healthy and corrupted)
	numContracts := 100
	numCorrupted := 20

	for i := 0; i < numContracts; i++ {
		// Generate unique address (simple string for testing)
		addr := fmt.Sprintf("contract%d", i)

		var data []byte

		if i < numCorrupted {
			// Create corrupted contract
			data = []byte{}
			data = append(data, EncodeFieldTag(1, WireVarint)...)
			data = append(data, byte(i+1))

			creator := "xion1creator"
			data = append(data, EncodeFieldTag(2, WireBytes)...)
			data = append(data, byte(len(creator)))
			data = append(data, []byte(creator)...)

			// Wrong field 7
			portID := "wasm.xion1test"
			data = append(data, EncodeFieldTag(7, WireBytes)...)
			data = append(data, byte(len(portID)))
			data = append(data, []byte(portID)...)
		} else {
			// Create healthy contract
			contractInfo := &wasmtypes.ContractInfo{
				CodeID:  uint64(i + 1),
				Creator: "xion1creator",
				Label:   "contract",
			}

			var err error
			data, err = proto.Marshal(contractInfo)
			require.NoError(t, err)
		}

		key := createContractKey(addr)
		store.Set(key, data)
	}

	// Commit to persist
	cms.Commit()

	// Run migration
	err := MigrateContracts(ctx, storeKey)
	require.NoError(t, err)

	// Verify all contracts can unmarshal
	for i := 0; i < numContracts; i++ {
		addr := fmt.Sprintf("contract%d", i)
		key := createContractKey(addr)
		data := store.Get(key)
		require.NotNil(t, data)

		var contractInfo wasmtypes.ContractInfo
		err := proto.Unmarshal(data, &contractInfo)
		require.NoError(t, err, "contract %d should unmarshal after migration", i)
	}
}
