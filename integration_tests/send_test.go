package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/icza/dyno"

	xiontypes "github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func BuildXionChain(t *testing.T) (*cosmos.CosmosChain, context.Context) {
	ctx := context.Background()

	var numFullNodes = 1
	var numValidators = 3

	// pulling image from env to foster local dev
	imageTag := os.Getenv("XION_IMAGE")
	imageTagComponents := strings.Split(imageTag, ":")

	// Chain factory
	cf := ibctest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*ibctest.ChainSpec{
		{
			Name:    imageTagComponents[0],
			Version: imageTagComponents[1],
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: imageTagComponents[0],
						Version:    imageTagComponents[1],
						UidGid:     "1025:1025",
					},
				},
				GasPrices:              "0.0uxion",
				GasAdjustment:          1.3,
				Type:                   "cosmos",
				ChainID:                "xion-1",
				Bin:                    "xiond",
				Bech32Prefix:           "xion",
				Denom:                  "uxion",
				TrustingPeriod:         "336h",
				ModifyGenesis:          modifyGenesisShortProposals(votingPeriod, maxDepositPeriod),
				UsingNewGenesisCommand: true,
			},
			NumValidators: &numValidators,
			NumFullNodes:  &numFullNodes,
		},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	xion := chains[0].(*cosmos.CosmosChain)

	// Relayer Factory
	client, network := ibctest.DockerSetup(t)
	//relayer := ibctest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).Build(
	//	t, client, network)

	// Prep Interchain
	// const ibcPath = "xion-osmo-dungeon-test"
	ic := ibctest.NewInterchain().
		AddChain(xion)
	//AddRelayer(relayer, "relayer").
	//AddLink(ibctest.InterchainLink{
	//	Chain1:  xion,
	//	Chain2:  osmosis,
	//	Relayer: relayer,
	//	Path:    ibcPath,
	//})

	// Log location
	f, err := ibctest.CreateLogFile(fmt.Sprintf("%d.json", time.Now().Unix()))
	require.NoError(t, err)
	// Reporter/logs
	rep := testreporter.NewReporter(f)
	eRep := rep.RelayerExecReporter(t)

	// Build Interchain
	require.NoError(t, ic.Build(ctx, eRep, ibctest.InterchainBuildOptions{
		TestName:          t.Name(),
		Client:            client,
		NetworkID:         network,
		BlockDatabaseFile: ibctest.DefaultBlockDatabaseFilepath(),

		SkipPathCreation: false},
	),
	)
	return xion, ctx
}

func TestXionSendPlatformFee(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	xion, ctx := BuildXionChain(t)

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := int64(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]
	t.Logf("created xion user %s", xionUser.FormattedAddress())

	xionUserBalInitial, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, fundAmount, xionUserBalInitial)

	// step 1: send a xion message with default (0%) platform fee
	recipientKeyName := "recipient-key"
	err = xion.CreateKey(ctx, recipientKeyName)
	require.NoError(t, err)
	receipientKeyAddressBytes, err := xion.GetAddress(ctx, recipientKeyName)
	require.NoError(t, err)
	recipientKeyAddress, err := types.Bech32ifyAddressBytes(xion.Config().Bech32Prefix, receipientKeyAddressBytes)
	require.NoError(t, err)

	_, err = xion.FullNodes[0].ExecTx(ctx,
		xionUser.KeyName(),
		"xion", "send", xionUser.KeyName(),
		"--chain-id", xion.Config().ChainID,
		recipientKeyAddress, fmt.Sprintf("%d%s", 100, xion.Config().Denom),
	)
	require.NoError(t, err)
	balance, err := xion.GetBalance(ctx, recipientKeyAddress, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, uint64(100), uint64(balance))

	// step 2: update the platform percentage to 5%
	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	setPlatformPercentageMsg := xiontypes.MsgSetPlatformPercentage{
		Authority:          authtypes.NewModuleAddress("gov").String(),
		PlatformPercentage: 500,
	}

	xion.Config().EncodingConfig.InterfaceRegistry.RegisterImplementations(
		(*types.Msg)(nil),
		&xiontypes.MsgSetPlatformPercentage{},
	)
	cdc := codec.NewProtoCodec(xion.Config().EncodingConfig.InterfaceRegistry)

	msg, err := cdc.MarshalInterfaceJSON(&setPlatformPercentageMsg)
	require.NoError(t, err)

	prop := cosmos.Proposal{
		Messages: []json.RawMessage{msg},
		Metadata: "",
		Deposit:  "100uxion",
		Title:    "Set platform percentage to 5%",
		Summary:  "Ups the platform fee to 5% for the integration test",
	}
	paramChangeTx, err := xion.SubmitProposal(ctx, xionUser.KeyName(), &prop)
	require.NoError(t, err)
	t.Logf("Platform percentage change proposal submitted with ID %s in transaction %s", paramChangeTx.ProposalID, paramChangeTx.TxHash)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := xion.QueryProposal(ctx, paramChangeTx.ProposalID)
		if err != nil {
			require.NoError(t, err)
		} else {
			if proposalInfo.Status == cosmos.ProposalStatusVotingPeriod {
				return true
			}
			t.Logf("Waiting for proposal to enter voting status VOTING, current status: %s", proposalInfo.Status)
		}
		return false
	}, time.Second*11, time.Second, "failed to reach status VOTING after 11s")

	err = xion.VoteOnProposalAllValidators(ctx, paramChangeTx.ProposalID, cosmos.ProposalVoteYes)
	require.NoError(t, err)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := xion.QueryProposal(ctx, paramChangeTx.ProposalID)
		if err != nil {
			require.NoError(t, err)
		} else {
			if proposalInfo.Status == cosmos.ProposalStatusPassed {
				return true
			}
			t.Logf("Waiting for proposal to enter voting status PASSED, current status: %s", proposalInfo.Status)
		}
		return false
	}, time.Second*11, time.Second, "failed to reach status PASSED after 11s")

	// step 3: transfer and verify platform fees is extracted
	initialSendingBalance, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	initialReceivingBalance, err := xion.GetBalance(ctx, recipientKeyAddress, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, uint64(100), uint64(initialReceivingBalance))

	_, err = xion.FullNodes[0].ExecTx(ctx,
		xionUser.KeyName(),
		"xion", "send", xionUser.KeyName(),
		"--chain-id", xion.Config().ChainID,
		recipientKeyAddress, fmt.Sprintf("%d%s", 200, xion.Config().Denom),
	)
	require.NoError(t, err)

	postSendingBalance, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, uint64(initialSendingBalance-200), uint64(postSendingBalance))
	postReceivingBalance, err := xion.GetBalance(ctx, recipientKeyAddress, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, uint64(290), uint64(postReceivingBalance))
}

