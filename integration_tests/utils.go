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
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	authTx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/icza/dyno"

	"github.com/docker/docker/client"
	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const (
	votingPeriod     = "10s"
	maxDepositPeriod = "10s"
)

// Function type for any function that modify the genesis file
type ModifyInterChainGenesisFn []func(ibc.ChainConfig, []byte, ...string) ([]byte, error)

type TestData struct {
	xionChain *cosmos.CosmosChain
	ctx       context.Context
	client    *client.Client
}

func RawJSONMsg(from, to, denom string) []byte {
	msg := fmt.Sprintf(`
	{
		"@type": "/cosmos.bank.v1beta1.MsgSend",
		"from_address": "%s",
		"to_address": "%s",
		"amount": [
			{
				"denom": "%s",
				"amount": "12345"
			}
		]
	}
	`, from, to, denom)
	var rawMsg json.RawMessage = []byte(msg)
	return rawMsg
}

func BuildXionChain(t *testing.T, gas string, modifyGenesis func(ibc.ChainConfig, []byte) ([]byte, error)) TestData {
	ctx := context.Background()

	var numFullNodes = 1
	var numValidators = 3

	// pulling image from env to foster local dev
	imageTag := os.Getenv("XION_IMAGE")
	imageTagComponents := strings.Split(imageTag, ":")

	// Chain factory
	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
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
				//GasPrices:              "0.1uxion",
				GasPrices:              gas,
				GasAdjustment:          2.0,
				Type:                   "cosmos",
				ChainID:                "xion-1",
				Bin:                    "xiond",
				Bech32Prefix:           "xion",
				Denom:                  "uxion",
				TrustingPeriod:         "336h",
				ModifyGenesis:          modifyGenesis,
				UsingNewGenesisCommand: true,
			},
			NumValidators: &numValidators,
			NumFullNodes:  &numFullNodes,
		},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	xion := chains[0].(*cosmos.CosmosChain)

	client, network := interchaintest.DockerSetup(t)

	// Prep Interchain
	ic := interchaintest.NewInterchain().
		AddChain(xion)

	// Log location
	f, err := interchaintest.CreateLogFile(fmt.Sprintf("%d.json", time.Now().Unix()))
	require.NoError(t, err)
	// Reporter/logs
	rep := testreporter.NewReporter(f)
	eRep := rep.RelayerExecReporter(t)

	// Build Interchain
	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:          t.Name(),
		Client:            client,
		NetworkID:         network,
		BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),

		SkipPathCreation: false},
	),
	)
	return TestData{xion, ctx, client}
}

/*
 * This function is a helper to run all functions that modify the genesis file
 * in a chain. It takes a list of functions of the type ModifyInterChainGenesisFn and a list of list of parameters for each
 * function. Each array in the parameter list are the parameters for a functions of the same index
 */
func ModifyInterChainGenesis(fns ModifyInterChainGenesisFn, params [][]string) func(ibc.ChainConfig, []byte) ([]byte, error) {
	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
		res := genbz
		var err error

		for i, fn := range fns {
			res, err = fn(chainConfig, res, params[i]...)
			if err != nil {
				return nil, fmt.Errorf("failed to modify genesis: %w", err)
			}
		}
		return res, nil
	}
}

// This function modifies the proposal parameters of the gov module in the genesis file
func ModifyGenesisShortProposals(chainConfig ibc.ChainConfig, genbz []byte, params ...string) ([]byte, error) {
	g := make(map[string]interface{})
	if err := json.Unmarshal(genbz, &g); err != nil {
		return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
	}
	if err := dyno.Set(g, params[0], "app_state", "gov", "params", "voting_period"); err != nil {
		return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
	}
	if err := dyno.Set(g, params[1], "app_state", "gov", "params", "max_deposit_period"); err != nil {
		return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
	}
	if err := dyno.Set(g, chainConfig.Denom, "app_state", "gov", "params", "min_deposit", 0, "denom"); err != nil {
		return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
	}
	if err := dyno.Set(g, "100", "app_state", "gov", "params", "min_deposit", 0, "amount"); err != nil {
		return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
	}
	out, err := json.Marshal(g)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal genesis bytes to json: %w", err)
	}
	return out, nil
}

