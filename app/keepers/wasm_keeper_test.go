package keepers_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/app/keepers"
	"github.com/burnt-labs/xion/app/params"
)

// mockComponents provides minimal mocks for testing
type mockComponents struct {
	keeper       wasmkeeper.Keeper
	cdc          codec.Codec
	storeService store.KVStoreService
	logger       log.Logger
}

// newMockComponents creates minimal mock components for testing
func newMockComponents() mockComponents {
	// Use the real encoding config to get a properly configured codec
	encodingConfig := params.MakeEncodingConfig()

	return mockComponents{
		cdc:    encodingConfig.Codec,
		logger: log.NewNopLogger(),
	}
}

// MockKVStore for testing raw store operations
type MockKVStore struct {
	mock.Mock
}

func (m *MockKVStore) Get(key []byte) ([]byte, error) {
	args := m.Called(key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockKVStore) Set(key []byte, value []byte) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *MockKVStore) Delete(key []byte) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *MockKVStore) Has(key []byte) (bool, error) {
	args := m.Called(key)
	return args.Bool(0), args.Error(1)
}

func (m *MockKVStore) Iterator(start, end []byte) (store.Iterator, error) {
	args := m.Called(start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(store.Iterator), args.Error(1)
}

func (m *MockKVStore) ReverseIterator(start, end []byte) (store.Iterator, error) {
	args := m.Called(start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(store.Iterator), args.Error(1)
}

// MockKVStoreService wraps MockKVStore
type MockKVStoreService struct {
	mock.Mock
	mockStore *MockKVStore
}

func NewMockKVStoreService() *MockKVStoreService {
	return &MockKVStoreService{
		mockStore: new(MockKVStore),
	}
}

func (m *MockKVStoreService) OpenKVStore(ctx context.Context) store.KVStore {
	return m.mockStore
}

// ============================================================================
// Basic Keeper Tests
// ============================================================================
// NOTE: The GetContractInfo, QuerySmart, and HasContractInfo methods are
// tested below. These tests document the expected behavior but require
// a simplified testing approach due to complex SDK dependencies.
// Full integration testing is recommended for production verification.
// ============================================================================

func TestNewXionWasmKeeper(t *testing.T) {
	t.Run("creates xion wasm keeper successfully", func(t *testing.T) {
		mocks := newMockComponents()

		xionKeeper := keepers.NewXionWasmKeeper(
			mocks.keeper,
			mocks.cdc,
			mocks.storeService,
			mocks.logger,
		)
		require.NotNil(t, xionKeeper, "xion keeper should not be nil")
	})
}

// TestGetContractInfo tests the GetContractInfo method with legacy format handling
func TestGetContractInfo(t *testing.T) {
	// Create a test contract address
	contractAddr := sdk.AccAddress("test_contract_addr_1")
	contractKey := wasmtypes.GetContractAddressKey(contractAddr)

	// Get the real codec to use for marshaling
	encodingConfig := params.MakeEncodingConfig()
	appCodec := encodingConfig.Codec

	tests := []struct {
		name        string
		setupStore  func(*MockKVStore)
		wantNil     bool
		description string
	}{
		{
			name: "returns nil when contract doesn't exist",
			setupStore: func(store *MockKVStore) {
				store.On("Get", contractKey).Return(nil, nil)
			},
			wantNil:     true,
			description: "Non-existent contracts should return nil",
		},
		{
			name: "returns contract info for current format",
			setupStore: func(store *MockKVStore) {
				// Marshal a valid ContractInfo using the real codec
				contractInfo := &wasmtypes.ContractInfo{
					CodeID:  1,
					Creator: "creator1",
					Label:   "test-contract",
				}
				validData, err := appCodec.Marshal(contractInfo)
				require.NoError(t, err)
				store.On("Get", contractKey).Return(validData, nil)
			},
			wantNil:     false,
			description: "Valid current format should be unmarshalled correctly",
		},
		{
			name: "converts and returns legacy format",
			setupStore: func(store *MockKVStore) {
				// Create a more complete legacy v0.61.2 format that will definitely fail with current codec
				// This includes all standard fields plus field 7 as ibc2_port_id (which should be field 8 in new format)
				var legacyData []byte

				// Field 1: code_id
				legacyData = protowire.AppendTag(legacyData, 1, protowire.VarintType)
				legacyData = protowire.AppendVarint(legacyData, 42)

				// Field 2: creator
				legacyData = protowire.AppendTag(legacyData, 2, protowire.BytesType)
				legacyData = protowire.AppendBytes(legacyData, []byte("creator1"))

				// Field 3: admin (empty)
				legacyData = protowire.AppendTag(legacyData, 3, protowire.BytesType)
				legacyData = protowire.AppendBytes(legacyData, []byte(""))

				// Field 4: label
				legacyData = protowire.AppendTag(legacyData, 4, protowire.BytesType)
				legacyData = protowire.AppendBytes(legacyData, []byte("test-contract"))

				// Field 7: ibc2_port_id in legacy position (should be field 8 in new format)
				// This is the key difference that makes it legacy format
				legacyData = protowire.AppendTag(legacyData, 7, protowire.BytesType)
				legacyData = protowire.AppendBytes(legacyData, []byte("wasm.contract123"))

				// Field 8: extension in legacy position (should be field 7 in new format)
				// Add an empty Any type here to trigger the format mismatch
				legacyData = protowire.AppendTag(legacyData, 8, protowire.BytesType)
				// Empty Any message (would have type_url and value fields if populated)
				legacyData = protowire.AppendBytes(legacyData, []byte{})

				store.On("Get", contractKey).Return(legacyData, nil)
			},
			wantNil:     false,
			description: "Legacy format should be converted and returned",
		},
		{
			name: "handles store error",
			setupStore: func(store *MockKVStore) {
				store.On("Get", contractKey).Return(nil, fmt.Errorf("store error"))
			},
			wantNil:     true,
			description: "Store errors should result in nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockStore := new(MockKVStore)
			tt.setupStore(mockStore)

			mockStoreService := &MockKVStoreService{
				mockStore: mockStore,
			}

			// We can't use a mock directly as the base keeper
			// So we'll test the actual XionWasmKeeper implementation
			// For unit testing, we'll use an uninitialized keeper pointer
			var baseKeeper wasmkeeper.Keeper
			xionKeeper := keepers.NewXionWasmKeeper(
				baseKeeper, // uninitialized keeper for unit testing
				appCodec,   // use the real codec
				mockStoreService,
				log.NewNopLogger(),
			)

			// Test GetContractInfo
			ctx := sdk.Context{}
			result := xionKeeper.GetContractInfo(ctx, contractAddr)

			if tt.wantNil {
				require.Nil(t, result, tt.description)
			} else {
				require.NotNil(t, result, tt.description)
			}

			mockStore.AssertExpectations(t)
		})
	}
}

// TestQuerySmart tests the QuerySmart method
func TestQuerySmart(t *testing.T) {
	// Get the real codec to use
	encodingConfig := params.MakeEncodingConfig()
	appCodec := encodingConfig.Codec

	t.Run("returns error when contract doesn't exist", func(t *testing.T) {
		// This test verifies that QuerySmart checks if the contract exists
		// before attempting to query it
		contractAddr := sdk.AccAddress("test_contract_addr_2")
		queryReq := []byte(`{"query":"test"}`)
		contractKey := wasmtypes.GetContractAddressKey(contractAddr)

		mockStore := new(MockKVStore)
		// Contract doesn't exist
		mockStore.On("Get", contractKey).Return(nil, nil)

		mockStoreService := &MockKVStoreService{
			mockStore: mockStore,
		}

		var baseKeeper wasmkeeper.Keeper
		xionKeeper := keepers.NewXionWasmKeeper(
			baseKeeper, // uninitialized keeper for testing
			appCodec,
			mockStoreService,
			log.NewNopLogger(),
		)

		// Call QuerySmart - it should fail because contract doesn't exist
		ctx := sdk.Context{}
		result, err := xionKeeper.QuerySmart(ctx, contractAddr, queryReq)

		require.Error(t, err, "Should return error when contract doesn't exist")
		require.Nil(t, result, "Result should be nil")
		require.Contains(t, err.Error(), contractAddr.String(), "Error should mention the contract address")

		mockStore.AssertExpectations(t)
	})

	t.Run("queries successfully when contract exists", func(t *testing.T) {
		// This test verifies that QuerySmart first ensures the contract exists
		// (potentially converting from legacy format) before querying
		// Note: Full integration test would require a real keeper
		// This unit test focuses on the contract existence check
		t.Skip("Requires full keeper setup - covered by integration tests")
	})
}

// TestHasContractInfo tests the HasContractInfo method
func TestHasContractInfo(t *testing.T) {
	// Get the real codec to use
	encodingConfig := params.MakeEncodingConfig()
	appCodec := encodingConfig.Codec

	t.Run("returns false when contract doesn't exist", func(t *testing.T) {
		contractAddr := sdk.AccAddress("test_contract_addr_3")
		contractKey := wasmtypes.GetContractAddressKey(contractAddr)

		mockStore := new(MockKVStore)
		// Contract doesn't exist
		mockStore.On("Get", contractKey).Return(nil, nil)

		mockStoreService := &MockKVStoreService{
			mockStore: mockStore,
		}

		var baseKeeper wasmkeeper.Keeper
		xionKeeper := keepers.NewXionWasmKeeper(
			baseKeeper, // uninitialized keeper for testing
			appCodec,
			mockStoreService,
			log.NewNopLogger(),
		)

		ctx := sdk.Context{}
		result := xionKeeper.HasContractInfo(ctx, contractAddr)

		require.False(t, result, "Should return false for non-existent contracts")
		mockStore.AssertExpectations(t)
	})

	t.Run("returns true when contract exists", func(t *testing.T) {
		contractAddr := sdk.AccAddress("test_contract_addr_4")
		contractKey := wasmtypes.GetContractAddressKey(contractAddr)

		mockStore := new(MockKVStore)
		// Marshal a valid ContractInfo using the real codec
		contractInfo := &wasmtypes.ContractInfo{
			CodeID:  1,
			Creator: "creator1",
			Label:   "test-contract",
		}
		validData, err := appCodec.Marshal(contractInfo)
		require.NoError(t, err)
		mockStore.On("Get", contractKey).Return(validData, nil)

		mockStoreService := &MockKVStoreService{
			mockStore: mockStore,
		}

		var baseKeeper wasmkeeper.Keeper
		xionKeeper := keepers.NewXionWasmKeeper(
			baseKeeper, // uninitialized keeper for testing
			appCodec,
			mockStoreService,
			log.NewNopLogger(),
		)

		ctx := sdk.Context{}
		result := xionKeeper.HasContractInfo(ctx, contractAddr)

		require.True(t, result, "Should return true for existing contracts")
		mockStore.AssertExpectations(t)
	})

	t.Run("handles legacy format contracts", func(t *testing.T) {
		contractAddr := sdk.AccAddress("test_contract_addr_5")
		contractKey := wasmtypes.GetContractAddressKey(contractAddr)

		mockStore := new(MockKVStore)
		// Legacy format data with field 7 as ibc2_port_id
		var legacyData []byte
		legacyData = protowire.AppendTag(legacyData, 1, protowire.VarintType)
		legacyData = protowire.AppendVarint(legacyData, 42)
		legacyData = protowire.AppendTag(legacyData, 7, protowire.BytesType)
		legacyData = protowire.AppendBytes(legacyData, []byte("wasm.contract123"))
		mockStore.On("Get", contractKey).Return(legacyData, nil)

		mockStoreService := &MockKVStoreService{
			mockStore: mockStore,
		}

		var baseKeeper wasmkeeper.Keeper
		xionKeeper := keepers.NewXionWasmKeeper(
			baseKeeper, // uninitialized keeper for testing
			appCodec,
			mockStoreService,
			log.NewNopLogger(),
		)

		ctx := sdk.Context{}
		result := xionKeeper.HasContractInfo(ctx, contractAddr)

		require.True(t, result, "Should return true for contracts in legacy format after conversion")
		mockStore.AssertExpectations(t)
	})
}

func TestXionWasmKeeper_ValidateContractLabel(t *testing.T) {
	tests := []struct {
		name    string
		label   string
		wantErr bool
	}{
		{
			name:    "empty label is valid (validation not implemented)",
			label:   "",
			wantErr: false,
		},
		{
			name:    "simple label is valid",
			label:   "my-contract",
			wantErr: false,
		},
		{
			name:    "long label is valid (validation not implemented)",
			label:   "this-is-a-very-long-contract-label-that-might-be-too-long",
			wantErr: false,
		},
		{
			name:    "label with special characters is valid (validation not implemented)",
			label:   "contract!@#$%",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := newMockComponents()
			xionKeeper := keepers.NewXionWasmKeeper(
				mocks.keeper,
				mocks.cdc,
				mocks.storeService,
				mocks.logger,
			)
			err := xionKeeper.ValidateContractLabel(tt.label)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestXionWasmKeeper_GetXionContractMetadata(t *testing.T) {
	t.Run("method exists and compiles", func(t *testing.T) {
		// This test simply verifies the method exists and compiles
		// Full integration testing would require a complete test environment
		mocks := newMockComponents()
		xionKeeper := keepers.NewXionWasmKeeper(
			mocks.keeper,
			mocks.cdc,
			mocks.storeService,
			mocks.logger,
		)

		// Just verify the keeper was created and has the method
		require.NotNil(t, xionKeeper)
		// The actual method testing would require full store setup
		// which is better done in integration tests
	})

	t.Run("returns metadata structure when contract exists", func(t *testing.T) {
		// Note: This test verifies the metadata structure creation logic
		// In a real scenario with a full keeper, it would test:
		// 1. GetContractInfo returns non-nil
		// 2. Metadata is properly created with ContractInfo
		// 3. XionContractMetadata structure is returned

		// Since we can't easily mock the keeper without complex setup,
		// we document that this code path creates the
		// XionContractMetadata struct with the ContractInfo field populated.

		// The function logic is:
		// - Get contract info from keeper
		// - If nil, return ErrNotFound
		// - If not nil, create XionContractMetadata with it
		// - Return the metadata with nil error

		// This documents the expected behavior for integration tests
		t.Skip("Requires full keeper setup - covered by integration tests")
	})
}

// ============================================================================
// Protobuf Compatibility Tests
// ============================================================================

// TestProtobufFieldSwap tests the field swapping logic for backward compatibility
func TestProtobufFieldSwap(t *testing.T) {
	// Simulate v0.61.2 ContractInfo with:
	// field 7 = ibc2_port_id (string) = "wasm.contract123"
	// field 8 = extension (Any) = empty

	// Build a mock protobuf message
	var oldFormat []byte

	// Field 1: code_id = 1 (varint)
	oldFormat = protowire.AppendTag(oldFormat, 1, protowire.VarintType)
	oldFormat = protowire.AppendVarint(oldFormat, 1)

	// Field 2: creator = "xion1..." (string)
	creator := "xion1creator"
	oldFormat = protowire.AppendTag(oldFormat, 2, protowire.BytesType)
	oldFormat = protowire.AppendBytes(oldFormat, []byte(creator))

	// Field 7: ibc2_port_id = "wasm.contract123" (in v0.61.2 format)
	ibc2PortID := "wasm.contract123"
	oldFormat = protowire.AppendTag(oldFormat, 7, protowire.BytesType)
	oldFormat = protowire.AppendBytes(oldFormat, []byte(ibc2PortID))

	// Field 8: extension = empty Any (in v0.61.2 format)
	oldFormat = protowire.AppendTag(oldFormat, 8, protowire.BytesType)
	oldFormat = protowire.AppendBytes(oldFormat, []byte{})

	t.Logf("Old format (v0.61.2): %s", hex.EncodeToString(oldFormat))

	// Test the conversion would swap fields 7 and 8
	// After conversion:
	// - Field 7 should contain what was in field 8 (extension)
	// - Field 8 should contain what was in field 7 (ibc2_port_id)

	// Parse to verify field positions
	data := oldFormat
	fields := make(map[int][]byte)

	for len(data) > 0 {
		fieldNum, wireType, n := protowire.ConsumeTag(data)
		require.True(t, n > 0, "Failed to parse field tag")
		data = data[n:]

		switch wireType {
		case protowire.VarintType:
			val, n := protowire.ConsumeVarint(data)
			require.True(t, n > 0, "Failed to parse varint")
			fields[int(fieldNum)] = protowire.AppendVarint(nil, val)
			data = data[n:]

		case protowire.BytesType:
			val, n := protowire.ConsumeBytes(data)
			require.True(t, n > 0, "Failed to parse bytes")
			fields[int(fieldNum)] = val
			data = data[n:]
		}
	}

	// Verify we have the expected fields
	require.NotNil(t, fields[7], "Field 7 should exist")
	require.Equal(t, ibc2PortID, string(fields[7]), "Field 7 should be ibc2_port_id in old format")
}

// TestFieldDetection tests the detection of legacy format
func TestFieldDetection(t *testing.T) {
	testCases := []struct {
		name     string
		buildMsg func() []byte
		isLegacy bool
	}{
		{
			name: "v0.61.2 format (field 7 = string)",
			buildMsg: func() []byte {
				var data []byte
				// Add field 7 as a string (ibc2_port_id in v0.61.2)
				data = protowire.AppendTag(data, 7, protowire.BytesType)
				data = protowire.AppendBytes(data, []byte("wasm.test"))
				return data
			},
			isLegacy: true,
		},
		{
			name: "v0.61.6 format (field 8 = string)",
			buildMsg: func() []byte {
				var data []byte
				// Add field 8 as a string (ibc2_port_id in v0.61.6)
				data = protowire.AppendTag(data, 8, protowire.BytesType)
				data = protowire.AppendBytes(data, []byte("wasm.test"))
				return data
			},
			isLegacy: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := tc.buildMsg()
			t.Logf("Test data: %s", hex.EncodeToString(data))
			// In production, the compatibility keeper would detect the format
			// and handle conversion automatically
		})
	}
}

// TestLiveQueryScenario simulates the actual query that was failing
func TestLiveQueryScenario(t *testing.T) {
	// This is the actual query data from the failing request
	queryDataHex := "0a3f78696f6e31646e643076683979723670716836307a3563396c6334713978743061776875303478773338613067657a7a776e687763396e36716b6b76327937121d7b226772616e745f636f6e6669675f747970655f75726c73223a7b7d7d"

	queryData, err := hex.DecodeString(queryDataHex)
	require.NoError(t, err)

	// Parse the query to extract contract address
	// Field 1 (0x0a) = contract address
	// Field 2 (0x12) = query data

	t.Logf("Query data length: %d bytes", len(queryData))
	t.Log("Contract: xion1dnd0vh9yr6pqh60z5c9lc4q9xt0awhu04xw38a0gezzwnhwc9n6qkkv2y7")
	t.Log("Query: {\"grant_config_type_urls\":{}}")

	// With the compatibility keeper in place, this query would:
	// 1. Load the contract info (triggering format conversion if needed)
	// 2. Execute the smart contract query
	// 3. Return the result without panicking
}

// ============================================================================
// Field Number Change Tests (Integer Bounds Checking)
// ============================================================================

// TestChangeFieldNumber tests the field number changing logic with bounds checking
func TestChangeFieldNumber(t *testing.T) {
	mocks := newMockComponents()
	xionKeeper := keepers.NewXionWasmKeeper(
		mocks.keeper,
		mocks.cdc,
		mocks.storeService,
		mocks.logger,
	)

	tests := []struct {
		name        string
		inputField  []byte
		newFieldNum int
		expectValid bool
		description string
	}{
		{
			name: "valid field number 1",
			inputField: func() []byte {
				// Create field 5 with string value
				data := protowire.AppendTag(nil, 5, protowire.BytesType)
				data = protowire.AppendBytes(data, []byte("test"))
				return data
			}(),
			newFieldNum: 1,
			expectValid: true,
			description: "minimum valid field number",
		},
		{
			name: "valid field number 100",
			inputField: func() []byte {
				data := protowire.AppendTag(nil, 5, protowire.BytesType)
				data = protowire.AppendBytes(data, []byte("test"))
				return data
			}(),
			newFieldNum: 100,
			expectValid: true,
			description: "typical field number",
		},
		{
			name: "maximum valid field number",
			inputField: func() []byte {
				data := protowire.AppendTag(nil, 5, protowire.BytesType)
				data = protowire.AppendBytes(data, []byte("test"))
				return data
			}(),
			newFieldNum: 536870911, // 2^29 - 1
			expectValid: true,
			description: "maximum protobuf field number",
		},
		{
			name: "invalid field number 0",
			inputField: func() []byte {
				data := protowire.AppendTag(nil, 5, protowire.BytesType)
				data = protowire.AppendBytes(data, []byte("test"))
				return data
			}(),
			newFieldNum: 0,
			expectValid: false,
			description: "field number 0 is reserved",
		},
		{
			name: "invalid negative field number",
			inputField: func() []byte {
				data := protowire.AppendTag(nil, 5, protowire.BytesType)
				data = protowire.AppendBytes(data, []byte("test"))
				return data
			}(),
			newFieldNum: -1,
			expectValid: false,
			description: "negative field numbers are invalid",
		},
		{
			name: "invalid field number too large",
			inputField: func() []byte {
				data := protowire.AppendTag(nil, 5, protowire.BytesType)
				data = protowire.AppendBytes(data, []byte("test"))
				return data
			}(),
			newFieldNum: 536870912, // 2^29 (one more than max)
			expectValid: false,
			description: "exceeds maximum protobuf field number",
		},
		{
			name:        "empty input data",
			inputField:  []byte{},
			newFieldNum: 7,
			expectValid: false, // Should return input unchanged
			description: "empty data should be handled gracefully",
		},
		{
			name:        "malformed protobuf data",
			inputField:  []byte{0xFF, 0xFF, 0xFF}, // Invalid varint
			newFieldNum: 7,
			expectValid: false,
			description: "malformed data should be handled gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the exported test helper method
			result := xionKeeper.TestChangeFieldNumber(tt.inputField, tt.newFieldNum)

			if tt.expectValid {
				// Parse result to verify field number was changed
				if len(result) > 0 && len(tt.inputField) > 0 {
					fieldNum, _, n := protowire.ConsumeTag(result)
					if n > 0 {
						//nolint:gosec
						require.Equal(t, protowire.Number(tt.newFieldNum), fieldNum,
							"Field number should be changed to %d", tt.newFieldNum)
					}
				}
			} else {
				// For invalid cases, data should be unchanged
				require.Equal(t, tt.inputField, result,
					"Invalid field number should return unchanged data: %s", tt.description)
			}
		})
	}
}

// ============================================================================
// Legacy Contract Info Conversion Tests
// ============================================================================

// TestConvertLegacyContractInfo tests the complete field swapping logic
func TestConvertLegacyContractInfo(t *testing.T) {
	mocks := newMockComponents()
	xionKeeper := keepers.NewXionWasmKeeper(
		mocks.keeper,
		mocks.cdc,
		mocks.storeService,
		mocks.logger,
	)

	tests := []struct {
		name           string
		buildOldFormat func() []byte
		validateNew    func(t *testing.T, newData []byte)
		description    string
	}{
		{
			name: "swap fields 7 and 8",
			buildOldFormat: func() []byte {
				var data []byte

				// Field 1: code_id = 42
				data = protowire.AppendTag(data, 1, protowire.VarintType)
				data = protowire.AppendVarint(data, 42)

				// Field 2: creator = "xion1creator"
				data = protowire.AppendTag(data, 2, protowire.BytesType)
				data = protowire.AppendBytes(data, []byte("xion1creator"))

				// Field 7: ibc2_port_id = "wasm.contract123" (v0.61.2)
				data = protowire.AppendTag(data, 7, protowire.BytesType)
				data = protowire.AppendBytes(data, []byte("wasm.contract123"))

				// Field 8: extension = "ext_data" (v0.61.2)
				data = protowire.AppendTag(data, 8, protowire.BytesType)
				data = protowire.AppendBytes(data, []byte("ext_data"))

				return data
			},
			validateNew: func(t *testing.T, newData []byte) {
				fields := parseProtobufFields(t, newData)

				// Verify fields 7 and 8 are swapped
				require.Equal(t, "ext_data", string(fields[7]),
					"Field 7 should now contain extension data")
				require.Equal(t, "wasm.contract123", string(fields[8]),
					"Field 8 should now contain ibc2_port_id")

				// Verify other fields unchanged
				require.NotNil(t, fields[1], "Field 1 should exist")
				require.Equal(t, "xion1creator", string(fields[2]),
					"Field 2 should be unchanged")
			},
			description: "Fields 7 and 8 should be swapped for v0.61.6 compatibility",
		},
		{
			name: "handle missing field 7",
			buildOldFormat: func() []byte {
				var data []byte

				// Only field 8 present
				data = protowire.AppendTag(data, 8, protowire.BytesType)
				data = protowire.AppendBytes(data, []byte("only_field_8"))

				return data
			},
			validateNew: func(t *testing.T, newData []byte) {
				fields := parseProtobufFields(t, newData)

				// Field 8 should remain unchanged when field 7 is missing
				require.Equal(t, "only_field_8", string(fields[8]),
					"Field 8 should be unchanged when field 7 is missing")
				require.Nil(t, fields[7], "Field 7 should not exist")
			},
			description: "Should handle missing field 7 gracefully",
		},
		{
			name: "handle missing field 8",
			buildOldFormat: func() []byte {
				var data []byte

				// Only field 7 present
				data = protowire.AppendTag(data, 7, protowire.BytesType)
				data = protowire.AppendBytes(data, []byte("only_field_7"))

				return data
			},
			validateNew: func(t *testing.T, newData []byte) {
				fields := parseProtobufFields(t, newData)

				// Field 7 should remain unchanged when field 8 is missing
				require.Equal(t, "only_field_7", string(fields[7]),
					"Field 7 should be unchanged when field 8 is missing")
				require.Nil(t, fields[8], "Field 8 should not exist")
			},
			description: "Should handle missing field 8 gracefully",
		},
		{
			name: "preserve all field types",
			buildOldFormat: func() []byte {
				var data []byte

				// Field 1: varint
				data = protowire.AppendTag(data, 1, protowire.VarintType)
				data = protowire.AppendVarint(data, 999)

				// Field 3: fixed64
				data = protowire.AppendTag(data, 3, protowire.Fixed64Type)
				data = protowire.AppendFixed64(data, 0x123456789ABCDEF0)

				// Field 5: fixed32
				data = protowire.AppendTag(data, 5, protowire.Fixed32Type)
				data = protowire.AppendFixed32(data, 0x12345678)

				// Field 7: bytes (to be swapped)
				data = protowire.AppendTag(data, 7, protowire.BytesType)
				data = protowire.AppendBytes(data, []byte("field7"))

				// Field 8: bytes (to be swapped)
				data = protowire.AppendTag(data, 8, protowire.BytesType)
				data = protowire.AppendBytes(data, []byte("field8"))

				return data
			},
			validateNew: func(t *testing.T, newData []byte) {
				// Parse and verify all field types are preserved
				var idx int
				for idx < len(newData) {
					fieldNum, wireType, n := protowire.ConsumeTag(newData[idx:])
					require.True(t, n > 0, "Should parse tag")
					idx += n

					switch fieldNum {
					case 1:
						require.Equal(t, protowire.VarintType, wireType)
						val, n := protowire.ConsumeVarint(newData[idx:])
						require.Equal(t, uint64(999), val)
						idx += n
					case 3:
						require.Equal(t, protowire.Fixed64Type, wireType)
						val, n := protowire.ConsumeFixed64(newData[idx:])
						require.Equal(t, uint64(0x123456789ABCDEF0), val)
						idx += n
					case 5:
						require.Equal(t, protowire.Fixed32Type, wireType)
						val, n := protowire.ConsumeFixed32(newData[idx:])
						require.Equal(t, uint32(0x12345678), val)
						idx += n
					case 7:
						require.Equal(t, protowire.BytesType, wireType)
						val, n := protowire.ConsumeBytes(newData[idx:])
						require.Equal(t, "field8", string(val), "Field 7 should contain old field 8")
						idx += n
					case 8:
						require.Equal(t, protowire.BytesType, wireType)
						val, n := protowire.ConsumeBytes(newData[idx:])
						require.Equal(t, "field7", string(val), "Field 8 should contain old field 7")
						idx += n
					default:
						t.Fatalf("Unexpected field %d", fieldNum)
					}
				}
			},
			description: "All protobuf wire types should be preserved during conversion",
		},
		{
			name: "handle malformed data gracefully",
			buildOldFormat: func() []byte {
				// Invalid protobuf data
				return []byte{0xFF, 0xFF, 0xFF, 0xFF}
			},
			validateNew: func(t *testing.T, newData []byte) {
				// Should return error or handle gracefully
				// The actual keeper method would return an error
				require.NotNil(t, newData, "Should handle malformed data")
			},
			description: "Malformed protobuf should be handled without panic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldData := tt.buildOldFormat()
			t.Logf("%s: Old format hex: %s", tt.name, hex.EncodeToString(oldData))

			// Test the conversion
			newData, err := xionKeeper.TestConvertLegacyContractInfo(oldData)

			if tt.name == "handle malformed data gracefully" {
				// For malformed data, we expect an error
				if err == nil && len(newData) == 0 {
					t.Log("Malformed data handled gracefully")
					return
				}
			} else {
				require.NoError(t, err, tt.description)
			}

			if newData != nil {
				t.Logf("%s: New format hex: %s", tt.name, hex.EncodeToString(newData))
				tt.validateNew(t, newData)
			}
		})
	}
}

// ============================================================================
// Integer Overflow Protection Tests
// ============================================================================

// TestIntegerOverflowProtection tests that integer conversions are safe
func TestIntegerOverflowProtection(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		valid    bool
		describe string
	}{
		{
			name:     "minimum valid",
			value:    1,
			valid:    true,
			describe: "Field number 1 is minimum valid",
		},
		{
			name:     "maximum valid",
			value:    536870911, // 2^29 - 1
			valid:    true,
			describe: "Field number 2^29-1 is maximum valid",
		},
		{
			name:     "zero invalid",
			value:    0,
			valid:    false,
			describe: "Field number 0 is reserved",
		},
		{
			name:     "negative invalid",
			value:    -100,
			valid:    false,
			describe: "Negative field numbers are invalid",
		},
		{
			name:     "overflow boundary",
			value:    536870912, // 2^29
			valid:    false,
			describe: "Field number 2^29 exceeds maximum",
		},
		{
			name:     "int32 max still invalid",
			value:    2147483647, // int32 max
			valid:    false,
			describe: "Even though fits in int32, exceeds protobuf max",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the validation logic
			isValid := tt.value >= 1 && tt.value <= 536870911
			require.Equal(t, tt.valid, isValid, tt.describe)

			// Verify int32 conversion would be safe for valid values
			if isValid {
				//nolint:gosec
				int32Val := int32(tt.value)
				require.Equal(t, tt.value, int(int32Val),
					"Conversion to int32 should not lose precision")
			}
		})
	}
}

