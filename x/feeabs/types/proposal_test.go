package types

import (
	"testing"

	v1beta1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/stretchr/testify/require"
)

func TestProposalConstants(t *testing.T) {
	require.Equal(t, "AddHostZone", ProposalTypeAddHostZone)
	require.Equal(t, "DeleteHostZone", ProposalTypeDeleteHostZone)
	require.Equal(t, "SetHostZone", ProposalTypeSetHostZone)
}

func TestNewAddHostZoneProposal(t *testing.T) {
	title := "Test Add Host Zone"
	description := "Test description for adding host zone"
	config := HostChainFeeAbsConfig{
		IbcDenom:                "ibc/123",
		OsmosisPoolTokenDenomIn: "uosmo",
		PoolId:                  1,
		Status:                  HostChainFeeAbsStatus_UPDATED,
	}

	proposal := NewAddHostZoneProposal(title, description, config)

	require.NotNil(t, proposal)
	addProposal, ok := proposal.(*AddHostZoneProposal)
	require.True(t, ok)

	require.Equal(t, title, addProposal.Title)
	require.Equal(t, description, addProposal.Description)
	require.NotNil(t, addProposal.HostChainConfig)
	require.Equal(t, config.IbcDenom, addProposal.HostChainConfig.IbcDenom)
}

func TestNewDeleteHostZoneProposal(t *testing.T) {
	title := "Test Delete Host Zone"
	description := "Test description for deleting host zone"
	ibcDenom := "ibc/456"

	proposal := NewDeleteHostZoneProposal(title, description, ibcDenom)

	require.NotNil(t, proposal)
	deleteProposal, ok := proposal.(*DeleteHostZoneProposal)
	require.True(t, ok)

	require.Equal(t, title, deleteProposal.Title)
	require.Equal(t, description, deleteProposal.Description)
	require.Equal(t, ibcDenom, deleteProposal.IbcDenom)
}

func TestNewSetHostZoneProposal(t *testing.T) {
	title := "Test Set Host Zone"
	description := "Test description for setting host zone"
	config := HostChainFeeAbsConfig{
		IbcDenom:                "ibc/789",
		OsmosisPoolTokenDenomIn: "uosmo",
		PoolId:                  2,
		Status:                  HostChainFeeAbsStatus_FROZEN,
	}

	proposal := NewSetHostZoneProposal(title, description, config)

	require.NotNil(t, proposal)
	setProposal, ok := proposal.(*SetHostZoneProposal)
	require.True(t, ok)

	require.Equal(t, title, setProposal.Title)
	require.Equal(t, description, setProposal.Description)
	require.NotNil(t, setProposal.HostChainConfig)
	require.Equal(t, config.IbcDenom, setProposal.HostChainConfig.IbcDenom)
}

func TestAddHostZoneProposal(t *testing.T) {
	config := HostChainFeeAbsConfig{
		IbcDenom:                "ibc/123",
		OsmosisPoolTokenDenomIn: "uosmo",
		PoolId:                  1,
		Status:                  HostChainFeeAbsStatus_UPDATED,
	}

	proposal := &AddHostZoneProposal{
		Title:           "Test Title",
		Description:     "Test Description",
		HostChainConfig: &config,
	}

	// Test GetTitle
	require.Equal(t, "Test Title", proposal.GetTitle())

	// Test GetDescription
	require.Equal(t, "Test Description", proposal.GetDescription())

	// Test ProposalRoute
	require.Equal(t, RouterKey, proposal.ProposalRoute())

	// Test ProposalType
	require.Equal(t, ProposalTypeAddHostZone, proposal.ProposalType())

	// Test ValidateBasic
	err := proposal.ValidateBasic()
	require.NoError(t, err)
}

func TestDeleteHostZoneProposal(t *testing.T) {
	proposal := &DeleteHostZoneProposal{
		Title:       "Test Delete Title",
		Description: "Test Delete Description",
		IbcDenom:    "ibc/456",
	}

	// Test GetTitle
	require.Equal(t, "Test Delete Title", proposal.GetTitle())

	// Test GetDescription
	require.Equal(t, "Test Delete Description", proposal.GetDescription())

	// Test ProposalRoute
	require.Equal(t, RouterKey, proposal.ProposalRoute())

	// Test ProposalType
	require.Equal(t, ProposalTypeDeleteHostZone, proposal.ProposalType())

	// Test ValidateBasic
	err := proposal.ValidateBasic()
	require.NoError(t, err)
}

func TestSetHostZoneProposal(t *testing.T) {
	config := HostChainFeeAbsConfig{
		IbcDenom:                "ibc/789",
		OsmosisPoolTokenDenomIn: "uosmo",
		PoolId:                  2,
		Status:                  HostChainFeeAbsStatus_FROZEN,
	}

	proposal := &SetHostZoneProposal{
		Title:           "Test Set Title",
		Description:     "Test Set Description",
		HostChainConfig: &config,
	}

	// Test GetTitle
	require.Equal(t, "Test Set Title", proposal.GetTitle())

	// Test GetDescription
	require.Equal(t, "Test Set Description", proposal.GetDescription())

	// Test ProposalRoute
	require.Equal(t, RouterKey, proposal.ProposalRoute())

	// Test ProposalType
	require.Equal(t, ProposalTypeSetHostZone, proposal.ProposalType())

	// Test ValidateBasic
	err := proposal.ValidateBasic()
	require.NoError(t, err)
}

func TestProposalValidateBasicInvalidCases(t *testing.T) {
	tests := []struct {
		name     string
		proposal v1beta1types.Content
	}{
		{
			name: "AddHostZoneProposal with empty title",
			proposal: &AddHostZoneProposal{
				Title:       "",
				Description: "Valid description",
				HostChainConfig: &HostChainFeeAbsConfig{
					IbcDenom: "ibc/123",
				},
			},
		},
		{
			name: "DeleteHostZoneProposal with empty title",
			proposal: &DeleteHostZoneProposal{
				Title:       "",
				Description: "Valid description",
				IbcDenom:    "ibc/456",
			},
		},
		{
			name: "SetHostZoneProposal with empty title",
			proposal: &SetHostZoneProposal{
				Title:       "",
				Description: "Valid description",
				HostChainConfig: &HostChainFeeAbsConfig{
					IbcDenom: "ibc/789",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.proposal.ValidateBasic()
			require.Error(t, err)
		})
	}
}

func TestProposalInterfaces(t *testing.T) {
	// Test that all proposals implement the Content interface
	var _ v1beta1types.Content = &AddHostZoneProposal{}
	var _ v1beta1types.Content = &DeleteHostZoneProposal{}
	var _ v1beta1types.Content = &SetHostZoneProposal{}
}