func getTotalCoinSupplyInBank(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, denom string, blockHeight uint64) string {
	if blockHeight == 0 {
		blockHeight, _ = xion.Height(ctx)
		require.Greater(t, blockHeight, 0)
	}
	jsonRes := make(map[string]interface{})
	queryRes, _, err := xion.FullNodes[0].ExecQuery(ctx, "bank", "total", "--height", strconv.FormatInt(int64(blockHeight), 10))
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(queryRes, &jsonRes))

	// Presuming we are the only denom on the chain
	totalSupply, err := dyno.GetSlice(jsonRes, "supply")
	require.NoError(t, err)
	xionCoin := totalSupply[0]
	require.NotEmpty(t, xionCoin)
	// Make sure we selected the uxion denom
	xionCoinDenom, err := dyno.GetString(xionCoin, "denom")
	require.NoError(t, err)
	require.Equal(t, xionCoinDenom, denom)
	initialXionSupply, err := dyno.GetString(xionCoin, "amount")
	require.NoError(t, err)
	return initialXionSupply
}

func getAddressBankBalanceAtHeight(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, address string, denom string, blockHeight uint64) string {
	if blockHeight == 0 {
		blockHeight, _ = xion.Height(ctx)
		require.Greater(t, blockHeight, 0)
	}
	jsonRes := make(map[string]interface{})
	queryRes, _, err := xion.FullNodes[0].ExecQuery(ctx, "bank", "balances", address, "--height", strconv.FormatInt(int64(blockHeight), 10))
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(queryRes, &jsonRes))

	balances, err := dyno.GetSlice(jsonRes, "balances")
	require.NoError(t, err)
	if len(balances) == 0 {
		return "0"
	}
	// Make sure we selected the uxion denom
	balanceDenom, err := dyno.GetString(balances[0], "denom")
	require.NoError(t, err)
	require.Equal(t, balanceDenom, denom)
	balance, err := dyno.GetString(balances[0], "amount")
	require.NoError(t, err)
	t.Logf("Balance for address %s: %s", address, balance)
	return balance
}

func GetModuleAddress(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, moduleName string) string {
	jsonRes := make(map[string]interface{})
	queryRes, _, err := xion.FullNodes[0].ExecQuery(ctx, "auth", "module-account", moduleName)
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(queryRes, &jsonRes))

	moduleAddress, err := dyno.GetString(jsonRes, "account", "base_account", "address")
	require.NoError(t, err)
	t.Logf("%s module address: %s", moduleName, moduleAddress)
	return moduleAddress
}

