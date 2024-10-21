package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

const (
	inflationMin        = "0.0"
	inflationMax        = "0.0"
	inflationRateChange = "0.0"
)

// In this test case, the mint module inflation is set to 0 by setting the inflation rate
// and inflation max and min values to 0 in the genesis file. We then send a bunch of
// transactions without fees and check if the total supply stays the same.
// The supply should remain constants because no token would be minted and there are no
// tx fees to pay validators with. This is a base test case to ensure that the mint module
// is not minting tokens when it shouldn't.
func TestMintModuleNoInflationNoFees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals, ModifyGenesisInflation}, [][]string{{votingPeriod, maxDepositPeriod}, {minInflation, maxInflation, inflationRateChange}}))
	xion, ctx := td.xionChain, td.ctx

	// Wait for some blocks and check if that supply stays the same
	chainHeight, _ := xion.Height(ctx)
	testutil.WaitForBlocks(ctx, int(chainHeight)+10, xion)

	assertion := func(t *testing.T, provision math.LegacyDec, feesAccrued int64, tokenChange int64) {
		require.Equal(t, math.NewInt(0), math.NewInt(tokenChange))
		t.Logf("No minted or Burned tokens. Token change: %d", tokenChange)
	}

	// Run test harness
	VerifyMintModuleTestRandomBlocks(t, xion, ctx, assertion)
}

// In this test case, the mint module inflation is left to the default value in
// the genesis file. We then send a bunch of transactions without fees and check
// if the total supply increases. The supply should increase because the mint module
// is minting tokens to pay validators with.
func TestMintModuleInflationNoFees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}))
	xion, ctx := td.xionChain, td.ctx

	chainHeight, _ := xion.Height(ctx)
	testutil.WaitForBlocks(ctx, int(chainHeight)+10, xion)
	assertion := func(t *testing.T, provision math.LegacyDec, feesAccrued int64, tokenChange int64) {
		require.Truef(t, provision.TruncateInt().GT(math.NewInt(feesAccrued)), "provision should be greater if tokens where minted, provision: %s, fees accrued:%s", provision.TruncateInt(), feesAccrued)
		// We have minted tokens because the fees accrued is less than the block provision
		mintedTokens := provision.TruncateInt().Sub(math.NewInt(feesAccrued))
		t.Logf("Minted tokens: %d and Token change: %d", mintedTokens.Int64(), int64(tokenChange))
		require.Equal(t, mintedTokens, math.NewInt(int64(tokenChange)))
	}

	// Run test harness
	VerifyMintModuleTestRandomBlocks(t, xion, ctx, assertion)
}

// TxCommand is a helper to retrieve a full command for broadcasting a tx
// with the chain node binary.
func feeTxCommand(chain *cosmos.CosmosChain, fee string, sender string, receiver string) []string {
	command := []string{"tx"}
	return chain.GetNode().NodeCommand(append(command,
		"bank", "send", sender, receiver,
		fmt.Sprintf("%s%s", "1000000", chain.Config().Denom),
		"--from", "faucet",
		"--fees", fmt.Sprintf("%s%s", fee, chain.Config().Denom),
		"--chain-id", chain.Config().ChainID,
		"--keyring-backend", keyring.BackendTest,
		"--output", "json",
		"-y",
	)...)
}

/*
Send bank send txs at some intervals

	chainHeight is the block height at which to stop sending txs (instead of using a clock)
	duration is the time in seconds between each tx
	txHashes is a pointer to a struct that contains the tx hashes of the bank send txs
	feeTypeHigh is a boolean that determines whether the test is for high fees or low fees
	- High fees are 2x the previous block provision since the rate of change of inflation is << 2x block_provision. This way we can be sure that we accrue more fees than block provision
	- Low fees are 0.5x the previous block provision. This way we can be sure that we accrue less fees than block provision
*/
func sendPeriodicBankTx(t *testing.T, chain *cosmos.CosmosChain, ctx context.Context, chainHeight uint64, duration int, txHashes *MintModuleTest, feeTypeHigh bool) {
	// Create a test wallet to send funds to
	err := chain.CreateKey(ctx, "testAccount")
	require.NoError(t, err)
	// Retrieve the test wallet address
	testAccountAddress, err := chain.GetNode().AccountKeyBech32(ctx, "testAccount")
	require.NoError(t, err)
	// Retrieve the faucet address. Every chain has a faucet account
	faucet, err := chain.GetNode().AccountKeyBech32(ctx, "faucet")
	require.NoError(t, err)
	// Get the current chain height
	cHeight, err := chain.Height(ctx)
	require.NoError(t, err)
	curHeight := uint64(cHeight)

	for curHeight < chainHeight {
		// Get the current block provision at some height
		blockProvisionDec := GetBlockAnnualProvision(t, chain, ctx, chain.Config().Denom, curHeight) // This the minted token amount for the current block
		var blockProvision math.Int
		if feeTypeHigh {
			// we are testing high fees
			// We send a fee that is 2x the block provision since
			// the inflation rate cannot increase that much
			blockProvision = blockProvisionDec.TruncateInt().Mul(math.NewInt(2))
		} else {
			// we are testing low fees
			// We send a fee that is 0.5x the block provision since
			// the inflation rate cannot decrease that much
			blockProvision = blockProvisionDec.TruncateInt().Quo(math.NewInt(2))
		}

		stdout, ee, err := chain.Exec(ctx, feeTxCommand(chain, blockProvision.String(), faucet, testAccountAddress), nil)
		if err != nil {
			t.Log(string(stdout))
			t.Log(string(ee))
			t.Fatal(err)
		}

		output := cosmos.CosmosTx{}
		err = json.Unmarshal([]byte(stdout), &output)
		if err != nil {
			t.Fatal(err)
		}

		// Save the hash of the Send tx for later analysis
		txHashes.TxHashes = append(txHashes.TxHashes, output.TxHash)

		time.Sleep(time.Duration(duration) * time.Second)
		cHeight, err := chain.Height(ctx)
		require.NoError(t, err)
		curHeight = uint64(cHeight)
	}
}

