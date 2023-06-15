package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	authTx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/icza/dyno"
	ibctest "github.com/strangelove-ventures/interchaintest/v7"

	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
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
func txCommand(chain *cosmos.CosmosChain, sender string, receiver string, amount string) []string {
	command := []string{"tx"}
	return chain.FullNodes[0].NodeCommand(append(command,
		"bank", "send", sender, receiver,
		fmt.Sprintf("%s%s", amount, chain.Config().Denom),
		"--from", sender,
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
	t.Logf("There were %d transactions total ", len(txHashes.TxHashes))
	// Get the balance of the user
	balance, err := chain.GetBalance(ctx, user, chain.Config().Denom)
	require.NoError(t, err)
	t.Logf("User %s has balance %d", user, balance)

	sequence := getAccountSequenceNumber(t, chain, ctx, user, uint64(0))
	t.Logf("Current Seq: %d", sequence)

	// Make sure all tx are in the same block
	for i, txHash := range txHashes.TxHashes {
		txResp, err := authTx.QueryTx(chain.FullNodes[0].CliContext(), txHash)
		if err != nil {
			t.Logf("Error querying for tx %s", txHash)
			// Skip this tx
			continue
		}
		// Query for the tx
		stdOut, _, err := chain.FullNodes[0].ExecQuery(ctx, "tx", txHash)
		require.NoError(t, err)
		txJsonRes := make(map[string]interface{})
		require.NoError(t, json.Unmarshal(stdOut, &txJsonRes))

		txHeight := txResp.Height

		sequence := getAccountSequenceNumber(t, chain, ctx, user, uint64(txHeight))

		// Query for the all tx in the block
		blockOut, _, blockErr := chain.FullNodes[0].Exec(ctx, []string{"xiond", "query", "block", strconv.FormatInt(int64(txHeight), 10), "--home", "/var/cosmos-chain/xion_test_testnet-1", "--node", fmt.Sprintf("tcp://xion-1-fn-0-%s:26657", t.Name())}, nil)
		require.NoError(t, blockErr)

		blockJsonRes := make(map[string]interface{})

		// block, err := dyno.GetSlice(blockJsonRes, "block")
		// blockData, err := dyno.GetSlice(block, "data")
		// txs, err := dyno.GetSlice(blockData, "txs")
		require.NoError(t, json.Unmarshal(blockOut, &blockJsonRes))

		t.Logf("Bank send msg %d BH: %d, Seq: %d, Hash: %s", i, txHeight, sequence, txResp.TxHash)

		t.Logf("Tx %d: %v", i, txJsonRes)
		t.Logf("Block txs %d: %v", i, &blockJsonRes)

	}
}

// This test would test the result of having multiple transactions (same tx e.g. send 1 xion from A to B)
// sent from the same account in the same block.
func TestMulBlockTxSameAddrOneTx(t *testing.T) {
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
		currentHeight, _ := chain.Height(ctx)
		for currentHeight < blockHeight {
			stdout, ee, err := xion.FullNodes[0].Exec(ctx,
				txCommand(xion, xionUser.FormattedAddress(), faucet, "1000000"),
				nil,
			)
			if err != nil {
				t.Log(string(stdout))
				t.Log(string(ee))
			}
			output := cosmos.CosmosTx{}
			err = json.Unmarshal([]byte(stdout), &output)
			require.NoError(t, err)

			// Save the hash of the send tx for later analysis
			txHashes.TxHashes = append(txHashes.TxHashes, output.TxHash)
			time.Sleep(time.Duration(duration) * time.Microsecond)
			currentHeight, _ = chain.Height(ctx)
		}
	}(t, xion, ctx, currentHeight+2, 1, &txHashes)
	mu.Unlock()
	// Wait for some blocks and check if that supply stays the same
	testutil.WaitForBlocks(ctx, int(currentHeight)+4, xion)

	// Run test harness
	checkTxInBlock(t, xion, ctx, currentHeight, xionUser.FormattedAddress(), &txHashes)
}

// This test would test the result of having multiple transactions
// (diff tx e.g. send 1 xion from A to B then send 2 xion from A to B)
// sent from the same account in the same block.
func TestMulBlockTxSameAddrMulTx(t *testing.T) {
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
		currentHeight, _ := chain.Height(ctx)
		for currentHeight < blockHeight {
			// Send random amount of xion
			randAmount := rand.Intn(1000000)
			stdout, ee, err := xion.FullNodes[0].Exec(ctx,
				txCommand(xion, xionUser.FormattedAddress(), faucet, strconv.Itoa(randAmount)),
				nil,
			)
			if err != nil {
				t.Log(string(stdout))
				t.Log(string(ee))
			}
			output := cosmos.CosmosTx{}
			err = json.Unmarshal([]byte(stdout), &output)
			require.NoError(t, err)

			// Save the hash of the send tx for later analysis
			txHashes.TxHashes = append(txHashes.TxHashes, output.TxHash)
			time.Sleep(time.Duration(duration) * time.Microsecond)
			currentHeight, _ = chain.Height(ctx)
		}
	}(t, xion, ctx, currentHeight+2, 1, &txHashes)
	mu.Unlock()
	// Wait for some blocks and check if that supply stays the same
	testutil.WaitForBlocks(ctx, int(currentHeight)+4, xion)

	// Run test harness
	checkTxInBlock(t, xion, ctx, currentHeight, xionUser.FormattedAddress(), &txHashes)
}