// This test confirms the property of the module described at
// https://www.notion.so/burntlabs/Mint-Module-Blog-Post-78f59fb108c04e9ea5fa826dda30a340
// Chain must have at least 12 blocks
func MintTestHarness(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context) {

	// We pick a random block height and 10 contiguous blocks from that height
	// and then test the property over these blocks

	currentBlockHeight, err := xion.Height(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, currentBlockHeight, uint64(12))
	// Get a random number from 1 to the (currentBlockHeight - 10)
	randomHeight := rand.Intn(int(currentBlockHeight)-11) + 2

	for i := randomHeight; i < randomHeight+10; i++ {
		t.Logf("Current random height: %d", randomHeight)
		// Get bank supply at previous height
		previousXionBankSupply, err := strconv.ParseUint(getTotalCoinSupplyInBank(t, xion, ctx, xion.Config().Denom, uint64(randomHeight-1)), 10, 64)
		t.Logf("Previous Xion bank supply: %d", previousXionBankSupply)
		require.NoError(t, err, "bank supply should be convertible to an int64")
		// Get bank supply at current height
		currentXionBankSupply, err := strconv.ParseUint(getTotalCoinSupplyInBank(t, xion, ctx, xion.Config().Denom, uint64(randomHeight)), 10, 64)
		t.Logf("Current Xion bank supply: %d", currentXionBankSupply)
		require.NoError(t, err, "bank supply should be convertible to an int64")
		tokenChange := currentXionBankSupply - previousXionBankSupply

		// Get the distribution module account address
		distributionModuleAddress := GetModuleAddress(t, xion, ctx, "distribution")
		// Get distribution module account balance in previous height
		previousDistributionModuleBalance, err := xion.GetBalance(ctx, distributionModuleAddress, xion.Config().Denom)
		require.NoError(t, err, "distribution module balance should be convertible to an int64")
		// Get distribution module account balance in current height
		currentDistributionModuleBalance, err := strconv.ParseUint(getAddressBankBalanceAtHeight(t, xion, ctx, distributionModuleAddress, xion.Config().Denom, uint64(randomHeight)), 10, 64)
		require.NoError(t, err, "distribution module balance should be convertible to an int64")

		delta := currentDistributionModuleBalance - uint64(previousDistributionModuleBalance)

		feesAccrued := delta - tokenChange

		// Query the current block provision
		var annualProvision json.Number
		queryRes, _, err := xion.FullNodes[0].ExecQuery(ctx, "mint", "annual-provisions", "--height", strconv.FormatInt(int64(randomHeight), 10))
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(queryRes, &annualProvision))
		// Query the block per year
		var params = make(map[string]interface{})
		queryRes, _, err = xion.FullNodes[0].ExecQuery(ctx, "mint", "params")
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(queryRes, &params))
		blocksPerYear, err := dyno.GetInteger(params, "blocks_per_year")
		require.NoError(t, err)
		// Calculate the block provision
		blockProvision := math.LegacyMustNewDecFromStr(annualProvision.String()).QuoInt(math.NewInt(blocksPerYear)) // This ideally is the minted tokens for the block

		// Make sure the minted tokens is equal to the block provision - fees accrued
		if blockProvision.TruncateInt().GT(math.NewIntFromUint64(feesAccrued)) {
			// We have minted tokens
			mintedTokens := blockProvision.TruncateInt().Sub(math.NewIntFromUint64(feesAccrued))
			require.Equal(t, mintedTokens, math.NewInt(int64(tokenChange)))
		} else if blockProvision.TruncateInt().LT(math.NewIntFromUint64(feesAccrued)) {
			// We have burned tokens
			burnedTokens := math.NewIntFromUint64(feesAccrued).Sub(blockProvision.TruncateInt())
			require.Equal(t, burnedTokens, math.NewInt(int64(tokenChange)))
		} else {
			// We have not minted or burned tokens
			require.Equal(t, math.NewInt(0), math.NewInt(int64(tokenChange)))
		}
	}
}

func TestMintModuleNoInflationNoFees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	xion, ctx := BuildXionChain(t)

	// Wait for some blocks and check if that supply stays the same
	chainHeight, _ := xion.Height(ctx)
	testutil.WaitForBlocks(ctx, int(chainHeight)+10, xion)

	// Run test harness
	MintTestHarness(t, xion, ctx)
}

func TestMintModuleInflationNoFees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	xion, ctx := BuildXionChain(t)

	// Wait for some blocks and check if that supply stays the same
	chainHeight, _ := xion.Height(ctx)
	testutil.WaitForBlocks(ctx, int(chainHeight)+10, xion)

	// Run test harness
	MintTestHarness(t, xion, ctx)
}
