package integration_tests

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/math"
	dkimTypes "github.com/burnt-labs/xion/x/dkim/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govModule "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
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
	customDomain   = "account.netflix.com"
	customSelector = "kk6c473czcop4fqv6yhfgiqupmfz3cm2"
)

var pubKeysBz, _ = json.Marshal([]Dkim{{
	PubKey:       pubKey_1,
	Domain:       domain_1,
	Selector:     selector_1,
	PoseidonHash: base64.StdEncoding.EncodeToString([]byte(poseidon_hash_1)),
}, {
	PubKey:       pubKey_2,
	Domain:       domain_2,
	Selector:     selector_2,
	PoseidonHash: base64.StdEncoding.EncodeToString([]byte(poseidon_hash_2)),
}, {
	PubKey:       pubKey_3,
	Domain:       domain_1,
	Selector:     selector_3,
	PoseidonHash: base64.StdEncoding.EncodeToString([]byte(poseidon_hash_3)),
}})

func TestDKIMModule(t *testing.T) {
	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisDKIMRecords, ModifyGenesisShortProposals}, [][]string{{string(pubKeysBz)}, {votingPeriod, maxDepositPeriod}}))

	xion, ctx := td.xionChain, td.ctx

	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")
	xion.Config().EncodingConfig.InterfaceRegistry.RegisterImplementations((*sdk.Msg)(nil), &dkimTypes.MsgAddDkimPubKeys{})
	xion.Config().EncodingConfig.InterfaceRegistry.RegisterImplementations((*sdk.Msg)(nil), &dkimTypes.MsgRemoveDkimPubKey{})

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	chainUser := users[0]
	govModAddress := GetModuleAddress(t, xion, ctx, govModule.ModuleName)

	// query chain for DKIM records
	dkimRecord, err := ExecQuery(t, ctx, xion.GetNode(), "dkim", "dkim-pubkey", domain_1, selector_1)
	require.NoError(t, err)
	require.Equal(t, dkimRecord["dkim_pub_key"].(map[string]interface{})["pub_key"].(string), pubKey_1)

	// query for all records of x.com
	allDkimRecords, err := ExecQuery(t, ctx, xion.GetNode(), "dkim", "qdkims", "--domain", domain_1)
	require.NoError(t, err)
	require.Len(t, allDkimRecords["dkim_pub_keys"].([]interface{}), 2)

	// query for a domain+poseidon hash pair matching domain_1 selector_3
	allDkimRecords, err = ExecQuery(t, ctx, xion.GetNode(), "dkim", "qdkims", "--domain", domain_1, "--hash", poseidon_hash_3)
	require.NoError(t, err)
	require.Len(t, allDkimRecords["dkim_pub_keys"].([]interface{}), 1)
	require.Equal(t, allDkimRecords["dkim_pub_keys"].([]interface{})[0].(map[string]interface{})["selector"], selector_3)

	// generate a dkim record by querying the chain
	// and then submit a proposal to add it
	dkimRecord, err = ExecQuery(t, ctx, xion.GetNode(), "dkim", "gdkim", customDomain, customSelector)
	require.NoError(t, err)

	customDkimPubkey := dkimRecord["pub_key"].(string)
	poseidonHash, err := base64.StdEncoding.DecodeString(dkimRecord["poseidon_hash"].(string))
	require.NoError(t, err)

	governancePubKeys := []dkimTypes.DkimPubKey{
		{
			Domain:       customDomain,
			Selector:     customSelector,
			PubKey:       customDkimPubkey,
			PoseidonHash: poseidonHash,
		},
	}

	createDkimMsg := dkimTypes.NewMsgAddDkimPubKeys(sdk.MustAccAddressFromBech32(govModAddress), governancePubKeys)
	require.NoError(t, createDkimMsg.ValidateBasic())

	err = createAndSubmitProposal(t, xion, ctx, chainUser, []cosmos.ProtoMessage{createDkimMsg}, "Add Netflix DKIM record", "Add Netflix DKIM record", "Add Netflix DKIM record", 1)
	require.NoError(t, err)

	// proposal must have gone through and msg submitted; let's query the chain for the pubkey
	dkimRecord, err = ExecQuery(t, ctx, xion.GetNode(), "dkim", "dkim-pubkey", customDomain, customSelector)
	require.NoError(t, err)
	require.Equal(t, dkimRecord["dkim_pub_key"].(map[string]interface{})["pub_key"].(string), customDkimPubkey)

	deleteDkimMsg := dkimTypes.NewMsgRemoveDkimPubKey(sdk.MustAccAddressFromBech32(govModAddress), dkimTypes.DkimPubKey{
		Domain:   customDomain,
		Selector: customSelector,
	})

	err = createAndSubmitProposal(t, xion, ctx, chainUser, []cosmos.ProtoMessage{deleteDkimMsg}, "Remove Netflix DKIM record", "Remove Netflix DKIM record", "Remove Netflix DKIM record", 2)
	require.NoError(t, err)

	// proposal must have gone through and msg submitted; let's query the chain for the pubkey
	_, err = ExecQuery(t, ctx, xion.GetNode(), "dkim", "dkim-pubkey", customDomain, customSelector)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func createAndSubmitProposal(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, proposer ibc.Wallet, proposalMsgs []cosmos.ProtoMessage, title, summary, metadata string, proposalId int) error {
	proposal, err := xion.BuildProposal(
		proposalMsgs,
		title,
		summary,
		metadata,
		"500000000"+xion.Config().Denom, // greater than min deposit",
		proposer.FormattedAddress(),
		false,
	)
	require.NoError(t, err)

	height, err := xion.Height(ctx)
	require.NoError(t, err, "error fetching height before submit upgrade proposal")

	_, err = xion.SubmitProposal(ctx, proposer.KeyName(), proposal)
	require.NoError(t, err) // only governance account can submit proposals

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
