package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
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

// RegisterLegacyAminoCodec registers the necessary x/zk interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgAddVKey{}, "zk/MsgAddVKey")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateVKey{}, "zk/MsgUpdateVKey")
	legacy.RegisterAminoMsg(cdc, &MsgRemoveVKey{}, "zk/MsgRemoveVKey")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "zk/MsgUpdateParams")
}

// RegisterInterfaces registers the x/zk interfaces types with the interface registry
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgAddVKey{},
		&MsgUpdateVKey{},
		&MsgRemoveVKey{},
		&MsgUpdateParams{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
