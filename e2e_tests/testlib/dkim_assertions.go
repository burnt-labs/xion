package testlib

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	dkimTypes "github.com/burnt-labs/xion/x/dkim/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govModule "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/stretchr/testify/require"
)

// DKIMTestData contains the test data for DKIM assertions.
type DKIMTestData struct {
	PubKey1   string
	PubKey2   string
	PubKey3   string
	Domain1   string
	Domain2   string
	Selector1 string
	Selector2 string
	Selector3 string
}

// DefaultDKIMTestData returns the standard test data used in DKIM tests.
func DefaultDKIMTestData() DKIMTestData {
	return DKIMTestData{
		PubKey1:   "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB",
		PubKey2:   "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB",
		PubKey3:   "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEApYmNCWAKIxf5uOEXIdEBPRDmMxcyiAnpDT3/xHad1n/d1yeZLhxCEOV6IeMNOIHD9p+VxqqzmFCvWkKvisBauAMxoJ0so7JHfjP3BOUb7hKOvcU4XiwyjhjMJQMNBImlB75Es04Kfu9RrC9tOFau5lN4ldjvNUjQH3YZoknK+LyXtJ8XBUrKdd4pptlzhMb3/J5q2wlHgUC0+jZUKtjCLHoHhQv7+vXdM2gZmPlmr5fofyAyPMPLdO5e65BXC2Z9kmSl1Zw3b41i9RlC8OwAyloI0Za/hzqQ/0sre9KtCNoPCtLhF/03dccG/282WkWCWVRxEBEC1q6s99GYm7SMqQIDAQAB",
		Domain1:   "x.com",
		Domain2:   "xion.com",
		Selector1: "dkim202406",
		Selector2: "dkim202407",
		Selector3: "google",
	}
}

// DKIMAssertionConfig contains configuration for running DKIM assertions.
type DKIMAssertionConfig struct {
	Chain           *cosmos.CosmosChain
	Ctx             context.Context
	User            ibc.Wallet       // Optional: if nil, will create and fund a new user
	ProposalTracker *ProposalTracker // Required: tracks proposal IDs
	TestData        DKIMTestData
}

// RunDKIMModuleAssertions seeds DKIM records via governance and validates queries.
// This is extracted from TestDKIMModule in dkim_test.go.
func RunDKIMModuleAssertions(t *testing.T, cfg DKIMAssertionConfig) {
	t.Log("Running DKIM module assertions")

	ctx := cfg.Ctx
	xion := cfg.Chain
	testData := cfg.TestData

	// Set up bech32 config
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// Get or create user
	var chainUser ibc.Wallet
	if cfg.User != nil {
		chainUser = cfg.User
	} else {
		fundAmount := math.NewInt(10_000_000_000)
		users := interchaintest.GetAndFundTestUsers(t, ctx, "dkim-test", fundAmount, xion)
		chainUser = users[0]
	}

	govModAddress := GetModuleAddress(t, xion, ctx, govModule.ModuleName)

	// Compute poseidon hashes
	hash1, err := dkimTypes.ComputePoseidonHash(testData.PubKey1)
	require.NoError(t, err)
	hash3, err := dkimTypes.ComputePoseidonHash(testData.PubKey3)
	require.NoError(t, err)

	// Create seed records
	seedRecords := []dkimTypes.DkimPubKey{
		{
			Domain:       testData.Domain1,
			Selector:     testData.Selector1,
			PubKey:       testData.PubKey1,
			PoseidonHash: []byte(hash1.String()),
		},
		{
			Domain:       testData.Domain1,
			Selector:     testData.Selector3,
			PubKey:       testData.PubKey3,
			PoseidonHash: []byte(hash3.String()),
		},
	}

	createDkimMsg := &dkimTypes.MsgAddDkimPubKeys{
		Authority:   govModAddress,
		DkimPubkeys: seedRecords,
	}
	require.NoError(t, createDkimMsg.ValidateBasic())

	// Submit proposal to seed DKIM records
	proposalID := cfg.ProposalTracker.NextID()
	err = SubmitAndPassProposal(t, ctx, xion, chainUser,
		[]cosmos.ProtoMessage{createDkimMsg},
		"Seed DKIM records", "Seed DKIM records", "Seed DKIM records",
		proposalID)
	require.NoError(t, err)

	// Verify queries work
	t.Log("Verifying DKIM queries...")

	// Query single DKIM record by domain and selector
	dkimRecord, err := ExecQuery(t, ctx, xion.GetNode(), "dkim", "dkim-pubkey", testData.Domain1, testData.Selector1)
	require.NoError(t, err)
	require.Equal(t, testData.PubKey1, dkimRecord["dkim_pub_key"].(map[string]interface{})["pub_key"].(string))

	// Query all records for domain
	allDkimRecords, err := ExecQuery(t, ctx, xion.GetNode(), "dkim", "qdkims", "--domain", testData.Domain1)
	require.NoError(t, err)
	require.Len(t, allDkimRecords["dkim_pub_keys"].([]interface{}), 2)

	// Query by domain + poseidon hash pair
	allDkimRecords, err = ExecQuery(t, ctx, xion.GetNode(), "dkim", "qdkims", "--domain", testData.Domain1, "--hash", hash3.String())
	require.NoError(t, err)
	require.Len(t, allDkimRecords["dkim_pub_keys"].([]interface{}), 1)
	require.Equal(t, testData.Selector3, allDkimRecords["dkim_pub_keys"].([]interface{})[0].(map[string]interface{})["selector"])

	t.Log("DKIM module assertions completed successfully")
}

