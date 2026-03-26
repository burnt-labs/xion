package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// RegisterLegacyAminoCodec registers the necessary x/tee interfaces and concrete types
// on the provided LegacyAmino codec. This module is query-only so nothing to register.
func RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {}

// RegisterInterfaces registers the x/tee interfaces types with the interface registry.
// This module is query-only so nothing to register.
func RegisterInterfaces(_ cdctypes.InterfaceRegistry) {}
