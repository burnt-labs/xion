package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"

	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// RegisterLegacyAminoCodec registers the necessary x/zk interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgAddVKey{}, "zk/MsgAddVKey", nil)
	cdc.RegisterConcrete(&MsgUpdateVKey{}, "zk/MsgUpdateVKey", nil)
	cdc.RegisterConcrete(&MsgRemoveVKey{}, "zk/MsgRemoveVKey", nil)
}

// RegisterInterfaces registers the x/zk interfaces types with the interface registry
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgAddVKey{},
		&MsgUpdateVKey{},
		&MsgRemoveVKey{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
