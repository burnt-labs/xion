package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers concrete types on the LegacyAmino codec
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgAddDkimPubKeys{}, ModuleName+"/MsgAddDkimPubKey", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	// this line is used by starport scaffolding # 3

	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgAddDkimPubKeys{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