// This function modifies the inflation parameters of the mint module in the genesis file
func ModifyGenesisInflation(chainConfig ibc.ChainConfig, genbz []byte, params ...string) ([]byte, error) {
	g := make(map[string]interface{})
	if err := json.Unmarshal(genbz, &g); err != nil {
		return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
	}
	if err := dyno.Set(g, params[0], "app_state", "mint", "params", "inflation_min"); err != nil {
		return nil, fmt.Errorf("failed to set inflation in genesis json: %w", err)
	}
	if err := dyno.Set(g, params[1], "app_state", "mint", "params", "inflation_max"); err != nil {
		return nil, fmt.Errorf("failed to set inflation in genesis json: %w", err)
	}
	if err := dyno.Set(g, params[2], "app_state", "mint", "params", "inflation_rate_change"); err != nil {
		return nil, fmt.Errorf("failed to set rate of inflation change in genesis json: %w", err)
	}
	out, err := json.Marshal(g)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal genesis bytes to json: %w", err)
	}
	return out, nil
}

// Helper method to retrieve the total token supply for a chain at some particular history denoted by the block height
func getTotalCoinSupplyInBank(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, denom string, blockHeight uint64) string {
	if blockHeight == 0 {
		// No history is required so use the most recent block height
		blockHeight, _ = xion.Height(ctx)
		require.Greater(t, blockHeight, 0)
	}

	/*
	 * Response is of the structure
	 * {"supply":[{"denom":"uxion","amount":"110000002059725"}]}
	 */
	jsonRes := make(map[string]interface{})
	queryRes, _, err := xion.FullNodes[0].ExecQuery(ctx, "bank", "total", "--height", strconv.FormatInt(int64(blockHeight), 10))
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(queryRes, &jsonRes))

	// Presuming we are the only denom on the chain then the returned array only has one Coin type ours (uxion)
	totalSupply, err := dyno.GetSlice(jsonRes, "supply")
	require.NoError(t, err)
	xionCoin := totalSupply[0]
	require.NotEmpty(t, xionCoin)
	// Make sure we selected the uxion denom
	xionCoinDenom, err := dyno.GetString(xionCoin, "denom")
	require.NoError(t, err)
	require.Equal(t, xionCoinDenom, denom)
	// Get the returned amount
	initialXionSupply, err := dyno.GetString(xionCoin, "amount")
	require.NoError(t, err)
	return initialXionSupply
}

// This function gets the bank balance for an address at some particular history denoted by the block height
func getAddressBankBalanceAtHeight(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, address string, denom string, blockHeight uint64) string {
	if blockHeight == 0 {
		blockHeight, _ = xion.Height(ctx)
		require.Greater(t, blockHeight, 0)
	}

	/*
	 * Response is of the structure
	 * {"supply":[{"denom":"uxion","amount":"110000002059725"}]}
	 */
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

// This function gets the module address for a module name
func GetModuleAddress(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, moduleName string) string {
	/*
		* Response is of the type
		* {
			"account": {
				"@type": "/cosmos.auth.v1beta1.ModuleAccount",
				"base_account": {
				"address": "xion1jv65s3grqf6v6jl3dp4t6c9t9rk99cd89k77l5",
				"pub_key": null,
				"account_number": "3",
				"sequence": "0"
				},
				"name": "distribution",
				"permissions": []
			}
			}
	*/
	jsonRes := make(map[string]interface{})
	queryRes, _, err := xion.FullNodes[0].ExecQuery(ctx, "auth", "module-account", moduleName)
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(queryRes, &jsonRes))

	moduleAddress, err := dyno.GetString(jsonRes, "account", "base_account", "address")
	require.NoError(t, err)
	t.Logf("%s module address: %s", moduleName, moduleAddress)
	return moduleAddress
}

