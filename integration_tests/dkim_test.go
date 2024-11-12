package integration_tests

import (
	"encoding/json"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	dkimTypes "github.com/burnt-labs/xion/x/dkim/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govModule "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

const pubKey_1 = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"
const pubKey_2 = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"

const domain_1 = "x.com"
const domain_2 = "xion.com"

const selector_1 = "dkim202406"
const selector_2 = "dkim202407"

const poseidon_hash_1 = "1983664618407009423875829639306275185491946247764487749439145140682408188330"
const poseidon_hash_2 = "1983664618407009423875829639306275185491946247764487749439145140682408188330"

var pubKeysBz, _ = json.Marshal([]Dkim{{
	PubKey:       pubKey_1,
	Domain:       domain_1,
	Selector:     selector_1,
	PoseidonHash: poseidon_hash_1,
}, {
	PubKey:       pubKey_2,
	Domain:       domain_2,
	Selector:     selector_2,
	PoseidonHash: poseidon_hash_2,
}})

func TestDKIMModule(t *testing.T) {
	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisDKIMRecords}, [][]string{{string(pubKeysBz)}}))

	xion, ctx := td.xionChain, td.ctx

	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")
	xion.Config().EncodingConfig.InterfaceRegistry.RegisterImplementations((*sdk.Msg)(nil), &dkimTypes.MsgAddDkimPubKeys{})

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	chainUser := users[0]
	govModAddress := GetModuleAddress(t, xion, ctx, govModule.ModuleName)

	// query chain for DKIM records
	dkimRecord, err := ExecQuery(t, ctx, xion.GetNode(), "dkim", "dkim-pubkey", "--domain", domain_1, "--selector", selector_1)
	require.NoError(t, err)
	require.Equal(t, dkimRecord["dkim_pubkey"].(map[string]interface{})["pub_key"].(string), pubKey_1)

	address, err := xion.GetAddress(ctx, chainUser.KeyName())
	require.NoError(t, err)

	addrString, err := sdk.Bech32ifyAddressBytes(xion.Config().Bech32Prefix, address)
	require.NoError(t, err)

	governancePubkey_1 := "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCD8gKP5B1x0stqA0NhBw0PbvVjbQ98s07tAovJmUBLk9D/VsjNCVx8WAzZxyKI+lbs9Okua/Knq5kDzO2dxSbus/LaDHCHx7YYqNWL0xdaPCSjFL/sYqX7V4wq4N/OcBoASitk61eGJXVgmEfJBRNfNoi3iHDf9GvpCNBKTHYkewIDAQAB"
	poseidonHash, err := dkimTypes.ComputePoseidonHash(governancePubkey_1)
	governanceDomain := "account.netflix.com"
	governanceSelector := "kk6c473czcop4fqv6yhfgiqupmfz3cm2"
	require.NoError(t, err, "error computing governance public key poseidon hash")

	governancePubKeys := []dkimTypes.DkimPubKey{
		{
			Domain:       governanceDomain,
			Selector:     governanceSelector,
			PubKey:       governancePubkey_1,
			PoseidonHash: poseidonHash.Bytes(),
		},
	}

	createDkimMsg := dkimTypes.NewMsgAddDkimPubKeys(sdk.MustAccAddressFromBech32(govModAddress), governancePubKeys)

	proposal, err := xion.BuildProposal(
		[]cosmos.ProtoMessage{createDkimMsg},
		"Add netflix DKIM public key",
		"Add netflix DKIM public key",
		"",
		"500000000"+xion.Config().Denom, // greater than min deposit
		addrString,
		false,
	)
	require.NoError(t, err)

	height, err := xion.Height(ctx)
	require.NoError(t, err, "error fetching height before submit upgrade proposal")

	_, err = xion.SubmitProposal(ctx, chainUser.KeyName(), proposal)
	require.NoError(t, err) // only governance account can submit proposals

	prop, err := xion.GovQueryProposal(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, govv1beta1.StatusVotingPeriod, prop.Status)

	err = xion.VoteOnProposalAllValidators(ctx, prop.ProposalId, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")

	err = testutil.WaitForBlocks(ctx, int(height+haltHeightDelta), xion)
	require.NoError(t, err)

	prop, err = xion.GovQueryProposal(ctx, 1)
	require.NoError(t, err)
	fmt.Println("Proposal status: ", prop)

	afterVoteHeight, err := xion.Height(ctx)
	require.NoError(t, err, "error fetching height after voting on proposal")

	_, err = cosmos.PollForProposalStatus(ctx, xion, height, afterVoteHeight, prop.ProposalId, govv1beta1.StatusPassed)
	require.NoError(t, err, "proposal status did not change to passed in expected number of blocks")

	height, err = xion.Height(ctx)
	require.NoError(t, err, "error fetching height before submit upgrade proposal")
	err = testutil.WaitForBlocks(ctx, int(height+4), xion)
	require.NoError(t, err)

	// proposal must have gone through and msg submitted; let's query the chain for the pubkey
	dkimRecord, err = ExecQuery(t, ctx, xion.GetNode(), "dkim", "dkim-pubkey", "--domain", governanceDomain, "--selector", governanceSelector)
	require.NoError(t, err)
	require.Equal(t, dkimRecord["dkim_pubkey"].(map[string]interface{})["pub_key"].(string), governancePubKeys)
}
