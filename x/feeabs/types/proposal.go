package types

import (
	v1beta1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

var (
	_ v1beta1types.Content = &AddHostZoneProposal{}
	_ v1beta1types.Content = &DeleteHostZoneProposal{}
	_ v1beta1types.Content = &SetHostZoneProposal{}
)

const (
	// ProposalTypeAddHostZone defines the type for a AddHostZoneProposal
	ProposalTypeAddHostZone    = "AddHostZone"
	ProposalTypeDeleteHostZone = "DeleteHostZone"
	ProposalTypeSetHostZone    = "SetHostZone"
)

func init() {
	v1beta1types.RegisterProposalType(ProposalTypeAddHostZone)
	v1beta1types.RegisterProposalType(ProposalTypeDeleteHostZone)
	v1beta1types.RegisterProposalType(ProposalTypeSetHostZone)
}

// NewAddHostZoneProposal creates a new add host zone proposal.
func NewAddHostZoneProposal(title, description string, config HostChainFeeAbsConfig) v1beta1types.Content {
	return &AddHostZoneProposal{
		Title:           title,
		Description:     description,
		HostChainConfig: &config,
	}
}

func NewDeleteHostZoneProposal(title, description, ibcDenom string) v1beta1types.Content {
	return &DeleteHostZoneProposal{
		Title:       title,
		Description: description,
		IbcDenom:    ibcDenom,
	}
}

func NewSetHostZoneProposal(title, description string, config HostChainFeeAbsConfig) v1beta1types.Content {
	return &SetHostZoneProposal{
		Title:           title,
		Description:     description,
		HostChainConfig: &config,
	}
}

// GetTitle returns the title of an add host zone proposal.
func (ahzp *AddHostZoneProposal) GetTitle() string { return ahzp.Title }

// GetDescription returns the description of an add host zone proposal.
func (ahzp *AddHostZoneProposal) GetDescription() string { return ahzp.Description }

// ProposalRoute returns the routing key of an add host zone proposal.
func (*AddHostZoneProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type of an add host zone proposal.
func (*AddHostZoneProposal) ProposalType() string { return ProposalTypeAddHostZone }

// ValidateBasic runs basic stateless validity checks
func (ahzp *AddHostZoneProposal) ValidateBasic() error {
	err := v1beta1types.ValidateAbstract(ahzp)
	if err != nil {
		return err
	}

	// TODO: add validate here

	return nil
}

// GetTitle returns the title of a delete host zone proposal.
func (dhzp *DeleteHostZoneProposal) GetTitle() string { return dhzp.Title }

// GetDescription returns the description of a delete host zone proposal.
func (dhzp *DeleteHostZoneProposal) GetDescription() string { return dhzp.Description }

// ProposalRoute returns the routing key of a delete host zone proposal.
func (*DeleteHostZoneProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type of a delete host zone proposal.
func (*DeleteHostZoneProposal) ProposalType() string { return ProposalTypeDeleteHostZone }

// ValidateBasic runs basic stateless validity checks
func (dhzp *DeleteHostZoneProposal) ValidateBasic() error {
	err := v1beta1types.ValidateAbstract(dhzp)
	if err != nil {
		return err
	}

	// TODO: add validate here

	return nil
}

// GetTitle returns the title of a set host zone proposal.
func (shzp *SetHostZoneProposal) GetTitle() string { return shzp.Title }

// GetDescription returns the description of a set host zone proposal.
func (shzp *SetHostZoneProposal) GetDescription() string { return shzp.Description }

// ProposalRoute returns the routing key of a set host zone proposal.
func (*SetHostZoneProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type of a set host zone proposal.
func (*SetHostZoneProposal) ProposalType() string { return ProposalTypeSetHostZone }

// ValidateBasic runs basic stateless validity checks
func (shzp *SetHostZoneProposal) ValidateBasic() error {
	err := v1beta1types.ValidateAbstract(shzp)
	if err != nil {
		return err
	}

	// TODO: add validate here

	return nil
}
