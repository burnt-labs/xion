package types

import (
	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/cosmos-sdk/codec"
	types "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	v1beta1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

var amino = codec.NewLegacyAmino()

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSendQueryIbcDenomTWAP{}, "feeabs/SendQueryIbcDenomTWAP", nil)
	cdc.RegisterConcrete(&MsgSwapCrossChain{}, "feeabs/SwapCrossChain", nil)
	cdc.RegisterConcrete(&MsgAddHostZone{}, "feeabs/AddHostZone", nil)
	cdc.RegisterConcrete(&AddHostZoneProposal{}, "feeabs/AddHostZoneProposal", nil)
	cdc.RegisterConcrete(&DeleteHostZoneProposal{}, "feeabs/DeleteHostZoneProposal", nil)
	cdc.RegisterConcrete(&SetHostZoneProposal{}, "feeabs/SetHostZoneProposal", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSendQueryIbcDenomTWAP{},
		&MsgSwapCrossChain{},
		&MsgFundFeeAbsModuleAccount{},
		&MsgUpdateParams{},
		&MsgAddHostZone{},
	)

	registry.RegisterImplementations(
		(*v1beta1types.Content)(nil),
		&AddHostZoneProposal{},
		&DeleteHostZoneProposal{},
		&SetHostZoneProposal{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)

	// Register legacy type URLs to maintain
	// backward compatibility with governance proposals stored before the proto package rename.
	// This allows the node to decode messages that were stored with the old package name.
	type customTypeURLRegistry interface {
		RegisterCustomTypeURL(iface any, typeURL string, impl proto.Message)
	}
	if customReg, ok := registry.(customTypeURLRegistry); ok {
		customReg.RegisterCustomTypeURL((*sdk.Msg)(nil), "/feeabstraction.feeabs.v1beta1.MsgUpdateParams", &MsgUpdateParams{})
		customReg.RegisterCustomTypeURL((*sdk.Msg)(nil), "/feeabstraction.feeabs.v1beta1.MsgSendQueryIbcDenomTWAP", &MsgSendQueryIbcDenomTWAP{})
		customReg.RegisterCustomTypeURL((*sdk.Msg)(nil), "/feeabstraction.feeabs.v1beta1.MsgSwapCrossChain", &MsgSwapCrossChain{})
		customReg.RegisterCustomTypeURL((*sdk.Msg)(nil), "/feeabstraction.feeabs.v1beta1.MsgFundFeeAbsModuleAccount", &MsgFundFeeAbsModuleAccount{})
		customReg.RegisterCustomTypeURL((*sdk.Msg)(nil), "/feeabstraction.feeabs.v1beta1.MsgAddHostZone", &MsgAddHostZone{})
		customReg.RegisterCustomTypeURL((*v1beta1types.Content)(nil), "/feeabstraction.feeabs.v1beta1.AddHostZoneProposal", &AddHostZoneProposal{})
		customReg.RegisterCustomTypeURL((*v1beta1types.Content)(nil), "/feeabstraction.feeabs.v1beta1.DeleteHostZoneProposal", &DeleteHostZoneProposal{})
		customReg.RegisterCustomTypeURL((*v1beta1types.Content)(nil), "/feeabstraction.feeabs.v1beta1.SetHostZoneProposal", &SetHostZoneProposal{})
	}
}

func init() {
	RegisterCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	sdk.RegisterLegacyAminoCodec(amino)
}
