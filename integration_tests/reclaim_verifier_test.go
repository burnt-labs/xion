package integration_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v8"

	"github.com/stretchr/testify/require"
)

func TestReclaimProofVerification(t *testing.T) {
	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}))
	xion, ctx := td.xionChain, td.ctx

	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	fp, err := os.Getwd()
	require.NoError(t, err)

	fundAmount := math.NewInt(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]
	xionUserAddr, err := types.AccAddressFromBech32(xionUser.FormattedAddress())
	require.NoError(t, err)
	// Upload contract
	codeID, err := xion.StoreContract(ctx, xionUser.FormattedAddress(), path.Join(fp, "integration_tests", "testdata", "contracts", "reclaim.wasm"))
	require.NoError(t, err)
	require.NotEmpty(t, codeID)

	// Instantiate contract
	jsonMsg := map[string]any{
		"owner": xionUserAddr.String(),
	}
	jsonMsgBytes, err := json.Marshal(jsonMsg)
	require.NoError(t, err)
	addr, err := xion.InstantiateContract(ctx, xionUser.FormattedAddress(), codeID, string(jsonMsgBytes), true)
	require.NoError(t, err)
	require.NotEmpty(t, addr)

	// Add epoch
	addEpochMsg := map[string]any{
		"add_epoch": map[string]any{
			"witness": []map[string]string{
				{
					"address": "0x244897572368Eadf65bfBc5aec98D8e5443a9072",
					"host":    "",
				},
			},
			"minimum_witness": "1",
		},
	}
	addEpochMsgBytes, err := json.Marshal(addEpochMsg)
	require.NoError(t, err)
	res, err := xion.ExecuteContract(ctx, xionUser.FormattedAddress(), addr, string(addEpochMsgBytes))
	require.NoError(t, err)
	txDetails, err := ExecQuery(t, ctx, xion.GetNode(), "tx", res.TxHash, "--output", "json")
	require.NoError(t, err)
	t.Logf("TxDetails: %s", txDetails)

	// Query epoch
	query := map[string]any{
		"get_epoch": map[string]any{"id": "1"},
	}
	queryBytes, err := json.Marshal(query)
	require.NoError(t, err)
	var epoch map[string]any
	err = xion.QueryContract(ctx, addr, string(queryBytes), &epoch)
	require.NoError(t, err)
	require.NotEmpty(t, epoch)
	fmt.Println("Epoch:", epoch)
	// Submit proof
	var proofMsg map[string]any
	file, err := os.ReadFile(path.Join(fp, "integration_tests", "testdata", "unsigned_msgs", "test.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(file, &proofMsg))
	proofMsgBytes, err := json.Marshal(proofMsg)
	require.NoError(t, err)

	res, err = xion.ExecuteContract(ctx, xionUser.FormattedAddress(), addr, string(proofMsgBytes))
	require.NoError(t, err)
	fmt.Println("Response:", res)
}
