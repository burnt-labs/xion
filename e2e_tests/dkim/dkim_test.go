package integration_tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/burnt-labs/xion/e2e_tests/testlib"
	dkimTypes "github.com/burnt-labs/xion/x/dkim/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govModule "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
)

const (
	pubKey_1 = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"
	pubKey_2 = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"
	pubKey_3 = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEApYmNCWAKIxf5uOEXIdEBPRDmMxcyiAnpDT3/xHad1n/d1yeZLhxCEOV6IeMNOIHD9p+VxqqzmFCvWkKvisBauAMxoJ0so7JHfjP3BOUb7hKOvcU4XiwyjhjMJQMNBImlB75Es04Kfu9RrC9tOFau5lN4ldjvNUjQH3YZoknK+LyXtJ8XBUrKdd4pptlzhMb3/J5q2wlHgUC0+jZUKtjCLHoHhQv7+vXdM2gZmPlmr5fofyAyPMPLdO5e65BXC2Z9kmSl1Zw3b41i9RlC8OwAyloI0Za/hzqQ/0sre9KtCNoPCtLhF/03dccG/282WkWCWVRxEBEC1q6s99GYm7SMqQIDAQAB"
)

const (
	domain_1 = "x.com"
	domain_2 = "xion.com"
)

const (
	selector_1 = "dkim202406"
	selector_2 = "dkim202407"
	selector_3 = "google"
)

var (
	poseidon_hash_1 = "1983664618407009423875829639306275185491946247764487749439145140682408188330"
	poseidon_hash_2 = "1983664618407009423875829639306275185491946247764487749439145140682408188330"
	poseidon_hash_3 = "14900978865743571023141723682019198695580050511337677317524514528673897510335"
)

const (
	customDomain   = "gmail.com"
	customSelector = "20230601"
)

var pubKeysBz, _ = json.Marshal([]dkimTypes.DkimPubKey{{
	PubKey:       pubKey_1,
	Domain:       domain_1,
	Selector:     selector_1,
	PoseidonHash: []byte(poseidon_hash_1),
}, {
	PubKey:       pubKey_2,
	Domain:       domain_2,
	Selector:     selector_2,
	PoseidonHash: []byte(poseidon_hash_2),
}, {
	PubKey:       pubKey_3,
	Domain:       domain_1,
	Selector:     selector_3,
	PoseidonHash: []byte(poseidon_hash_3),
}})

// TestDKIMModule tests basic DKIM query operations
func TestDKIMModule(t *testing.T) {
	ctx := t.Context()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	xion := testlib.BuildXionChain(t)

	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	chainUser := users[0]
	govModAddress := testlib.GetModuleAddress(t, xion, ctx, govModule.ModuleName)

	// First, add test DKIM records via governance since they're not in genesis
	hash1, err := dkimTypes.ComputePoseidonHash(pubKey_1)
	require.NoError(t, err)
	hash3, err := dkimTypes.ComputePoseidonHash(pubKey_3)
	require.NoError(t, err)

	seedRecords := []dkimTypes.DkimPubKey{
		{
			Domain:       domain_1,
			Selector:     selector_1,
			PubKey:       pubKey_1,
			PoseidonHash: []byte(hash1.String()),
		},
		{
			Domain:       domain_1,
			Selector:     selector_3,
			PubKey:       pubKey_3,
			PoseidonHash: []byte(hash3.String()),
		},
	}

	createDkimMsg := &dkimTypes.MsgAddDkimPubKeys{
		Authority:   govModAddress,
		DkimPubkeys: seedRecords,
	}
	require.NoError(t, createDkimMsg.ValidateBasic())

	err = createAndSubmitProposal(t, xion, ctx, chainUser, []cosmos.ProtoMessage{createDkimMsg}, "Seed DKIM records", "Seed DKIM records", "Seed DKIM records", 1)
	require.NoError(t, err)

	// Now test queries

	// Query single DKIM record by domain and selector
	dkimRecord, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "dkim", "dkim-pubkey", domain_1, selector_1)
	require.NoError(t, err)
	require.Equal(t, pubKey_1, dkimRecord["dkim_pub_key"].(map[string]interface{})["pub_key"].(string))

	// Query all records for x.com domain
	allDkimRecords, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "dkim", "qdkims", "--domain", domain_1)
	require.NoError(t, err)
	require.Len(t, allDkimRecords["dkim_pub_keys"].([]interface{}), 2)

	// Query by domain + poseidon hash pair (should return selector_3)
	allDkimRecords, err = testlib.ExecQuery(t, ctx, xion.GetNode(), "dkim", "qdkims", "--domain", domain_1, "--hash", hash3.String())
	require.NoError(t, err)
	require.Len(t, allDkimRecords["dkim_pub_keys"].([]interface{}), 1)
	require.Equal(t, selector_3, allDkimRecords["dkim_pub_keys"].([]interface{})[0].(map[string]interface{})["selector"])

	// Test gdkim DNS lookup (may fail due to external DNS changes)
	t.Run("gdkim_dns_lookup", func(t *testing.T) {
		dkimRecord, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "dkim", "gdkim", "gmail.com", "20230601")
		if err != nil {
			t.Skipf("DNS DKIM lookup failed (external dependency): %v", err)
			return
		}
		require.NotEmpty(t, dkimRecord["pub_key"])
		require.NotEmpty(t, dkimRecord["poseidon_hash"])
	})
}

