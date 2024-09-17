package integration_tests

import (
	"fmt"
	"os"
	"path"
	"testing"

	"cosmossdk.io/math"

	ibctest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

func TestXionTokenFactory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}))
	xion, ctx := td.xionChain, td.ctx

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := math.NewInt(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, xion) // TODO: add a second user
	// TODO: extract both users
	xionUser := users[0]
	uaddr := xionUser.FormattedAddress()

	xionUser2 := users[1]
	uaddr2 := xionUser2.FormattedAddress()

	currentHeight, _ := xion.Height(ctx)
	testutil.WaitForBlocks(ctx, int(currentHeight)+8, xion)

	t.Logf("created xion user %s", uaddr)
	t.Logf("created xion user 2 %s", uaddr2)

	tfDenom := CreateTokenFactoryDenom(t, ctx, xion, xionUser, "ictestdenom", fmt.Sprintf("0%s", xion.Config().Denom))
	t.Log("tfDenom", tfDenom)

	// mint
	MintTokenFactoryDenom(t, ctx, xion, xionUser, 100, tfDenom)
	t.Log("minted tfDenom to user")
	balance, err := xion.GetBalance(ctx, uaddr, tfDenom)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(100).Int64(), balance.Int64())

	// mint-to
	expectedMint := uint64(70)
	MintToTokenFactoryDenom(t, ctx, xion, xionUser, xionUser2, expectedMint, tfDenom)
	t.Log("minted tfDenom to user")
	balance, err = xion.GetBalance(ctx, uaddr2, tfDenom)
	require.NoError(t, err)
	require.Equal(t, balance.Int64(), int64(expectedMint), "balance not 70")

	fp, err := os.Getwd()
	require.NoError(t, err)
	codeID, err := xion.StoreContract(ctx, xionUser.FormattedAddress(), path.Join(fp,
		"integration_tests", "testdata", "contracts", "tokenfactory_core.wasm"))
	require.NoError(t, err)

	// This allows the uaddr here to mint tokens on behalf of the contract. Typically you only allow a contract here, but this is testing.
	coreInitMsg := fmt.Sprintf(`{"allowed_mint_addresses":["%s"],"existing_denoms":["%s"]}`, uaddr, tfDenom)
	coreTFContract, err := xion.InstantiateContract(ctx, xionUser.FormattedAddress(), codeID, coreInitMsg, true)
	require.NoError(t, err)

	// change admin to the contract
	TransferTokenFactoryAdmin(t, ctx, xion, xionUser, coreTFContract, tfDenom)

	// ensure the admin is the contract
	admin := GetTokenFactoryAdmin(t, ctx, xion, tfDenom)
	t.Log("admin", admin)
	if admin != coreTFContract {
		t.Fatal("admin not coreTFContract. Did not properly transfer.")
	}

	// Mint on the contract for the user to ensure mint bindings work.
	mintMsg := fmt.Sprintf(`{"mint":{"address":"%s","denom":[{"denom":"%s","amount":"31"}]}}`, uaddr2, tfDenom)
	_, err = xion.ExecuteContract(ctx, xionUser.FormattedAddress(), coreTFContract, mintMsg)
	require.NoError(t, err)

	// ensure uaddr2 has 31+70 = 101
	balance, err = xion.GetBalance(ctx, uaddr2, tfDenom)
	require.NoError(t, err)
	fmt.Println(balance)
	require.Equal(t, balance.Int64(), int64(101))
}
