package keepers

import (
	"context"
	"fmt"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"google.golang.org/protobuf/encoding/protowire"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// XionWasmKeeper extends wasmd keeper with Xion-specific functionality
// and backward compatibility for the v0.61.2 → v0.61.6 protobuf format change.
//
// This keeper provides:
//  1. Automatic handling of the field 7/8 swap in ContractInfo between wasmd versions
//  2. Future extension points for Xion-specific validation and business logic
//  3. Transparent compatibility without modifying the wasmd fork
//
// The compatibility issue:
// - v0.61.2: field 7 = ibc2_port_id (string), field 8 = extension (Any)
// - v0.61.6: field 7 = extension (Any), field 8 = ibc2_port_id (string)
type XionWasmKeeper struct {
	wasmkeeper.Keeper
	cdc          codec.Codec
	storeService store.KVStoreService
	logger       log.Logger
	convertCache map[string][]byte // Cache for converted contract data
}

// NewXionWasmKeeper creates a new Xion-specific wasm keeper with backward compatibility
func NewXionWasmKeeper(
	keeper wasmkeeper.Keeper,
	cdc codec.Codec,
	storeService store.KVStoreService,
	logger log.Logger,
) *XionWasmKeeper {
	return &XionWasmKeeper{
		Keeper:       keeper,
		cdc:          cdc,
		storeService: storeService,
		logger:       logger.With("module", "xion-wasm"),
		convertCache: make(map[string][]byte),
	}
}

// GetContractInfo overrides the base method to handle legacy v0.61.2 format
// This ensures contracts stored with the old protobuf schema can still be read
func (k *XionWasmKeeper) GetContractInfo(ctx sdk.Context, contractAddr sdk.AccAddress) *wasmtypes.ContractInfo {
	store := k.storeService.OpenKVStore(ctx)
	key := wasmtypes.GetContractAddressKey(contractAddr)

	bz, err := store.Get(key)
	if err != nil || bz == nil {
		return nil
	}

	// Try to unmarshal with current format first
	var contractInfo wasmtypes.ContractInfo
	if err := k.cdc.Unmarshal(bz, &contractInfo); err == nil {
		return &contractInfo
	}

	// Failed to unmarshal - likely legacy v0.61.2 format
	k.logger.Debug("standard unmarshal failed, trying legacy format",
		"contract", contractAddr.String(),
		"error", err)

	// Check cache first to avoid repeated conversions
	cacheKey := fmt.Sprintf("contract:%s", contractAddr.String())
	if cachedData, exists := k.convertCache[cacheKey]; exists {
		if err := k.cdc.Unmarshal(cachedData, &contractInfo); err == nil {
			return &contractInfo
		}
	}

	// Convert from legacy format
	convertedBz, err := k.convertLegacyContractInfo(bz)
	if err != nil {
		k.logger.Error("failed to convert legacy contract info",
			"contract", contractAddr.String(),
			"error", err)
		return nil
	}

	// Try unmarshaling the converted data
	if err := k.cdc.Unmarshal(convertedBz, &contractInfo); err != nil {
		k.logger.Error("failed to unmarshal converted contract info",
			"contract", contractAddr.String(),
			"error", err)
		return nil
	}

	// Cache the successful conversion
	k.convertCache[cacheKey] = convertedBz

	k.logger.Info("successfully converted legacy contract info",
		"contract", contractAddr.String())
	return &contractInfo
}

// QuerySmart overrides to ensure contract is readable before querying
// This prevents panics when querying contracts stored in legacy format
func (k *XionWasmKeeper) QuerySmart(ctx context.Context, contractAddr sdk.AccAddress, req []byte) ([]byte, error) {
	// Ensure we can read the contract info first (handles format conversion if needed)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	contractInfo := k.GetContractInfo(sdkCtx, contractAddr)
	if contractInfo == nil {
		return nil, wasmtypes.ErrNoSuchContractFn(contractAddr.String()).
			Wrapf("address %s", contractAddr.String())
	}

	// Now proceed with the query using the base keeper
	return k.Keeper.QuerySmart(ctx, contractAddr, req)
}

// HasContractInfo checks if a contract exists, handling legacy format
func (k *XionWasmKeeper) HasContractInfo(ctx sdk.Context, contractAddr sdk.AccAddress) bool {
	return k.GetContractInfo(ctx, contractAddr) != nil
}

// convertLegacyContractInfo converts protobuf data from v0.61.2 to v0.61.6 format
// by swapping fields 7 and 8 to match the new schema
func (k *XionWasmKeeper) convertLegacyContractInfo(oldData []byte) ([]byte, error) {
	// Parse the protobuf message field by field
	fields := make(map[int][]byte)

	// Read all fields from the old data
	for len(oldData) > 0 {
		// Read field header (field number and wire type)
		fieldNum, wireType, n := protowire.ConsumeTag(oldData)
		if n < 0 {
			return nil, fmt.Errorf("failed to read field tag")
		}
		oldData = oldData[n:]

		// Store the complete field (including tag) for reconstruction
		fieldTag := protowire.AppendTag(nil, fieldNum, wireType)

		// Read field value based on wire type
		switch wireType {
		case protowire.VarintType:
			val, n := protowire.ConsumeVarint(oldData)
			if n < 0 {
				return nil, fmt.Errorf("failed to read varint field %d", fieldNum)
			}
			fields[int(fieldNum)] = append(fieldTag, protowire.AppendVarint(nil, val)...)
			oldData = oldData[n:]

		case protowire.BytesType:
			val, n := protowire.ConsumeBytes(oldData)
			if n < 0 {
				return nil, fmt.Errorf("failed to read bytes field %d", fieldNum)
			}
			fields[int(fieldNum)] = append(fieldTag, protowire.AppendBytes(nil, val)...)
			oldData = oldData[n:]

		case protowire.Fixed32Type:
			val, n := protowire.ConsumeFixed32(oldData)
			if n < 0 {
				return nil, fmt.Errorf("failed to read fixed32 field %d", fieldNum)
			}
			fields[int(fieldNum)] = append(fieldTag, protowire.AppendFixed32(nil, val)...)
			oldData = oldData[n:]

		case protowire.Fixed64Type:
			val, n := protowire.ConsumeFixed64(oldData)
			if n < 0 {
				return nil, fmt.Errorf("failed to read fixed64 field %d", fieldNum)
			}
			fields[int(fieldNum)] = append(fieldTag, protowire.AppendFixed64(nil, val)...)
			oldData = oldData[n:]

		default:
			return nil, fmt.Errorf("unknown wire type %d for field %d", wireType, fieldNum)
		}
	}

	// Swap fields 7 and 8 to match the new schema
	// v0.61.2: field 7 = ibc2_port_id, field 8 = extension
	// v0.61.6: field 7 = extension, field 8 = ibc2_port_id
	if field7, ok7 := fields[7]; ok7 {
		if field8, ok8 := fields[8]; ok8 {
			// Both fields exist, swap them
			k.logger.Debug("swapping fields 7 and 8 for legacy compatibility")

			// Create new field data with swapped field numbers
			newField7 := k.changeFieldNumber(field8, 7) // extension becomes field 7
			newField8 := k.changeFieldNumber(field7, 8) // ibc2_port_id becomes field 8

			fields[7] = newField7
			fields[8] = newField8
		}
	}

	// Reconstruct the protobuf message with fields in order
	var result []byte
	for i := 1; i <= 10; i++ { // ContractInfo has at most 8 fields, using 10 for safety
		if fieldData, ok := fields[i]; ok {
			result = append(result, fieldData...)
		}
	}

	return result, nil
}

// changeFieldNumber updates the field number in protobuf field data
func (k *XionWasmKeeper) changeFieldNumber(fieldData []byte, newFieldNum int) []byte {
	if len(fieldData) == 0 {
		return fieldData
	}

	// Validate field number is within valid protobuf range
	// Protobuf field numbers must be between 1 and 536,870,911 (2^29 - 1)
	// For ContractInfo, we only use fields 1-10, so this check is defensive
	if newFieldNum < 1 || newFieldNum > 536870911 {
		k.logger.Error("invalid field number for protobuf", "fieldNum", newFieldNum)
		return fieldData // Return unchanged if field number is invalid
	}

	// Parse the existing tag
	_, wireType, n := protowire.ConsumeTag(fieldData)
	if n < 0 {
		return fieldData // Return unchanged if we can't parse
	}

	// Create new tag with the new field number but same wire type
	// Safe conversion: we've validated newFieldNum is within int32 range
	//nolint:gosec
	fieldNum32 := int32(newFieldNum) // Safe after range validation above
	newTag := protowire.AppendTag(nil, protowire.Number(fieldNum32), wireType)

	// Combine new tag with the existing value
	return append(newTag, fieldData[n:]...)
}

// ValidateContractLabel provides Xion-specific label validation
// This is an extension point for custom business logic
func (k *XionWasmKeeper) ValidateContractLabel(label string) error {
	// Add any Xion-specific label validation here
	// For example: minimum/maximum length, character restrictions, etc.
	return nil
}

// GetXionContractMetadata retrieves contract info with Xion-specific extensions
// This demonstrates how to add custom metadata while maintaining compatibility
func (k *XionWasmKeeper) GetXionContractMetadata(
	ctx context.Context,
	contractAddr sdk.AccAddress,
) (*XionContractMetadata, error) {
	// Use our overridden GetContractInfo that handles legacy format
	contractInfo := k.GetContractInfo(sdk.UnwrapSDKContext(ctx), contractAddr)
	if contractInfo == nil {
		return nil, wasmtypes.ErrNotFound
	}

	// Return with any Xion-specific extensions
	metadata := &XionContractMetadata{
		ContractInfo: contractInfo,
		// Add Xion-specific fields as needed
	}

	return metadata, nil
}

// XionContractMetadata extends standard contract info with Xion-specific data
type XionContractMetadata struct {
	ContractInfo *wasmtypes.ContractInfo
	// Add Xion-specific fields as needed
	// XionVersion  string
	// XionFeatures []string
	// CustomFields map[string]interface{}
}

// Test helper methods - only exposed for testing package
// These methods are exported for use in tests but should not be used in production code

// TestChangeFieldNumber is a test helper that exposes the changeFieldNumber method for testing
func (k *XionWasmKeeper) TestChangeFieldNumber(fieldData []byte, newFieldNum int) []byte {
	return k.changeFieldNumber(fieldData, newFieldNum)
}

// TestConvertLegacyContractInfo is a test helper that exposes the convertLegacyContractInfo method for testing
func (k *XionWasmKeeper) TestConvertLegacyContractInfo(oldData []byte) ([]byte, error) {
	return k.convertLegacyContractInfo(oldData)
}