// TestDKIMgoverGovernance tests adding and removing DKIM records via governance proposals
func TestDKIMGovernance(t *testing.T) {
	ctx := t.Context()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	xion := testlib.BuildXionChain(t)

	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	chainUser := users[0]
	govModAddress := testlib.GetModuleAddress(t, xion, ctx, govModule.ModuleName)

	// Use a hardcoded test key instead of DNS lookup
	// This avoids test flakiness from external DNS changes
	testDomain := "test.example.com"
	testSelector := "governance_test"
	testPubKey := pubKey_1 // Reuse existing test key

	hash, err := dkimTypes.ComputePoseidonHash(testPubKey)
	require.NoError(t, err)

	// Submit proposal to add the DKIM record
	governancePubKeys := []dkimTypes.DkimPubKey{
		{
			Domain:       testDomain,
			Selector:     testSelector,
			PubKey:       testPubKey,
			PoseidonHash: []byte(hash.String()),
		},
	}

	createDkimMsg := &dkimTypes.MsgAddDkimPubKeys{
		Authority:   govModAddress, // Pass string directly
		DkimPubkeys: governancePubKeys,
	}
	require.NoError(t, createDkimMsg.ValidateBasic())

	err = createAndSubmitProposal(t, xion, ctx, chainUser, []cosmos.ProtoMessage{createDkimMsg}, "Add test DKIM record", "Add test DKIM record", "Add test DKIM record", 1)
	require.NoError(t, err)

	// Verify the record was added
	dkimRecord, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "dkim", "dkim-pubkey", testDomain, testSelector)
	require.NoError(t, err)
	require.Equal(t, dkimRecord["dkim_pub_key"].(map[string]interface{})["pub_key"].(string), testPubKey)

	// Submit proposal to remove the DKIM record
	deleteDkimMsg := &dkimTypes.MsgRemoveDkimPubKey{
		Authority: govModAddress,
		Domain:    testDomain,
		Selector:  testSelector,
	}
	err = createAndSubmitProposal(t, xion, ctx, chainUser, []cosmos.ProtoMessage{deleteDkimMsg}, "Remove test DKIM record", "Remove test DKIM record", "Remove test DKIM record", 2)
	require.NoError(t, err)

	// Verify the record was removed
	_, err = testlib.ExecQuery(t, ctx, xion.GetNode(), "dkim", "dkim-pubkey", testDomain, testSelector)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func createAndSubmitProposal(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, proposer ibc.Wallet, proposalMsgs []cosmos.ProtoMessage, title, summary, metadata string, proposalId int) error {
	proposal, err := xion.BuildProposal(
		proposalMsgs,
		title,
		summary,
		metadata,
		"500000000"+xion.Config().Denom, // greater than min deposit
		proposer.FormattedAddress(),
		false,
	)
	require.NoError(t, err)

	height, err := xion.Height(ctx)
	require.NoError(t, err, "error fetching height before submit upgrade proposal")

	_, err = xion.SubmitProposal(ctx, proposer.KeyName(), proposal)
	require.NoError(t, err)

	prop, err := xion.GovQueryProposal(ctx, uint64(proposalId))
	require.NoError(t, err)
	require.Equal(t, govv1beta1.StatusVotingPeriod, prop.Status)

	err = xion.VoteOnProposalAllValidators(ctx, prop.ProposalId, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := xion.GovQueryProposal(ctx, uint64(prop.ProposalId))
		if err != nil {
			require.NoError(t, err)
		} else {
			if proposalInfo.Status == govv1beta1.StatusPassed {
				return true
			}
			t.Logf("Waiting for proposal to enter voting status PASSED, current status: %s", proposalInfo.Status)
		}
		return false
	}, time.Second*11, time.Second, "failed to reach status PASSED after 11s")

	afterVoteHeight, err := xion.Height(ctx)
	require.NoError(t, err, "error fetching height after voting on proposal")

	_, err = cosmos.PollForProposalStatus(ctx, xion, height, afterVoteHeight, prop.ProposalId, govv1beta1.StatusPassed)
	require.NoError(t, err, "proposal status did not change to passed in expected number of blocks")

	height, err = xion.Height(ctx)
	require.NoError(t, err, "error fetching height before submit upgrade proposal")
	err = testutil.WaitForBlocks(ctx, int(height+4), xion)
	return err
}