type MintModuleTest struct {
	TxHashes []string
}

// Here we test the mint module by sending a bunch of transactions with extra high fees
// and checking if the total supply increases. We also check if the total supply
// increases by the correct amount.
func TestMintModuleInflationHighFees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	td := BuildXionChain(t, "0.00uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}))
	xion, ctx := td.xionChain, td.ctx

	txHashes := MintModuleTest{
		TxHashes: []string{},
	}

	go func(t *testing.T, chain *cosmos.CosmosChain, ctx context.Context, blockHeight uint64, duration int, txHashes *MintModuleTest) {
		sendPeriodicBankTx(t, xion, ctx, blockHeight, duration, txHashes, true)
	}(t, xion, ctx, 20, 4, &txHashes)

	// Wait some blocks
	testutil.WaitForBlocks(ctx, 25, xion)

	require.NotEmpty(t, txHashes.TxHashes)

	assertion := func(t *testing.T, provision math.LegacyDec, feesAccrued int64, tokenChange int64) {
		// We have burned tokens because the fees accrued is greater than the block provision so the fees
		// accrued are used to pay validators and the remaining is burned
		require.True(t, provision.TruncateInt().LT(math.NewInt(feesAccrued)), "provision should be lower, in order to burn tokens")
		burnedTokens := math.NewInt(feesAccrued).Sub(provision.TruncateInt())
		t.Logf("Burned tokens: %d and Token change: %d", burnedTokens.Int64(), tokenChange)
		require.Equal(t, burnedTokens, math.NewInt(tokenChange).Abs())
	}
	// Run test harness
	VerifyMintModuleTest(t, xion, ctx, txHashes.TxHashes, assertion)
}

// Here we test the mint module by sending a bunch of transactions with extra low fees
// and checking if the total supply increases. We also check if the total supply
// increases by the correct amount.
func TestMintModuleInflationLowFees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	td := BuildXionChain(t, "0.00uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}))
	xion, ctx := td.xionChain, td.ctx

	txHashes := MintModuleTest{
		TxHashes: []string{},
	}
	var mu sync.Mutex

	mu.Lock()
	go func(t *testing.T, chain *cosmos.CosmosChain, ctx context.Context, blockHeight uint64, duration int, txHashes *MintModuleTest) {
		sendPeriodicBankTx(t, xion, ctx, blockHeight, duration, txHashes, false)
	}(t, xion, ctx, 20, 4, &txHashes)
	mu.Unlock()

	// Wait some blocks
	testutil.WaitForBlocks(ctx, 25, xion)

	require.NotEmpty(t, txHashes.TxHashes)

	assertion := func(t *testing.T, provision math.LegacyDec, feesAccrued int64, tokenChange int64) {
		require.Truef(t, provision.TruncateInt().GT(math.NewInt(feesAccrued)), "provision should be greater if tokens where minted")
		// We have minted tokens because the fees accrued is less than the block provision
		mintedTokens := provision.TruncateInt().Sub(math.NewInt(feesAccrued))
		t.Logf("Minted tokens: %d and Token change: %d", mintedTokens.Int64(), int64(tokenChange))
		require.Equal(t, mintedTokens, math.NewInt(int64(tokenChange)))
	}
	// Run test harness
	VerifyMintModuleTest(t, xion, ctx, txHashes.TxHashes, assertion)
}
