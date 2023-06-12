package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	authTx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/icza/dyno"
	ibctest "github.com/strangelove-ventures/interchaintest/v7"

	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
)

/// This integration test would test the result of having multiple transactions sent from the same account in the same block.
/// The goal is to see the result of having one of those tx fail and the other succeed.
/// We want to see if the sequence number is incremented correctly.

type BlockTxTest struct {
	TxHashes []string
}

// TxCommand is a helper to retrieve a full command for broadcasting a tx
// with the chain node binary.
func txCommand(chain *cosmos.CosmosChain, sender string, receiver string) []string {
	command := []string{"tx"}
	return chain.FullNodes[0].NodeCommand(append(command,
		"bank", "send", sender, receiver,
		fmt.Sprintf("%s%s", "1000000", chain.Config().Denom),
		"--from", "faucet",
		"--chain-id", chain.Config().ChainID,
		"--keyring-backend", keyring.BackendTest,
		"--output", "json",
		"-y",
	)...)
}

func getAccountSequenceNumber(t *testing.T, chain *cosmos.CosmosChain, ctx context.Context, address string, height uint64) uint64 {
	if height == 0 {
		height, _ = chain.Height(ctx)
	}
	/*
	 * Response is of the structure
	 * {"@type":"/cosmos.auth.v1beta1.BaseAccount","address":"addr","pub_key":null,"account_number":"1","sequence":"0"}
	 */
	jsonRes := make(map[string]interface{})
	queryRes, _, err := chain.FullNodes[0].ExecQuery(ctx, "account", address, "--height", strconv.FormatInt(int64(height), 10))
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(queryRes, &jsonRes))

	sequence, err := dyno.GetString(jsonRes, "sequence")
	require.NoError(t, err)

	sequenceNum, _ := strconv.ParseUint(sequence, 10, 64)
	return sequenceNum
}

func checkTxInBlock(t *testing.T, chain *cosmos.CosmosChain, ctx context.Context, height uint64, user string, txHashes *BlockTxTest) {
	// Make sure all tx are in the same block
	for i, txHash := range txHashes.TxHashes {
		txResp, err := authTx.QueryTx(chain.FullNodes[0].CliContext(), txHash)
		require.NoError(t, err)
		t.Logf("Bank send msg %d BH: %d", i, txResp.Height)
		require.Equal(t, height, uint64(txResp.Height))
	}
	sequence := getAccountSequenceNumber(t, chain, ctx, user, height)
	require.Equal(t, uint64(len(txHashes.TxHashes)), sequence)
}

func TestMultipleBlockTx(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	xion, ctx := BuildXionChain(t, ModifyInterChainGenesis(ModifyInterChainGenesisFn{}, [][]string{{}}))

	txHashes := BlockTxTest{TxHashes: []string{}}

	// Retrieve the faucet address. Every chain has a faucet account
	faucet, err := xion.FullNodes[0].AccountKeyBech32(ctx, "faucet")
	require.NoError(t, err)
	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := int64(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]

	currentHeight, _ := xion.Height(ctx)
	var mu sync.Mutex

	mu.Lock()
	go func(t *testing.T, chain *cosmos.CosmosChain, ctx context.Context, blockHeight uint64, duration int, txHashes *BlockTxTest) {
		if currentHeight < blockHeight {
			stdout, ee, err := xion.FullNodes[0].Exec(ctx,
				txCommand(xion, xionUser.FormattedAddress(), faucet),
				nil,
			)
			if err != nil {
				t.Log(string(stdout))
				t.Log(string(ee))
				// Stop the routine
				currentHeight = blockHeight + 1
			} else {
				currentHeight, _ = chain.Height(ctx)
			}

			output := cosmos.CosmosTx{}
			err = json.Unmarshal([]byte(stdout), &output)
			if err != nil {
				currentHeight, _ = chain.Height(ctx)

			}

			// Save the hash of the send tx for later analysis
			txHashes.TxHashes = append(txHashes.TxHashes, output.TxHash)
			time.Sleep(time.Duration(duration) * time.Millisecond)
		}
	}(t, xion, ctx, currentHeight+1, 100, &txHashes)
	mu.Unlock()
	// Wait for some blocks and check if that supply stays the same
	testutil.WaitForBlocks(ctx, int(currentHeight)+4, xion)

	// Run test harness
	checkTxInBlock(t, xion, ctx, currentHeight, xionUser.FormattedAddress(), &txHashes)
}