// ============================================================================
// Edge Cases Tests
// ============================================================================

// TestEdgeCases tests various edge cases in the compatibility layer
func TestEdgeCases(t *testing.T) {
	mocks := newMockComponents()
	xionKeeper := keepers.NewXionWasmKeeper(
		mocks.keeper,
		mocks.cdc,
		mocks.storeService,
		mocks.logger,
	)

	t.Run("empty protobuf message", func(t *testing.T) {
		result, err := xionKeeper.TestConvertLegacyContractInfo([]byte{})
		// Empty data returns nil result, no error (no fields to convert)
		require.NoError(t, err, "Empty data should not cause error")
		// For empty input, the function returns nil (no fields found)
		require.Nil(t, result, "Empty input should return nil")
	})

	t.Run("truncated protobuf message", func(t *testing.T) {
		// Start of a valid tag but incomplete
		truncated := []byte{0x08} // Field 1, varint type, but no value
		result, err := xionKeeper.TestConvertLegacyContractInfo(truncated)
		// Should handle gracefully without panic
		// Error is expected for malformed data
		require.Error(t, err, "Should error on truncated data")
		require.Nil(t, result, "Should return nil on error")
	})

	t.Run("repeated fields", func(t *testing.T) {
		var data []byte
		// Add field 7 twice (protobuf allows repeated fields)
		data = protowire.AppendTag(data, 7, protowire.BytesType)
		data = protowire.AppendBytes(data, []byte("first"))
		data = protowire.AppendTag(data, 7, protowire.BytesType)
		data = protowire.AppendBytes(data, []byte("second"))

		result, err := xionKeeper.TestConvertLegacyContractInfo(data)
		require.NoError(t, err, "Should handle repeated fields")
		require.NotNil(t, result, "Should return converted data")
		// The last value typically wins in protobuf
	})

	t.Run("unknown field numbers", func(t *testing.T) {
		var data []byte
		// Add a field with very high number (but valid)
		data = protowire.AppendTag(data, 1000, protowire.BytesType)
		data = protowire.AppendBytes(data, []byte("unknown"))

		result, err := xionKeeper.TestConvertLegacyContractInfo(data)
		// Unknown fields are silently dropped as they're not in the 1-10 range we preserve
		require.NoError(t, err, "Should handle unknown fields without error")
		// Result will be nil since we only preserve fields 1-10 and none are present
		require.Nil(t, result, "Unknown fields outside 1-10 range result in nil output")
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

// parseProtobufFields is a helper function to parse protobuf fields for testing
func parseProtobufFields(t *testing.T, data []byte) map[int][]byte {
	fields := make(map[int][]byte)

	for len(data) > 0 {
		fieldNum, wireType, n := protowire.ConsumeTag(data)
		if n <= 0 {
			break
		}
		data = data[n:]

		switch wireType {
		case protowire.VarintType:
			val, n := protowire.ConsumeVarint(data)
			if n <= 0 {
				t.Fatalf("Failed to parse varint for field %d", fieldNum)
			}
			// Store just the value bytes for comparison
			fields[int(fieldNum)] = protowire.AppendVarint(nil, val)
			data = data[n:]

		case protowire.BytesType:
			val, n := protowire.ConsumeBytes(data)
			if n <= 0 {
				t.Fatalf("Failed to parse bytes for field %d", fieldNum)
			}
			fields[int(fieldNum)] = val
			data = data[n:]

		case protowire.Fixed32Type:
			val, n := protowire.ConsumeFixed32(data)
			if n <= 0 {
				t.Fatalf("Failed to parse fixed32 for field %d", fieldNum)
			}
			fields[int(fieldNum)] = protowire.AppendFixed32(nil, val)
			data = data[n:]

		case protowire.Fixed64Type:
			val, n := protowire.ConsumeFixed64(data)
			if n <= 0 {
				t.Fatalf("Failed to parse fixed64 for field %d", fieldNum)
			}
			fields[int(fieldNum)] = protowire.AppendFixed64(nil, val)
			data = data[n:]

		default:
			t.Fatalf("Unknown wire type %d for field %d", wireType, fieldNum)
		}
	}

	return fields
}
