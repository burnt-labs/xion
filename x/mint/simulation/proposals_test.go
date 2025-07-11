package simulation_test

import (
	"math/rand"
	"testing"

	"gotest.tools/v3/assert"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/burnt-labs/xion/x/mint/simulation"
	"github.com/burnt-labs/xion/x/mint/types"
)

func TestProposalMsgs(t *testing.T) {
	// initialize parameters
	s := rand.NewSource(1)
	//nolint:gosec // G404: Use of weak random number generator (math/rand instead of crypto/rand)
	r := rand.New(s)

	ctx := sdk.NewContext(nil, tmproto.Header{}, true, nil)
	accounts := simtypes.RandomAccounts(r, 3)

	// execute ProposalMsgs function
	weightedProposalMsgs := simulation.ProposalMsgs()
	assert.Assert(t, len(weightedProposalMsgs) == 1)

	w0 := weightedProposalMsgs[0]

	// tests w0 interface:
	assert.Equal(t, simulation.OpWeightMsgUpdateParams, w0.AppParamsKey())
	assert.Equal(t, simulation.DefaultWeightMsgUpdateParams, w0.DefaultWeight())

	msg := w0.MsgSimulatorFn()(r, ctx, accounts)
	msgUpdateParams, ok := msg.(*types.MsgUpdateParams)
	assert.Assert(t, ok)

	assert.Equal(t, sdk.AccAddress(address.Module("gov")).String(), msgUpdateParams.Authority)
	assert.Equal(t, uint64(122877), msgUpdateParams.Params.BlocksPerYear)
	assert.DeepEqual(t, math.LegacyNewDecWithPrec(95, 2), msgUpdateParams.Params.GoalBonded)
	assert.DeepEqual(t, math.LegacyNewDecWithPrec(94, 2), msgUpdateParams.Params.InflationMax)
	assert.DeepEqual(t, math.LegacyNewDecWithPrec(23, 2), msgUpdateParams.Params.InflationMin)
	assert.DeepEqual(t, math.LegacyNewDecWithPrec(89, 2), msgUpdateParams.Params.InflationRateChange)
	assert.Equal(t, "XhhuTSkuxK", msgUpdateParams.Params.MintDenom)
}
