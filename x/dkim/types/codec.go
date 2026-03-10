package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

var amino = codec.NewLegacyAmino()

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	sdk.RegisterLegacyAminoCodec(amino)
}

// RegisterLegacyAminoCodec registers concrete types on the LegacyAmino codec
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgAddDkimPubKeys{}, ModuleName+"/MsgAddDkimPubKeys")
	legacy.RegisterAminoMsg(cdc, &MsgRemoveDkimPubKey{}, ModuleName+"/MsgRemoveDkimPubKey")
	legacy.RegisterAminoMsg(cdc, &MsgRevokeDkimPubKey{}, ModuleName+"/MsgRevokeDkimPubKey")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, ModuleName+"/MsgUpdateParams")
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