// This test would test the result of having multiple transactions
// (diff tx e.g. send 1 xion from A to B then send 2 xion from A to B)
// sent from the same account in the same block.
// with increasing sequence
func TestMulBlockTxSameAddrMulTxIncSeq(t *testing.T) {
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
		currentHeight, _ := chain.Height(ctx)
		sequenceNumber := uint64(0)
		for currentHeight < blockHeight {
			// Send random amount of xion
			randAmount := rand.Intn(1000000)
			stdout, ee, err := xion.FullNodes[0].Exec(ctx,
				append(txCommand(xion, xionUser.FormattedAddress(), faucet, strconv.Itoa(randAmount)), "--sequence", strconv.Itoa(int(sequenceNumber))),
				nil,
			)
			if err != nil {
				t.Log(string(stdout))
				t.Log(string(ee))
			}
			output := cosmos.CosmosTx{}
			err = json.Unmarshal([]byte(stdout), &output)
			require.NoError(t, err)

			// Save the hash of the send tx for later analysis
			txHashes.TxHashes = append(txHashes.TxHashes, output.TxHash)
			time.Sleep(time.Duration(duration) * time.Microsecond)
			currentHeight, _ = chain.Height(ctx)
			sequenceNumber++
		}
	}(t, xion, ctx, currentHeight+2, 1, &txHashes)
	mu.Unlock()
	// Wait for some blocks and check if that supply stays the same
	testutil.WaitForBlocks(ctx, int(currentHeight)+4, xion)

	// Run test harness
	checkTxInBlock(t, xion, ctx, currentHeight, xionUser.FormattedAddress(), &txHashes)
}

// This test would test the result of having multiple transactions (same tx e.g. send 1 xion from A to B)
// sent from the same account in the same block with increasing sequence number.
func TestMulBlockTxSameAddrOneTxIncSeq(t *testing.T) {
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
		currentHeight, _ := chain.Height(ctx)
		sequenceNumber := 0
		for currentHeight < blockHeight {
			stdout, ee, err := xion.FullNodes[0].Exec(ctx,
				append(txCommand(xion, xionUser.FormattedAddress(), faucet, "1000000"), "--sequence", strconv.Itoa(int(sequenceNumber))),
				nil,
			)
			if err != nil {
				t.Log(string(stdout))
				t.Log(string(ee))
			}
			output := cosmos.CosmosTx{}
			err = json.Unmarshal([]byte(stdout), &output)
			require.NoError(t, err)

			// Save the hash of the send tx for later analysis
			txHashes.TxHashes = append(txHashes.TxHashes, output.TxHash)
			time.Sleep(time.Duration(duration) * time.Microsecond)
			currentHeight, _ = chain.Height(ctx)
			sequenceNumber++
		}
	}(t, xion, ctx, currentHeight+2, 1, &txHashes)
	mu.Unlock()
	// Wait for some blocks and check if that supply stays the same
	testutil.WaitForBlocks(ctx, int(currentHeight)+4, xion)

	// Run test harness
	checkTxInBlock(t, xion, ctx, currentHeight, xionUser.FormattedAddress(), &txHashes)
}

// This test would test the result of having multiple transactions
// (diff tx e.g. send 1 xion from A to B then send 2 xion from A to B)
// sent from the same account in the same block.
func TestMulBlockTxDiffAddrMulTx(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	xion, ctx := BuildXionChain(t, ModifyInterChainGenesis(ModifyInterChainGenesisFn{}, [][]string{{}}))

	txHashes := BlockTxTest{TxHashes: []string{}}

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := int64(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, []ibc.Chain{xion, xion, xion, xion, xion, xion, xion, xion, xion, xion, xion, xion, xion, xion}...)
	xionUser := users[0]

	currentHeight, _ := xion.Height(ctx)
	var mu sync.Mutex

	mu.Lock()
	go func(t *testing.T, chain *cosmos.CosmosChain, ctx context.Context, blockHeight uint64, duration int, txHashes *BlockTxTest) {
		currentHeight, _ := chain.Height(ctx)
		receivingUserId := 1
		for currentHeight < blockHeight {
			// Send random amount of xion
			randAmount := rand.Intn(1000000)
			stdout, ee, err := xion.FullNodes[0].Exec(ctx,
				txCommand(xion, xionUser.FormattedAddress(), users[receivingUserId].FormattedAddress(), strconv.Itoa(randAmount)),
				nil,
			)
			if err != nil {
				t.Log(string(stdout))
				t.Log(string(ee))
			}
			output := cosmos.CosmosTx{}
			err = json.Unmarshal([]byte(stdout), &output)
			require.NoError(t, err)

			// Save the hash of the send tx for later analysis
			txHashes.TxHashes = append(txHashes.TxHashes, output.TxHash)
			time.Sleep(time.Duration(duration) * time.Microsecond)
			currentHeight, _ = chain.Height(ctx)
			receivingUserId++
		}
	}(t, xion, ctx, currentHeight+2, 1, &txHashes)
	mu.Unlock()
	// Wait for some blocks and check if that supply stays the same
	testutil.WaitForBlocks(ctx, int(currentHeight)+4, xion)

	// Run test harness
	checkTxInBlock(t, xion, ctx, currentHeight, xionUser.FormattedAddress(), &txHashes)
}

