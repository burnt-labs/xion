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
	cdc.RegisterConcrete(&MsgRemoveDkimPubKey{}, ModuleName+"/MsgRemoveDkimPubKey", nil)
	cdc.RegisterConcrete(&MsgRevokeDkimPubKey{}, ModuleName+"/MsgRevokeDkimPubKey", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, ModuleName+"/MsgUpdateParams", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	// this line is used by starport scaffolding # 3

	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgAddDkimPubKeys{},
		&MsgRemoveDkimPubKey{},
		&MsgRevokeDkimPubKey{},
		&MsgUpdateParams{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