// Retrieve a block annual provision. This is the minted tokens for the block for validators and delegator
func GetBlockAnnualProvision(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, denom string, blockHeight uint64) math.LegacyDec {
	if blockHeight == 0 {
		blockHeight, _ = xion.Height(ctx)
		require.Greater(t, blockHeight, 0)
	}

	// Query the current block provision
	// Response is a string
	var annualProvision json.Number
	queryRes, _, err := xion.FullNodes[0].ExecQuery(ctx, "mint", "annual-provisions", "--height", strconv.FormatInt(int64(blockHeight), 10))
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
	return math.LegacyMustNewDecFromStr(annualProvision.String()).QuoInt(math.NewInt(blocksPerYear))
}

// This test confirms the property of the module described at
// https://www.notion.so/burntlabs/Mint-Module-Blog-Post-78f59fb108c04e9ea5fa826dda30a340
/*
 * The mint harness test logic is as follows
 * Given a particular block height, we compute the minted or burned tokens at that block using the following algorithm

 * Get the total token supply at the previous height
 * Get the total token supply at the current height
 * Compute the change in the total supply (d)
 * Get the balance of the distribution module account in the previous block
 * Get the balance of the distribution module account at the current block
 * The difference (f) is the sum of the total fees and minted token at the current block
 * The difference ( d - f )is the total fees accrued at the current block
 * Get the block provision at the current height (this is the number of tokens to be minted for validators and delegators at that block
 * Perform the following checks
 * If we have more fees accrued than the block provision, we burn the excess tokens
 * If instead there are fewer fees than block provision, we only mint (provision - fees) tokens to meet expectations
 * Otherwise, we had equal fees and provisions. We do not mint or burn any tokens
 * If these checks passes then the mint module functions as expected
 */
func MintModuleTestHarness(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, blockHeight int) {
	// Get bank supply at previous height
	previousXionBankSupply, err := strconv.ParseInt(getTotalCoinSupplyInBank(t, xion, ctx, xion.Config().Denom, uint64(blockHeight-1)), 10, 64)
	t.Logf("Previous Xion bank supply: %d", previousXionBankSupply)
	require.NoError(t, err, "bank supply should be convertible to an int64")
	// Get bank supply at current height
	currentXionBankSupply, err := strconv.ParseInt(getTotalCoinSupplyInBank(t, xion, ctx, xion.Config().Denom, uint64(blockHeight)), 10, 64)
	t.Logf("Current Xion bank supply: %d", currentXionBankSupply)
	require.NoError(t, err, "bank supply should be convertible to an int64")
	tokenChange := currentXionBankSupply - previousXionBankSupply

	// Get the distribution module account address
	distributionModuleAddress := GetModuleAddress(t, xion, ctx, "distribution")
	// Get distribution module account balance in previous height
	previousDistributionModuleBalance, err := strconv.ParseInt(getAddressBankBalanceAtHeight(t, xion, ctx, distributionModuleAddress, xion.Config().Denom, uint64(blockHeight-1)), 10, 64)
	require.NoError(t, err, "distribution module balance should be convertible to an int64")
	// Get distribution module account balance in current height
	currentDistributionModuleBalance, err := strconv.ParseInt(getAddressBankBalanceAtHeight(t, xion, ctx, distributionModuleAddress, xion.Config().Denom, uint64(blockHeight)), 10, 64)
	require.NoError(t, err, "distribution module balance should be convertible to an int64")

	delta := currentDistributionModuleBalance - previousDistributionModuleBalance

	// Fees Accrued is the total fees in a block. Since the tokens in the distribution module account
	// are the fees and the minted tokens, we can compute the fees accrued by subtracting the token change
	// from the delta
	feesAccrued := delta - tokenChange
	t.Logf("Fees accrued: %d", feesAccrued)

	// Get the block provision. This is the minted tokens for the block for validators and delegator
	blockProvision := GetBlockAnnualProvision(t, xion, ctx, xion.Config().Denom, uint64(blockHeight))

	if blockProvision.TruncateInt().GT(math.NewInt(feesAccrued)) {
		// We have minted tokens because the fees accrued is less than the block provision
		mintedTokens := blockProvision.TruncateInt().Sub(math.NewInt(feesAccrued))
		t.Logf("Minted tokens: %d and Token change: %d", mintedTokens.Int64(), int64(tokenChange))
		require.Equal(t, mintedTokens, math.NewInt(int64(tokenChange)))
	} else if blockProvision.TruncateInt().LT(math.NewInt(feesAccrued)) {
		// We have burned tokens because the fees accrued is greater than the block provision so the fees
		// accrued are used to pay validators and the remaining is burned
		burnedTokens := math.NewInt(feesAccrued).Sub(blockProvision.TruncateInt())
		t.Logf("Burned tokens: %d and Token change: %d", burnedTokens.Int64(), tokenChange)
		require.Equal(t, burnedTokens, math.NewInt(tokenChange).Abs())
	} else {
		// We have not minted or burned tokens but just used all fees accrued to pay validators
		require.Equal(t, math.NewInt(0), math.NewInt(tokenChange))
		t.Logf("No minted or Burned tokens. Token change: %d", tokenChange)
	}
}