// This test would test the result of having multiple transactions
// (diff tx e.g. send 1 xion from A to B then send 2 xion from A to B)
// sent from the same account in the same block.
// with increasing sequence number
func TestMulBlockTxDiffAddrMulTxIncSeq(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	xion, ctx := BuildXionChain(t, ModifyInterChainGenesis(ModifyInterChainGenesisFn{}, [][]string{{}}))

	txHashes := BlockTxTest{TxHashes: []string{}}

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := int64(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, []ibc.Chain{xion, xion, xion, xion, xion, xion, xion, xion, xion, xion, xion, xion, xion, xion}...)
	xionUser := users[0]

	currentHeight, _ := xion.Height(ctx)
	var mu sync.Mutex

	mu.Lock()
	go func(t *testing.T, chain *cosmos.CosmosChain, ctx context.Context, blockHeight uint64, duration int, txHashes *BlockTxTest) {
		currentHeight, _ := chain.Height(ctx)
		receivingUserId := 1
		sequenceNumber := 0
		for currentHeight < blockHeight {
			// Send random amount of xion
			randAmount := rand.Intn(1000000)
			stdout, ee, err := xion.FullNodes[0].Exec(ctx,
				append(txCommand(xion, xionUser.FormattedAddress(), users[receivingUserId].FormattedAddress(), strconv.Itoa(randAmount)), "--sequence", strconv.Itoa(sequenceNumber)),
				nil,
			)
			if err != nil {
				t.Log(string(stdout))
				t.Log(string(ee))
			}
			output := cosmos.CosmosTx{}
			err = json.Unmarshal([]byte(stdout), &output)
			require.NoError(t, err)

			// Save the hash of the send tx for later analysis
			txHashes.TxHashes = append(txHashes.TxHashes, output.TxHash)
			time.Sleep(time.Duration(duration) * time.Microsecond)
			currentHeight, _ = chain.Height(ctx)
			receivingUserId++
			sequenceNumber++
		}
	}(t, xion, ctx, currentHeight+2, 1, &txHashes)
	mu.Unlock()
	// Wait for some blocks and check if that supply stays the same
	testutil.WaitForBlocks(ctx, int(currentHeight)+4, xion)

	// Run test harness
	checkTxInBlock(t, xion, ctx, currentHeight, xionUser.FormattedAddress(), &txHashes)
}

// This test would test the result of having multiple transactions (same tx e.g. send 1 xion from A to B)
// sent from the same account in the same block with increasing sequence number.
// but with a failing tx in the middle
func TestMulBlockTxSameAddrOneFailingTxIncSeq(t *testing.T) {
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
		currentHeight, _ := chain.Height(ctx)
		sequenceNumber := 0
		for currentHeight < blockHeight {
			var err error
			var stdout, ee []byte
			if sequenceNumber == 5 {
				// send a tx with a failing message
				stdout, ee, err = xion.FullNodes[0].Exec(ctx,
					append(txCommand(xion, xionUser.FormattedAddress(), faucet, "100000000"), "--sequence", strconv.Itoa(int(sequenceNumber))),
					nil,
				)
			} else {
				stdout, ee, err = xion.FullNodes[0].Exec(ctx,
					append(txCommand(xion, xionUser.FormattedAddress(), faucet, "1000000"), "--sequence", strconv.Itoa(int(sequenceNumber))),
					nil,
				)
			}
			if err != nil {
				t.Log(string(stdout))
				t.Log(string(ee))
			}
			output := cosmos.CosmosTx{}
			err = json.Unmarshal([]byte(stdout), &output)
			require.NoError(t, err)

			// Save the hash of the send tx for later analysis
			txHashes.TxHashes = append(txHashes.TxHashes, output.TxHash)
			time.Sleep(time.Duration(duration) * time.Microsecond)
			currentHeight, _ = chain.Height(ctx)
			sequenceNumber++
		}
	}(t, xion, ctx, currentHeight+2, 1, &txHashes)
	mu.Unlock()
	// Wait for some blocks and check if that supply stays the same
	testutil.WaitForBlocks(ctx, int(currentHeight)+4, xion)

	// Run test harness
	checkTxInBlock(t, xion, ctx, currentHeight, xionUser.FormattedAddress(), &txHashes)
}
