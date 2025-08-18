package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRevokeAuthzGrants{}, "grantmanager/MsgRevokeAuthzGrants", nil)
	cdc.RegisterConcrete(&MsgRevokeFeegrantAllowances{}, "grantmanager/MsgRevokeFeegrantAllowances", nil)

}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRevokeAuthzGrants{},
		&MsgRevokeFeegrantAllowances{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
