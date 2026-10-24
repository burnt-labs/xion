package app

import (
	"crypto/sha256"
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/log/v2"
	"cosmossdk.io/math"
	globalfeetypes "github.com/burnt-labs/xion/x/globalfee/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
)

// blockGasMeterTestOptions returns the stock daemon base-app options only, so
// the regression exercises exactly the production wiring in NewWasmApp —
// including its restored aggregate block-gas meter.
func blockGasMeterTestOptions(t *testing.T) []func(*baseapp.BaseApp) {
	t.Helper()
	opts := simtestutil.AppOptionsMap{
		flags.FlagHome:           t.TempDir(),
		flags.FlagChainID:        "testing",
		server.FlagPruning:       "default",
		server.FlagMempoolMaxTxs: -1,
		server.FlagMinGasPrices:  "0.025stake",
	}
	return server.DefaultBaseappOptions(opts)
}

// setupBlockGasMeterApp initializes the v32 app with a funded payer, a settled
// genesis block, and the requested consensus Block.max_gas.
func setupBlockGasMeterApp(t *testing.T, maxGas int64) (*WasmApp, sdk.AccAddress, sdk.AccAddress, cryptotypes.PrivKey) {
	t.Helper()
	genTime := time.Unix(1_800_000_000, 0).UTC().Truncate(time.Second)
	gapp := NewWasmApp(log.NewNopLogger(), dbm.NewMemDB(), true,
		simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()}, nil,
		blockGasMeterTestOptions(t)...)
	t.Cleanup(func() { require.NoError(t, gapp.Close()) })

	// Fresh accounts; keys/state live only inside this test.
	payerPriv := secp256k1.GenPrivKey()
	payer := sdk.AccAddress(payerPriv.PubKey().Address())
	recipient := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	pv := mock.NewPV()
	pub, err := pv.GetPubKey()
	require.NoError(t, err)
	validator := cmttypes.NewValidator(pub, 1)
	vals := cmttypes.NewValidatorSet([]*cmttypes.Validator{validator})

	account := authtypes.NewBaseAccount(payer, payerPriv.PubKey(), 0, 0)
	genesis, err := GenesisStateWithValSet(gapp.AppCodec(), gapp.DefaultGenesis(), vals,
		[]authtypes.GenesisAccount{account},
		banktypes.Balance{Address: payer.String(), Coins: sdk.NewCoins(sdk.NewInt64Coin("stake", 1_000_000_000_000))})
	require.NoError(t, err)

	feeGenesis := globalfeetypes.DefaultGenesisState()
	feeGenesis.Params.MinimumGasPrices = sdk.NewDecCoins(sdk.NewDecCoinFromDec("stake", math.LegacyMustNewDecFromStr("0.025")))
	genesis[globalfeetypes.ModuleName] = gapp.AppCodec().MustMarshalJSON(feeGenesis)
	state, err := json.Marshal(genesis)
	require.NoError(t, err)

	params := *simtestutil.DefaultConsensusParams
	blockParams := *params.Block
	blockParams.MaxGas = maxGas
	params.Block = &blockParams
	_, err = gapp.InitChain(&abci.RequestInitChain{
		ChainId:         "testing",
		Time:            genTime,
		InitialHeight:   1,
		ConsensusParams: &params,
		AppStateBytes:   state,
		Validators:      []abci.ValidatorUpdate{},
	})
	require.NoError(t, err)
	_, err = gapp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: genTime, NextValidatorsHash: vals.Hash(), ProposerAddress: validator.Address})
	require.NoError(t, err)
	_, err = gapp.Commit()
	require.NoError(t, err)

	return gapp, payer, recipient, payerPriv
}

// signBlockGasMeterTransfers creates individually valid, fee-paying signed
// transfers from a single payer with consecutive sequences.
func signBlockGasMeterTransfers(t *testing.T, gapp *WasmApp, ctx sdk.Context, payer, recipient sdk.AccAddress, payerPriv cryptotypes.PrivKey, count, gasLimit int) [][]byte {
	t.Helper()
	account := gapp.AccountKeeper.GetAccount(ctx, payer)
	txs := make([][]byte, count)
	for i := range txs {
		msg := &banktypes.MsgSend{
			FromAddress: payer.String(),
			ToAddress:   recipient.String(),
			Amount:      sdk.NewCoins(sdk.NewInt64Coin("stake", 1)),
		}
		tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(int64(i+1))), gapp.TxConfig(), []sdk.Msg{msg},
			sdk.NewCoins(sdk.NewInt64Coin("stake", 25_000)), uint64(gasLimit), "testing",
			[]uint64{account.GetAccountNumber()}, []uint64{account.GetSequence() + uint64(i)}, payerPriv)
		require.NoError(t, err)
		txs[i], err = gapp.TxConfig().TxEncoder()(tx)
		require.NoError(t, err)
	}
	return txs
}

