package params

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

func TestParameterChangeProposalCodec(t *testing.T) {
	proposal := &ParameterChangeProposal{
		Title:       "set minimum gas price",
		Description: "update the global fee",
		Changes: []ParamChange{{
			Subspace: "globalfee",
			Key:      "MinimumGasPricesParam",
			Value:    `[{"denom":"uxion","amount":"0.001"}]`,
		}},
	}

	registry := codectypes.NewInterfaceRegistry()
	RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	packed, err := codectypes.NewAnyWithValue(proposal)
	require.NoError(t, err)
	require.Equal(t, "/cosmos.params.v1beta1.ParameterChangeProposal", packed.TypeUrl)
	bz, err := cdc.Marshal(packed)
	require.NoError(t, err)
	var decodedAny codectypes.Any
	require.NoError(t, cdc.Unmarshal(bz, &decodedAny))
	var content govv1beta1.Content
	require.NoError(t, cdc.UnpackAny(&decodedAny, &content))
	require.Equal(t, proposal, content)
	require.Equal(t, proposal.Title, content.GetTitle())
	require.Equal(t, proposal.Description, content.GetDescription())
	require.Equal(t, "params", content.ProposalRoute())
	require.Equal(t, "ParameterChange", content.ProposalType())
	require.NoError(t, content.ValidateBasic())

	amino := codec.NewLegacyAmino()
	RegisterLegacyAminoCodec(amino)
	bz, err = amino.MarshalJSON(proposal)
	require.NoError(t, err)
	require.Contains(t, string(bz), `"type":"cosmos-sdk/ParameterChangeProposal"`)
	var aminoDecoded ParameterChangeProposal
	require.NoError(t, amino.UnmarshalJSON(bz, &aminoDecoded))
	require.Equal(t, proposal, &aminoDecoded)
}

func TestParameterChangeProposalValidateBasic(t *testing.T) {
	proposal := ParameterChangeProposal{
		Title:       "set minimum gas price",
		Description: "update the global fee",
		Changes:     []ParamChange{{Subspace: "globalfee", Key: "MinimumGasPricesParam", Value: `[]`}},
	}
	require.NoError(t, proposal.ValidateBasic())

	blankTitle := proposal
	blankTitle.Title = " "
	require.ErrorContains(t, blankTitle.ValidateBasic(), "proposal title cannot be blank")

	emptyChanges := proposal
	emptyChanges.Changes = nil
	require.EqualError(t, emptyChanges.ValidateBasic(), "submitted parameter changes are empty")
}
