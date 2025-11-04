package e2e_xion

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/burnt-labs/xion/e2e_tests/testlib"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
)

func init() {
	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")
}

// TestXionMinFeeConcurrentTransactions tests concurrent transaction handling
func TestXionMinFeeConcurrentTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	ctx := context.Background()

	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	chainSpec.GasPrices = "0.025uxion"
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	fundAmount := math.NewInt(100_000_000_000)
	numUsers := 5
	users := make([]ibc.Chain, numUsers)
	for i := 0; i < numUsers; i++ {
		users[i] = xion
	}
	fundedUsers := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, users...)

	t.Run("HandlesConcurrentValidFees", func(t *testing.T) {
		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		// Launch 5 concurrent transactions with valid fees
		for i := 0; i < numUsers; i++ {
			wg.Add(1)
			go func(userIdx int) {
				defer wg.Done()

				sender := fundedUsers[userIdx]
				recipient := fundedUsers[(userIdx+1)%numUsers]

				// Add delay to avoid sequence conflicts
				time.Sleep(time.Duration(userIdx*100) * time.Millisecond)

				_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
					sender.KeyName(),
					"0.025uxion",
					"bank", "send", sender.FormattedAddress(),
					recipient.FormattedAddress(),
					fmt.Sprintf("%d%s", 100, xion.Config().Denom),
					"--chain-id", xion.Config().ChainID,
				)

				if err == nil {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()

		err := testutil.WaitForBlocks(ctx, 3, xion)
		require.NoError(t, err)

		// At least some transactions should succeed
		require.Greater(t, successCount, 0, "At least some concurrent transactions should succeed")
		t.Logf("✓ %d/%d concurrent transactions succeeded", successCount, numUsers)
	})

	t.Run("RejectsConcurrentBelowMinimumFees", func(t *testing.T) {
		var wg sync.WaitGroup
		failCount := 0
		var mu sync.Mutex

		// Launch concurrent transactions with invalid fees
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(userIdx int) {
				defer wg.Done()

				sender := fundedUsers[userIdx]
				recipient := fundedUsers[(userIdx+1)%numUsers]

				time.Sleep(time.Duration(userIdx*100) * time.Millisecond)

				_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
					sender.KeyName(),
					"0.01uxion", // Below minimum
					"bank", "send", sender.FormattedAddress(),
					recipient.FormattedAddress(),
					fmt.Sprintf("%d%s", 100, xion.Config().Denom),
					"--chain-id", xion.Config().ChainID,
				)
				if err != nil {
					mu.Lock()
					failCount++
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()

		// All below-minimum transactions should fail
		require.Equal(t, 3, failCount, "All below-minimum fee transactions should be rejected")
	})
}

// TestXionMinFeeSequenceHandling tests transaction sequence handling
func TestXionMinFeeSequenceHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	ctx := context.Background()

	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	chainSpec.GasPrices = "0.025uxion"
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, xion)
	sender := users[0]
	recipient := users[1]

	t.Run("ProcessesSequentialTransactions", func(t *testing.T) {
		// Send 3 transactions sequentially
		for i := 0; i < 3; i++ {
			_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
				sender.KeyName(),
				"0.025uxion",
				"bank", "send", sender.FormattedAddress(),
				recipient.FormattedAddress(),
				fmt.Sprintf("%d%s", 100, xion.Config().Denom),
				"--chain-id", xion.Config().ChainID,
			)
			require.NoError(t, err, "Sequential transaction %d should succeed", i+1)

			// Small delay between transactions
			time.Sleep(500 * time.Millisecond)
		}

		err := testutil.WaitForBlocks(ctx, 2, xion)
		require.NoError(t, err)
	})
}
