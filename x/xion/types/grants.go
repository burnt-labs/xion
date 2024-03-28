package types

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"

	proto "github.com/gogo/protobuf/proto"

	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (g CodeIdGrant) UnpackInterfaces(unpacker cdctypes.AnyUnpacker) error {
	var f wasmtypes.ContractAuthzFilterX
	if err := unpacker.UnpackAny(g.Filter, &f); err != nil {
		return errorsmod.Wrap(err, "filter")
	}
	var l wasmtypes.ContractAuthzLimitX
	if err := unpacker.UnpackAny(g.Limit, &l); err != nil {
		return errorsmod.Wrap(err, "limit")
	}
	return nil
}

// NewCodeIdGrant constructor
func NewCodeIdGrant(codeId string, limit wasmtypes.ContractAuthzLimitX, filter wasmtypes.ContractAuthzFilterX) (*CodeIdGrant, error) {
	pFilter, ok := filter.(proto.Message)
	if !ok {
		return nil, sdkerrors.ErrInvalidType.Wrap("filter is not a proto type")
	}
	anyFilter, err := cdctypes.NewAnyWithValue(pFilter)
	if err != nil {
		return nil, errorsmod.Wrap(err, "filter")
	}
	return CodeIdGrant{
		CodeId: codeId,
		Filter: anyFilter,
	}.WithNewLimits(limit)
}

// WithNewLimits factory method to create a new grant with given limit
func (g CodeIdGrant) WithNewLimits(limit wasmtypes.ContractAuthzLimitX) (*CodeIdGrant, error) {
	pLimit, ok := limit.(proto.Message)
	if !ok {
		return nil, sdkerrors.ErrInvalidType.Wrap("limit is not a proto type")
	}
	anyLimit, err := cdctypes.NewAnyWithValue(pLimit)
	if err != nil {
		return nil, errorsmod.Wrap(err, "limit")
	}

	return &CodeIdGrant{
		CodeId: g.CodeId,
		Limit:  anyLimit,
		Filter: g.Filter,
	}, nil
}

// ValidateBasic validates the grant
func (g CodeIdGrant) ValidateBasic() error {
	// CodeIdGrant uses the same limit as contract grant.
	// to avoid code duplication, we just validate the contract grant
	contractGrant := wasmtypes.ContractGrant{
		Contract: "",
		Limit:    g.Limit,
		Filter:   g.Filter,
	}
	// execution limits
	if err := contractGrant.GetLimit().ValidateBasic(); err != nil {
		return errorsmod.Wrap(err, "limit")
	}
	// filter
	if err := contractGrant.GetFilter().ValidateBasic(); err != nil {
		return errorsmod.Wrap(err, "filter")
	}
	return nil
}
