// Package params keeps x/params ParameterChangeProposal decodable after SDK v0.55 removed x/params.
// Passed gov proposals in state (xion-mainnet-1 #4, #14, #42) wrap it in MsgExecLegacyContent.
// params.pb.go is vendored verbatim from cosmos-sdk v0.53.8 x/params/types/proposal (package clause changed).
// No proposal type or gov route is registered, so new ParameterChangeProposals are still rejected.
package params

import (
	"errors"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

var _ govv1beta1.Content = &ParameterChangeProposal{}

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&ParameterChangeProposal{}, "cosmos-sdk/ParameterChangeProposal", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*govv1beta1.Content)(nil), &ParameterChangeProposal{})
}

func (pcp *ParameterChangeProposal) GetTitle() string { return pcp.Title }

func (pcp *ParameterChangeProposal) GetDescription() string { return pcp.Description }

func (pcp *ParameterChangeProposal) ProposalRoute() string { return "params" }

func (pcp *ParameterChangeProposal) ProposalType() string { return "ParameterChange" }

func (pcp *ParameterChangeProposal) ValidateBasic() error {
	if err := govv1beta1.ValidateAbstract(pcp); err != nil {
		return err
	}
	if len(pcp.Changes) == 0 {
		return errors.New("submitted parameter changes are empty")
	}
	return nil
}
