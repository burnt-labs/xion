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
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
)

const (
	inflationMin        = "0.0"
	inflationMax        = "0.0"
	inflationRateChange = "0.0"
)

func TestMintModuleNoInflationNoFees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	xion, ctx := BuildXionChain(t, ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisInflation}, [][]string{{inflationMin, inflationMax, inflationRateChange}}))

	// Wait for some blocks and check if that supply stays the same
	chainHeight, _ := xion.Height(ctx)
	testutil.WaitForBlocks(ctx, int(chainHeight)+10, xion)

	// Run test harness
	VerifyMintModuleTestRandomBlocks(t, xion, ctx)
}

func TestMintModuleInflationNoFees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	xion, ctx := BuildXionChain(t, ModifyInterChainGenesis(ModifyInterChainGenesisFn{}, [][]string{{}}))

	chainHeight, _ := xion.Height(ctx)
	testutil.WaitForBlocks(ctx, int(chainHeight)+10, xion)

	// Run test harness
	VerifyMintModuleTestRandomBlocks(t, xion, ctx)
}

// TxCommand is a helper to retrieve a full command for broadcasting a tx
// with the chain node binary.
func feeTxCommand(chain *cosmos.CosmosChain, fee string, address ...string) []string {
	command := []string{"tx"}
	return chain.FullNodes[0].NodeCommand(append(command,
		"bank", "send", address[0], address[1],
		fmt.Sprintf("%s%s", "1000000", chain.Config().Denom),
		"--from", "faucet",
		"--fees", fmt.Sprintf("%s%s", fee, chain.Config().Denom),
		"--chain-id", chain.Config().ChainID,
		"--keyring-backend", keyring.BackendTest,
		"--output", "json",
		"-y",
	)...)
}

// Send bank send txs at duration intervals
func sendPeriodicBankTx(t *testing.T, chain *cosmos.CosmosChain, ctx context.Context, chainHeight uint64, duration int, txHashes *MintModuleTest, feeTypeHigh bool) {
	// Create a test wallet to send funds to
	err := chain.CreateKey(ctx, "testAccount")
	require.NoError(t, err)
	testAccountAddress, err := chain.FullNodes[0].AccountKeyBech32(ctx, "testAccount")
	require.NoError(t, err)
	faucet, err := chain.FullNodes[0].AccountKeyBech32(ctx, "faucet")
	require.NoError(t, err)
	curHeight, _ := chain.Height(ctx)

	for curHeight < chainHeight {
		// Get the current block provision
		blockProvisionDec := GetBlockAnnualProvision(t, chain, ctx, chain.Config().Denom, curHeight) // This the minted amount for the current block
		var blockProvision math.Int
		if feeTypeHigh {
			// we are testing high fees
			blockProvision = blockProvisionDec.TruncateInt().Mul(math.NewInt(2))
		} else {
			// we are testing low fees
			blockProvision = blockProvisionDec.TruncateInt().Quo(math.NewInt(2))
		}
		// We send a fee that is 2x the block provision since the inflation rate cannot increase that much
		stdout, ee, err := chain.Exec(ctx, feeTxCommand(chain, blockProvision.String(), faucet, testAccountAddress), nil)
		if err != nil {
			t.Log(string(stdout))
			t.Log(string(ee))
			t.Fatal(err)
			break
		}

		output := cosmos.CosmosTx{}
		err = json.Unmarshal([]byte(stdout), &output)
		if err != nil {
			t.Fatal(err)
			break
		}

		txHashes.TxHashes = append(txHashes.TxHashes, output.TxHash)

		time.Sleep(time.Duration(duration) * time.Second)
		curHeight, _ = chain.Height(ctx)
	}
}

type MintModuleTest struct {
	TxHashes []string
}

func TestMintModuleInflationHighFees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	xion, ctx := BuildXionChain(t, ModifyInterChainGenesis(ModifyInterChainGenesisFn{}, [][]string{{}}))

	txHashes := MintModuleTest{
		TxHashes: []string{},
	}
	var mu sync.Mutex

	mu.Lock()
	go func(t *testing.T, chain *cosmos.CosmosChain, ctx context.Context, blockHeight uint64, duration int, txHashes *MintModuleTest) {
		sendPeriodicBankTx(t, xion, ctx, blockHeight, duration, txHashes, true)
	}(t, xion, ctx, 20, 4, &txHashes)
	mu.Unlock()

	// Wait some blocks
	testutil.WaitForBlocks(ctx, 25, xion)

	require.NotEmpty(t, txHashes.TxHashes)
	// Run test harness
	VerifyMintModuleTest(t, xion, ctx, txHashes.TxHashes)
}

func TestMintModuleInflationLowFees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	xion, ctx := BuildXionChain(t, ModifyInterChainGenesis(ModifyInterChainGenesisFn{}, [][]string{{}}))

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
	// Run test harness
	VerifyMintModuleTest(t, xion, ctx, txHashes.TxHashes)
}