// RunDKIMGovernanceAssertions tests add/remove of DKIM records via governance.
// This is extracted from TestDKIMGovernance in dkim_test.go.
func RunDKIMGovernanceAssertions(t *testing.T, cfg DKIMAssertionConfig) {
	t.Log("Running DKIM governance assertions")

	ctx := cfg.Ctx
	xion := cfg.Chain
	testData := cfg.TestData

	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// Get or create user
	var chainUser ibc.Wallet
	if cfg.User != nil {
		chainUser = cfg.User
	} else {
		fundAmount := math.NewInt(10_000_000_000)
		users := interchaintest.GetAndFundTestUsers(t, ctx, "dkim-gov-test", fundAmount, xion)
		chainUser = users[0]
	}

	govModAddress := GetModuleAddress(t, xion, ctx, govModule.ModuleName)

	// Test add via governance
	testDomain := "test.example.com"
	testSelector := "governance_test"

	hash, err := dkimTypes.ComputePoseidonHash(testData.PubKey1)
	require.NoError(t, err)

	governancePubKeys := []dkimTypes.DkimPubKey{
		{
			Domain:       testDomain,
			Selector:     testSelector,
			PubKey:       testData.PubKey1,
			PoseidonHash: []byte(hash.String()),
		},
	}

	createDkimMsg := &dkimTypes.MsgAddDkimPubKeys{
		Authority:   govModAddress,
		DkimPubkeys: governancePubKeys,
	}
	require.NoError(t, createDkimMsg.ValidateBasic())

	// Submit proposal to add
	addProposalID := cfg.ProposalTracker.NextID()
	err = SubmitAndPassProposal(t, ctx, xion, chainUser,
		[]cosmos.ProtoMessage{createDkimMsg},
		"Add test DKIM record", "Add test DKIM record", "Add test DKIM record",
		addProposalID)
	require.NoError(t, err)

	// Verify record was added
	dkimRecord, err := ExecQuery(t, ctx, xion.GetNode(), "dkim", "dkim-pubkey", testDomain, testSelector)
	require.NoError(t, err)
	require.Equal(t, testData.PubKey1, dkimRecord["dkim_pub_key"].(map[string]interface{})["pub_key"].(string))

	// Submit proposal to remove
	deleteDkimMsg := &dkimTypes.MsgRemoveDkimPubKey{
		Authority: govModAddress,
		Domain:    testDomain,
		Selector:  testSelector,
	}

	removeProposalID := cfg.ProposalTracker.NextID()
	err = SubmitAndPassProposal(t, ctx, xion, chainUser,
		[]cosmos.ProtoMessage{deleteDkimMsg},
		"Remove test DKIM record", "Remove test DKIM record", "Remove test DKIM record",
		removeProposalID)
	require.NoError(t, err)

	// Verify record was removed
	_, err = ExecQuery(t, ctx, xion.GetNode(), "dkim", "dkim-pubkey", testDomain, testSelector)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")

	t.Log("DKIM governance assertions completed successfully")
}