// TestBlockGasMeterEnforcesAggregateBudget verifies the restored aggregate
// block-gas metering. With a finite Block.max_gas, the stock SDK v0.55 defaults
// accepted and executed more aggregate gas than the limit; the restored meter
// must cut an over-budget block partway through and keep the committed balance
// consistent with executed transactions. With max_gas=-1 (today's live mainnet),
// budgeted execution remains unlimited and every transaction succeeds.
func TestBlockGasMeterEnforcesAggregateBudget(t *testing.T) {
	for _, tc := range []struct {
		name             string
		maxGas           int64
		expectedSucceeds int
		expectOverBudget bool
	}{
		// The over-budget case reproduces the stock-default behavior: 30
		// individually valid transfers at 1M declared gas each executed
		// completely (Σ 2,274,421 actual gas) against a 1,000,000 budget.
		{"finite budget is enforced", 1_000_000, 13, true},
		{"unset budget stays unlimited", -1, 30, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gapp, payer, recipient, payerPriv := setupBlockGasMeterApp(t, tc.maxGas)
			ctx := gapp.NewContext(true)
			require.Equal(t, tc.maxGas, gapp.GetConsensusParams(ctx).Block.MaxGas)
			txs := signBlockGasMeterTransfers(t, gapp, ctx, payer, recipient, payerPriv, 30, 1_000_000)
			require.Len(t, txs, 30)

			blockHeight := gapp.LastBlockHeight() + 1
			blockHash := sha256.Sum256([]byte("block-gas-meter-enforcement"))
			proposal, err := gapp.ProcessProposal(&abci.RequestProcessProposal{
				Height: blockHeight, Time: ctx.BlockTime(), Hash: blockHash[:], Txs: txs,
				ProposerAddress: ctx.BlockHeader().ProposerAddress,
			})
			require.NoError(t, err)
			result, err := gapp.FinalizeBlock(&abci.RequestFinalizeBlock{
				Height: blockHeight, Time: ctx.BlockTime(), Hash: blockHash[:], Txs: txs,
				ProposerAddress: ctx.BlockHeader().ProposerAddress,
			})
			require.NoError(t, err)

			var totalGas int64
			succeeded := 0
			for i, txResult := range result.TxResults {
				totalGas += txResult.GasUsed
				if txResult.Code == 0 {
					succeeded++
				} else {
					t.Logf("tx[%d] status=%d gas=%d log=%s", i, txResult.Code, txResult.GasUsed, txResult.Log)
				}
			}
			_, err = gapp.Commit()
			require.NoError(t, err)
			balance := gapp.BankKeeper.GetBalance(gapp.NewContext(true), recipient, "stake").Amount
			t.Logf("max_gas=%d succeeded=%d total_actual_gas=%d balance=%s", tc.maxGas, succeeded, totalGas, balance)

			if tc.expectOverBudget {
				require.Greater(t, totalGas, tc.maxGas, "test setup must generate a block that exceeds the budget")
				require.Less(t, succeeded, len(txs), "the meter must stop the over-budget block before executing every transaction")
				for i := succeeded; i < len(result.TxResults); i++ {
					require.Equal(t, uint32(11), result.TxResults[i].Code, "post-meter transactions must fail with out-of-gas class")
				}
			}
			// Success count is workload-specific (the meter cuts wherever the
			// budget is exhausted); assert the boundary behavior, not an
			// exact count beyond this fixture's stable seed set.
			require.Equal(t, tc.expectedSucceeds, succeeded)
			require.Equal(t, math.NewInt(int64(tc.expectedSucceeds)), balance, "committed state must match executed transactions")
			// The stock NoOp-mempool ProcessProposal accepts without aggregate
			// validation; this patch deliberately restores only the
			// execution-layer backstop.
			require.Equal(t, abci.ResponseProcessProposal_ACCEPT, proposal.Status)
		})
	}
}
