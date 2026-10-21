package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	RegisterCodec(cdc)
}

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateAudience{}, "jwk/CreateAudience", nil)
	cdc.RegisterConcrete(&MsgUpdateAudience{}, "jwk/UpdateAudience", nil)
	cdc.RegisterConcrete(&MsgDeleteAudience{}, "jwk/DeleteAudience", nil)
	cdc.RegisterConcrete(&MsgCreateAudienceClaim{}, "jwk/CreateAudienceClaim", nil)
	cdc.RegisterConcrete(&MsgDeleteAudienceClaim{}, "jwk/DeleteAudienceClaim", nil)
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "jwk/MsgUpdateParams")
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateAudience{},
		&MsgUpdateAudience{},
		&MsgDeleteAudience{},
		&MsgCreateAudienceClaim{},
		&MsgDeleteAudienceClaim{},
		&MsgUpdateParams{},
	)
	// this line is used by starport scaffolding # 3

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
