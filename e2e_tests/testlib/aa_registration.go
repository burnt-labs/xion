package testlib

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/interchaintest/v10/chain/cosmos"

	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	aatypes "github.com/burnt-labs/xion/x/abstractaccount/types"
)

// AccountWasmChecksum returns the sha256 checksum of the xion_account.wasm
// test contract. The e2e chains use it as the abstract-account address
// derivation hash so accounts land on the same addresses as when addresses
// were derived from this contract's checksum, keeping the pre-generated ZK
// proofs (which are bound to those addresses) valid.
func AccountWasmChecksum() []byte {
	bz, err := os.ReadFile(IntegrationTestPath("testdata", "contracts", "xion_account.wasm"))
	if err != nil {
		panic(fmt.Sprintf("read xion_account.wasm: %v", err))
	}
	sum := sha256.Sum256(bz)
	return sum[:]
}

// QueryAAContractAddress returns the address the chain will instantiate (or
// has instantiated) an abstract account at for the given sender and salt. The
// address is derived with the module-managed address derivation hash, not the
// implementation code's checksum, so it must be obtained from the chain.
func QueryAAContractAddress(t *testing.T, ctx context.Context, node *cosmos.ChainNode, sender, salt string) string {
	res, err := ExecQuery(t, ctx, node, "abstract-account", "account-address", sender, "--salt", salt)
	require.NoError(t, err)
	addr, ok := res["address"].(string)
	require.True(t, ok, "account-address query must return an address")
	return addr
}

// EnableAARegistration configures the abstract-account module's address
// derivation hash and enables registration via governance. Chains upgraded
// through the module's v3 migration come out with registration paused until
// the hash is configured. The proposal is submitted by validator 0 so no new
// account is created that would shift account numbers.
func EnableAARegistration(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, proposalID uint64) {
	paramsResp, err := ExecQuery(t, ctx, xion.GetNode(), "abstract-account", "params")
	require.NoError(t, err)
	paramsJSON, err := json.Marshal(paramsResp)
	require.NoError(t, err)

	var params aatypes.Params
	require.NoError(t, xion.Config().EncodingConfig.Codec.UnmarshalJSON(paramsJSON, &params))
	params.AddressDerivationHash = AccountWasmChecksum()
	params.RegistrationEnabled = true

	msg := &aatypes.MsgUpdateParams{
		Sender: Authority,
		Params: &params,
	}

	validatorNode := xion.Validators[0]
	validatorAddr, err := validatorNode.KeyBech32(ctx, "validator", "acc")
	require.NoError(t, err)

	proposal, err := xion.BuildProposal(
		[]cosmos.ProtoMessage{msg},
		"Enable abstract account registration",
		"Configure the address derivation hash and enable abstract account registration",
		"",
		"500000000"+xion.Config().Denom, // greater than min deposit
		validatorAddr,
		false,
	)
	require.NoError(t, err)

	_, err = xion.SubmitProposal(ctx, "validator", proposal)
	require.NoError(t, err)

	prop, err := xion.GovQueryProposal(ctx, proposalID)
	require.NoError(t, err)
	require.Equal(t, govv1beta1.StatusVotingPeriod, prop.Status)

	require.NoError(t, xion.VoteOnProposalAllValidators(ctx, prop.ProposalId, cosmos.ProposalVoteYes))

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := xion.GovQueryProposal(ctx, proposalID)
		return err == nil && proposalInfo.Status == govv1beta1.StatusPassed
	}, time.Second*30, time.Second, "enable-registration proposal %d did not pass", proposalID)

	require.Eventuallyf(t, func() bool {
		res, err := ExecQuery(t, ctx, xion.GetNode(), "abstract-account", "params")
		if err != nil {
			return false
		}
		enabled, _ := res["registration_enabled"].(bool)
		return enabled
	}, time.Second*30, time.Second, "abstract account registration was not enabled")
}
