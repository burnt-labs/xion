package types

import (
	"cosmossdk.io/x/feegrant"

	xionMintTypes "github.com/burnt-labs/xion/x/mint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the necessary x/bank interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgSend{}, "xion/MsgSend")
	legacy.RegisterAminoMsg(cdc, &MsgMultiSend{}, "xion/MsgMultiSend")
	legacy.RegisterAminoMsg(cdc, &MsgSetPlatformPercentage{}, "xion/MsgSetPlatformPercentage")
	legacy.RegisterAminoMsg(cdc, &xionMintTypes.MsgUpdateParams{}, "xion/x/mint/MsgUpdateParams")

	cdc.RegisterConcrete(&AuthzAllowance{}, "xion/AuthzAllowance", nil)
	cdc.RegisterConcrete(&ContractsAllowance{}, "xion/ContractsAllowance", nil)
	cdc.RegisterConcrete(&MultiAnyAllowance{}, "xion/MultiAnyAllowance", nil)
	cdc.RegisterConcrete(xionMintTypes.Params{}, "xion/x/mint/Params", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSend{},
		&MsgMultiSend{},
		&MsgSetPlatformPercentage{},
		&xionMintTypes.MsgUpdateParams{},
	)

	registry.RegisterInterface(
		"cosmos.feegrant.v1beta1.FeeAllowanceI",
		(*feegrant.FeeAllowanceI)(nil),
		&AuthzAllowance{},
		&ContractsAllowance{},
		&MultiAnyAllowance{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var amino = codec.NewLegacyAmino()

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	sdk.RegisterLegacyAminoCodec(amino)
}