// Run Mint module test harness over some random block height
// Chain must have at least 12 blocks
func VerifyMintModuleTestRandomBlocks(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context) {

	currentBlockHeight, err := xion.Height(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, currentBlockHeight, uint64(12))
	// Get a random number from 1 to the (currentBlockHeight - 10)
	randomHeight := rand.Intn(int(currentBlockHeight)-11) + 2 // we start from 2 because we need at least 2 blocks to run the test

	for i := randomHeight; i < randomHeight+10; i++ {
		t.Logf("Current random height: %d", i)
		MintModuleTestHarness(t, xion, ctx, i)
	}
}

// Run Mint module test over some txHash
func VerifyMintModuleTest(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, txHashes []string) {

	for i, txHash := range txHashes {
		txResp, err := authTx.QueryTx(xion.FullNodes[0].CliContext(), txHash)
		require.NoError(t, err)
		t.Logf("Bank send msg %d BH: %d", i, txResp.Height)
		MintModuleTestHarness(t, xion, ctx, int(txResp.Height)+1) // check my block and the next one
	}
}

func TxCommandOverrideGas(t *testing.T, tn *cosmos.ChainNode, keyName, gas string, command ...string) []string {
	command = append([]string{"tx"}, command...)
	return tn.NodeCommand(append(command,
		"--from", keyName,
		"--gas-prices", gas,
		"--gas-adjustment", fmt.Sprint(tn.Chain.Config().GasAdjustment),
		"--gas", "auto",
		"--keyring-backend", keyring.BackendTest,
		"--output", "json",
		"-y",
	)...)
}

func ExecTx(t *testing.T, ctx context.Context, tn *cosmos.ChainNode, keyName string, command ...string) (string, error) {
	stdout, _, err := tn.Exec(ctx, TxCommandOverrideGas(t, tn, keyName, "0.1uxion", command...), nil)
	if err != nil {
		return "", err
	}
	output := cosmos.CosmosTx{}
	err = json.Unmarshal([]byte(stdout), &output)
	if err != nil {
		return "", err
	}
	if output.Code != 0 {
		return output.TxHash, fmt.Errorf("transaction failed with code %d: %s", output.Code, output.RawLog)
	}
	if err := testutil.WaitForBlocks(ctx, 2, tn); err != nil {
		return "", err
	}
	return output.TxHash, nil
}

func ExecQuery(t *testing.T, ctx context.Context, tn *cosmos.ChainNode, command ...string) (map[string]interface{}, error) {
	jsonRes := make(map[string]interface{})
	t.Logf("querying with cmd: %s", command)
	output, _, err := tn.ExecQuery(ctx, command...)
	if err != nil {
		return jsonRes, err
	}
	require.NoError(t, json.Unmarshal(output, &jsonRes))

	return jsonRes, nil
}

func ExecBin(t *testing.T, ctx context.Context, tn *cosmos.ChainNode, keyName string, command ...string) (map[string]interface{}, error) {
	jsonRes := make(map[string]interface{})
	output, _, err := tn.ExecBin(ctx, command...)
	if err != nil {
		return jsonRes, err
	}
	fmt.Printf("%+s\n", output)
	require.NoError(t, json.Unmarshal(output, &jsonRes))

	return jsonRes, nil
}
