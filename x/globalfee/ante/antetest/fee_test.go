package antetest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"

	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibcchanneltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"

	"cosmossdk.io/math"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"

	xionfeeante "github.com/burnt-labs/xion/x/globalfee/ante"
	globfeetypes "github.com/burnt-labs/xion/x/globalfee/types"
	xiontypes "github.com/burnt-labs/xion/x/xion/types"
)

var testGasLimit uint64 = 200_000

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) TestGetDefaultGlobalFees() {
	// set globalfees and min gas price
	feeDecorator, _ := s.SetupTestGlobalFeeStoreAndMinGasPrice([]sdk.DecCoin{}, &globfeetypes.Params{}, bondDenom)
	defaultGlobalFees, err := feeDecorator.DefaultZeroGlobalFee(s.ctx)
	s.Require().NoError(err)
	s.Require().Greater(len(defaultGlobalFees), 0)

	if defaultGlobalFees[0].Denom != testBondDenom {
		s.T().Fatalf("bond denom: %s, default global fee denom: %s", testBondDenom, defaultGlobalFees[0].Denom)
	}
}

func (s *IntegrationTestSuite) TestGlobalFeeSetAnteHandler() {
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}

	denominator := int64(100000)
	high := math.LegacyNewDec(400).Quo(math.LegacyNewDec(denominator)) // 0.004
	med := math.LegacyNewDec(200).Quo(math.LegacyNewDec(denominator))  // 0.002
	low := math.LegacyNewDec(100).Quo(math.LegacyNewDec(denominator))  // 0.001

	highFeeAmt := math.NewInt(high.MulInt64(int64(2) * denominator).RoundInt64())
	medFeeAmt := math.NewInt(med.MulInt64(int64(2) * denominator).RoundInt64())
	lowFeeAmt := math.NewInt(low.MulInt64(int64(2) * denominator).RoundInt64())

	globalfeeParamsEmpty := []sdk.DecCoin{}
	minGasPriceEmpty := []sdk.DecCoin{}
	globalfeeParams0 := []sdk.DecCoin{
		sdk.NewDecCoinFromDec("photon", math.LegacyNewDec(0)),
		sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(0)),
	}
	minGasPrice0 := []sdk.DecCoin{
		sdk.NewDecCoinFromDec("stake", math.LegacyNewDec(0)),
		sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(0)),
	}

	globalfeeParamsHigh := []sdk.DecCoin{
		sdk.NewDecCoinFromDec("uxion", high),
	}
	minGasPrice := []sdk.DecCoin{
		sdk.NewDecCoinFromDec("uxion", med),
	}
	globalfeeParamsLow := []sdk.DecCoin{
		sdk.NewDecCoinFromDec("uxion", low),
	}

	testCases := map[string]struct {
		minGasPrice []sdk.DecCoin
		globalFee   []sdk.DecCoin
		gasPrice    sdk.Coins
		gasLimit    uint64
		txMsg       sdk.Msg
		txCheck     bool
		expErr      bool
		networkFee  bool
	}{
		"empty_min_gas_price, nonempty global fee": {
			minGasPrice: minGasPriceEmpty,
			globalFee:   globalfeeParamsHigh,
			gasPrice:    sdk.NewCoins(sdk.NewCoin("uxion", highFeeAmt)),
			gasLimit:    testGasLimit,
			txMsg:       testdata.NewTestMsg(addr1),
			txCheck:     true,
			expErr:      false,
			networkFee:  true,
		},
		"nonempty min_gas_price with defaultGlobalFee denom, empty global fee": {
			minGasPrice: minGasPrice,
			globalFee:   globalfeeParamsEmpty, // default 0uxion
			gasPrice:    sdk.NewCoins(sdk.NewCoin("uxion", medFeeAmt)),
			gasLimit:    testGasLimit,
			txMsg:       testdata.NewTestMsg(addr1),
			txCheck:     true,
			expErr:      false,
			networkFee:  true,
		},
		"zero min_gas_price, zero global fee": {
			minGasPrice: minGasPrice0,
			globalFee:   globalfeeParams0,
			gasPrice:    sdk.NewCoins(sdk.NewCoin("stake", math.ZeroInt())),
			gasLimit:    testGasLimit,
			txMsg:       testdata.NewTestMsg(addr1),
			txCheck:     true,
			expErr:      false,
			networkFee:  true,
		},
		// zero min_gas_price and empty  global fee
		"zero min_gas_price, empty global fee, zero fee in min_gas_price_denom": {
			minGasPrice: minGasPrice0,
			globalFee:   globalfeeParamsEmpty,
			gasPrice:    sdk.NewCoins(sdk.NewCoin("stake", math.ZeroInt())),
			gasLimit:    testGasLimit,
			txMsg:       testdata.NewTestMsg(addr1),
			txCheck:     true,
			expErr:      false,
			networkFee:  true,
		},
		// empty min_gas_price, zero global fee
		"empty min_gas_price, zero global fee": {
			minGasPrice: minGasPriceEmpty,
			globalFee:   globalfeeParams0,
			gasPrice:    sdk.NewCoins(sdk.NewCoin("uxion", math.ZeroInt())),
			gasLimit:    testGasLimit,
			txMsg:       testdata.NewTestMsg(addr1),
			txCheck:     true,
			expErr:      false,
			networkFee:  true,
		},
		// zero min_gas_price, nonzero global fee
		"zero min_gas_price, nonzero global fee": {
			minGasPrice: minGasPrice0,
			globalFee:   globalfeeParamsLow,
			gasPrice:    sdk.NewCoins(sdk.NewCoin("uxion", lowFeeAmt)),
			gasLimit:    testGasLimit,
			txMsg:       testdata.NewTestMsg(addr1),
			txCheck:     true,
			expErr:      false,
			networkFee:  true,
		},
		"fee lower than globalfee and min_gas_price": {
			minGasPrice: minGasPrice,
			globalFee:   globalfeeParamsHigh,
			gasPrice:    sdk.NewCoins(sdk.NewCoin("uxion", lowFeeAmt)),
			gasLimit:    testGasLimit,
			txMsg:       testdata.NewTestMsg(addr1),
			txCheck:     true,
			expErr:      false,
			networkFee:  true,
		},
		"does not bypass msg type: ibc.core.channel.v1.MsgRecvPacket": {
			minGasPrice: minGasPrice,
			globalFee:   globalfeeParamsLow,
			gasPrice:    sdk.NewCoins(sdk.NewCoin("uxion", math.ZeroInt())),
			gasLimit:    testGasLimit,
			txMsg: ibcchanneltypes.NewMsgRecvPacket(
				ibcchanneltypes.Packet{}, nil, ibcclienttypes.Height{}, ""),
			txCheck:    true,
			expErr:     false,
			networkFee: true,
		},
		"bypass msg type: xion.v1.MsgSend": {
			minGasPrice: minGasPrice,
			globalFee:   globalfeeParamsLow,
			gasPrice:    sdk.NewCoins(sdk.NewCoin("uxion", math.ZeroInt())),
			gasLimit:    testGasLimit,
			txMsg:       &xiontypes.MsgSend{ToAddress: addr1.String(), FromAddress: addr1.String()},
			txCheck:     true,
			expErr:      false,
			networkFee:  false,
		},
		"bypass msg with excessive gas should fail": {
			minGasPrice: minGasPrice,
			globalFee:   globalfeeParamsLow,
			gasPrice:    sdk.NewCoins(sdk.NewCoin("uxion", math.ZeroInt())),
			gasLimit:    10_000_000, // 10x the max bypass gas limit (1M) - should fail
			txMsg:       &xiontypes.MsgSend{ToAddress: addr1.String(), FromAddress: addr1.String()},
			txCheck:     true,
			expErr:      true, // This should now fail due to gas limit enforcement
			networkFee:  false,
		},
		"bypass msg at gas limit boundary should pass": {
			minGasPrice: minGasPrice,
			globalFee:   globalfeeParamsLow,
			gasPrice:    sdk.NewCoins(sdk.NewCoin("uxion", math.ZeroInt())),
			gasLimit:    1_000_000, // Exactly at the max bypass gas limit - should pass
			txMsg:       &xiontypes.MsgSend{ToAddress: addr1.String(), FromAddress: addr1.String()},
			txCheck:     true,
			expErr:      false, // This should pass as it's within the limit
			networkFee:  false,
		},
	}

	globalfeeParams := &globfeetypes.Params{
		BypassMinFeeMsgTypes:            globfeetypes.DefaultBypassMinFeeMsgTypes,
		MaxTotalBypassMinFeeMsgGasUsage: globfeetypes.DefaultmaxTotalBypassMinFeeMsgGasUsage,
	}

	for name, tc := range testCases {
		s.Run(name, func() {
			// set globalfees and min gas price
			fmt.Println(name)
			globalfeeParams.MinimumGasPrices = tc.globalFee
			_, antehandler := s.SetupTestGlobalFeeStoreAndMinGasPrice(tc.minGasPrice, globalfeeParams, bondDenom)

			// set fee decorator to ante handler
			s.Require().NoError(s.txBuilder.SetMsgs(tc.txMsg))
			s.txBuilder.SetFeeAmount(tc.gasPrice)
			s.txBuilder.SetGasLimit(tc.gasLimit)
			tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
			s.Require().NoError(err)

			s.ctx = s.ctx.WithIsCheckTx(tc.txCheck)
			tcCtx, err := antehandler(s.ctx, tx, false)
			if !tc.expErr {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}

			// Calculate expected Min Gas Fee
			// Combine(local, network)
			// where:
			// Network = tc.globalFee * GasPrice
			// Local = tc.minGasPrice
			networkFee := getFee(tc.globalFee, tc.gasLimit)
			localFee := getFee(tc.minGasPrice, tc.gasLimit)
			minGas := tcCtx.MinGasPrices()
			expected, err := xionfeeante.CombinedFeeRequirement(networkFee, localFee)
			s.Require().NoError(err)
			if !tc.networkFee {
				s.Require().Equal(sdk.NewDecCoins(tc.minGasPrice...), minGas)
			} else {
				s.Require().Equal(expected, minGas)
			}
		})
	}
}

func getFee(originFee sdk.DecCoins, _ uint64) sdk.DecCoins {
	targetFee := originFee
	if len(originFee) == 0 {
		targetFee = []sdk.DecCoin{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(0))}
	}

	return targetFee.Sort()
}

func (s *IntegrationTestSuite) TestGetTxFeeRequired() {
	// create global fee params
	globalfeeParamsEmpty := &globfeetypes.Params{MinimumGasPrices: []sdk.DecCoin{}}

	// setup tests with default global fee i.e. "0uxion" and empty local min gas prices
	feeDecorator, _ := s.SetupTestGlobalFeeStoreAndMinGasPrice([]sdk.DecCoin{}, globalfeeParamsEmpty, noBondDenom)

	// set a subspace that doesn't have the stakingtypes.KeyBondDenom key registered
	// feeDecorator.StakingSubspace = s.app.GetSubspace(globfeetypes.ModuleName)

	// check that an error is returned when staking bond denom is empty
	_, err := feeDecorator.GetTxFeeRequired(s.ctx, nil)
	s.Require().Equal(err.Error(), "empty staking bond denomination")

	// set non-zero local min gas prices
	localMinGasPrices := sdk.NewDecCoins(sdk.NewDecCoin("uxion", math.NewInt(1)))

	// setup tests with non-empty local min gas prices
	feeDecorator, _ = s.SetupTestGlobalFeeStoreAndMinGasPrice(
		localMinGasPrices,
		globalfeeParamsEmpty,
		bondDenom,
	)

	// mock tx data
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}

	// privfee, accNums, accSeqs := []cryptotypes.PrivKey{priv2}, []uint64{0}, []uint64{0}
	s.Require().NoError(s.txBuilder.SetMsgs(testdata.NewTestMsg(addr1)))
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("uxion", math.ZeroInt())))

	s.txBuilder.SetGasLimit(uint64(1))
	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)

	// check that the required fees returned in CheckTx mode are equal to
	// local min gas prices since they're greater than the default global fee values.
	s.Require().True(s.ctx.IsCheckTx())
	res, err := feeDecorator.GetTxFeeRequired(s.ctx, tx)
	s.Require().True(res.Equal(localMinGasPrices))
	s.Require().NoError(err)

	// check that the global fee is returned in DeliverTx mode.
	globalFee, err := feeDecorator.GetGlobalFee(s.ctx)
	s.Require().NoError(err)

	ctx := s.ctx.WithIsCheckTx(false)
	res, err = feeDecorator.GetTxFeeRequired(ctx, tx)
	s.Require().NoError(err)
	s.Require().True(res.Equal(globalFee))
}

// TestNewFeeDecoratorPanic tests the panic condition in NewFeeDecorator
func (s *IntegrationTestSuite) TestNewFeeDecoratorPanic() {
	// Test panic when globalfeeSubspace doesn't have key table
	s.Run("panic when no key table", func() {
		// Create a subspace without setting up the key table
		subspaceWithoutKeyTable := s.app.GetSubspace("non-existent-module")

		// This should panic because HasKeyTable() returns false
		s.Require().Panics(func() {
			xionfeeante.NewFeeDecorator(subspaceWithoutKeyTable, bondDenom)
		})
	})
}

// TestAnteHandleEdgeCases tests edge cases to achieve 100% coverage
func (s *IntegrationTestSuite) TestAnteHandleEdgeCases() {
	// Test 1: Invalid FeeTx type (tx not implementing sdk.FeeTx interface)
	s.Run("invalid FeeTx type", func() {
		feeDecorator, _ := s.SetupTestGlobalFeeStoreAndMinGasPrice([]sdk.DecCoin{}, &globfeetypes.Params{}, bondDenom)

		// Create a mock transaction that doesn't implement sdk.FeeTx
		mockTx := &MockTx{}

		ctx := s.ctx.WithIsCheckTx(true)
		_, err := feeDecorator.AnteHandle(ctx, mockTx, false, NextFn)
		s.Require().Error(err)
		s.Require().Contains(err.Error(), "Tx must implement the sdk.FeeTx interface")
	})

	// Test 2: Simulation mode (should bypass all checks)
	s.Run("simulation mode", func() {
		feeDecorator, antehandler := s.SetupTestGlobalFeeStoreAndMinGasPrice([]sdk.DecCoin{}, &globfeetypes.Params{}, bondDenom)

		s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
		priv1, _, addr1 := testdata.KeyTestPubAddr()
		privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}

		// Set up a transaction that would normally fail
		err := s.txBuilder.SetMsgs(&xiontypes.MsgSend{ToAddress: addr1.String(), FromAddress: addr1.String()})
		s.Require().NoError(err)
		s.txBuilder.SetFeeAmount(sdk.NewCoins()) // Zero fees
		s.txBuilder.SetGasLimit(10_000_000)      // High gas limit

		tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
		s.Require().NoError(err)

		ctx := s.ctx.WithIsCheckTx(true)
		// simulate=true should bypass all checks
		_, err = feeDecorator.AnteHandle(ctx, tx, true, antehandler)
		s.Require().NoError(err) // Should pass in simulation mode
	})

	// Test 3: GetTxFeeRequired error path
	s.Run("GetTxFeeRequired error", func() {
		// Use noBondDenom to cause DefaultZeroGlobalFee to fail
		feeDecorator, _ := s.SetupTestGlobalFeeStoreAndMinGasPrice([]sdk.DecCoin{}, &globfeetypes.Params{}, noBondDenom)

		s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
		priv1, _, addr1 := testdata.KeyTestPubAddr()
		privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}

		// Set up a non-bypass message (this will trigger GetTxFeeRequired)
		err := s.txBuilder.SetMsgs(&ibcchanneltypes.MsgRecvPacket{
			Packet: ibcchanneltypes.Packet{}, Signer: addr1.String(),
		})
		s.Require().NoError(err)
		s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(100))))
		s.txBuilder.SetGasLimit(200_000)

		tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
		s.Require().NoError(err)

		ctx := s.ctx.WithIsCheckTx(true)
		_, err = feeDecorator.AnteHandle(ctx, tx, false, NextFn)
		// Should fail when GetTxFeeRequired encounters the invalid bondDenom
		s.Require().Error(err)
		s.Require().Contains(err.Error(), "empty staking bond denomination")
	})
}

// MockTx is a mock transaction that doesn't implement sdk.FeeTx
type MockTx struct{}

func (tx MockTx) GetMsgs() []sdk.Msg {
	return []sdk.Msg{}
}

func (tx MockTx) GetMsgsV2() ([]proto.Message, error) {
	return []proto.Message{}, nil
}

func (tx MockTx) ValidateBasic() error {
	return nil
}

// NextFn is a simple next function for testing
func NextFn(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
	return ctx, nil
}

// TestFeeUtilityFunctions tests the utility functions in fee_utils.go for 100% coverage
func (s *IntegrationTestSuite) TestFeeUtilityFunctions() {
	// Test CombinedFeeRequirement
	s.Run("CombinedFeeRequirement", func() {
		// Test empty global fees (should return error)
		globalFees := sdk.DecCoins{}
		minGasPrices := sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(1))}
		_, err := xionfeeante.CombinedFeeRequirement(globalFees, minGasPrices)
		s.Require().Error(err)
		s.Require().Contains(err.Error(), "global fee cannot be empty")

		// Test empty min gas prices (should return global fees)
		globalFees = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10))}
		minGasPrices = sdk.DecCoins{}
		result, err := xionfeeante.CombinedFeeRequirement(globalFees, minGasPrices)
		s.Require().NoError(err)
		s.Require().Equal(globalFees, result)

		// Test min gas price higher than global fee (should use min gas price)
		globalFees = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(1))}
		minGasPrices = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10))}
		result, err = xionfeeante.CombinedFeeRequirement(globalFees, minGasPrices)
		s.Require().NoError(err)
		s.Require().Equal(minGasPrices, result)

		// Test global fee higher than min gas price (should use global fee)
		globalFees = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10))}
		minGasPrices = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(1))}
		result, err = xionfeeante.CombinedFeeRequirement(globalFees, minGasPrices)
		s.Require().NoError(err)
		s.Require().Equal(globalFees, result)

		// Test different denoms (should use global fee since no overlap)
		globalFees = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10))}
		minGasPrices = sdk.DecCoins{sdk.NewDecCoinFromDec("atom", math.LegacyNewDec(5))}
		result, err = xionfeeante.CombinedFeeRequirement(globalFees, minGasPrices)
		s.Require().NoError(err)
		s.Require().Equal(globalFees, result)
	})

	// Test Find function
	s.Run("Find", func() {
		// Test empty coins
		coins := sdk.DecCoins{}
		found, coin := xionfeeante.Find(coins, "uxion")
		s.Require().False(found)
		s.Require().Equal(sdk.DecCoin{}, coin)

		// Test single coin - found
		coins = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10))}
		found, coin = xionfeeante.Find(coins, "uxion")
		s.Require().True(found)
		s.Require().Equal("uxion", coin.Denom)
		s.Require().Equal(math.LegacyNewDec(10), coin.Amount)

		// Test single coin - not found
		found, coin = xionfeeante.Find(coins, "atom")
		s.Require().False(found)
		s.Require().Equal(sdk.DecCoin{}, coin)

		// Test multiple coins - found at beginning
		coins = sdk.DecCoins{
			sdk.NewDecCoinFromDec("atom", math.LegacyNewDec(5)),
			sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10)),
		}
		found, coin = xionfeeante.Find(coins, "atom")
		s.Require().True(found)
		s.Require().Equal("atom", coin.Denom)

		// Test multiple coins - found at end
		found, coin = xionfeeante.Find(coins, "uxion")
		s.Require().True(found)
		s.Require().Equal("uxion", coin.Denom)

		// Test multiple coins - found in middle
		coins = sdk.DecCoins{
			sdk.NewDecCoinFromDec("atom", math.LegacyNewDec(5)),
			sdk.NewDecCoinFromDec("osmo", math.LegacyNewDec(7)),
			sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10)),
		}
		found, coin = xionfeeante.Find(coins, "osmo")
		s.Require().True(found)
		s.Require().Equal("osmo", coin.Denom)

		// Test multiple coins - not found (binary search left branch)
		found, coin = xionfeeante.Find(coins, "abc") // Less than middle element "osmo"
		s.Require().False(found)
		s.Require().Equal(sdk.DecCoin{}, coin)

		// Test multiple coins - not found (binary search right branch)
		found, coin = xionfeeante.Find(coins, "zzz") // Greater than middle element "osmo"
		s.Require().False(found)
		s.Require().Equal(sdk.DecCoin{}, coin)

		// Test multiple coins - not found (generic case)
		found, coin = xionfeeante.Find(coins, "notfound")
		s.Require().False(found)
		s.Require().Equal(sdk.DecCoin{}, coin)
	})

	// Test IsAllGT function
	s.Run("IsAllGT", func() {
		// Test empty a (should return false)
		a := sdk.DecCoins{}
		b := sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(1))}
		result := xionfeeante.IsAllGT(a, b)
		s.Require().False(result)

		// Test empty b (should return true)
		a = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(1))}
		b = sdk.DecCoins{}
		result = xionfeeante.IsAllGT(a, b)
		s.Require().True(result)

		// Test b not subset of a (should return false)
		a = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10))}
		b = sdk.DecCoins{sdk.NewDecCoinFromDec("atom", math.LegacyNewDec(5))}
		result = xionfeeante.IsAllGT(a, b)
		s.Require().False(result)

		// Test a > b (should return true)
		a = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10))}
		b = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(5))}
		result = xionfeeante.IsAllGT(a, b)
		s.Require().True(result)

		// Test a <= b (should return false)
		a = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(5))}
		b = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10))}
		result = xionfeeante.IsAllGT(a, b)
		s.Require().False(result)

		// Test a == b (should return false)
		a = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10))}
		b = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10))}
		result = xionfeeante.IsAllGT(a, b)
		s.Require().False(result)

		// Test multiple coins - all greater
		a = sdk.DecCoins{
			sdk.NewDecCoinFromDec("atom", math.LegacyNewDec(10)),
			sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(20)),
		}
		b = sdk.DecCoins{
			sdk.NewDecCoinFromDec("atom", math.LegacyNewDec(5)),
			sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10)),
		}
		result = xionfeeante.IsAllGT(a, b)
		s.Require().True(result)

		// Test multiple coins - not all greater
		a = sdk.DecCoins{
			sdk.NewDecCoinFromDec("atom", math.LegacyNewDec(3)),
			sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(20)),
		}
		b = sdk.DecCoins{
			sdk.NewDecCoinFromDec("atom", math.LegacyNewDec(5)),
			sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10)),
		}
		result = xionfeeante.IsAllGT(a, b)
		s.Require().False(result)
	})

	// Test DenomsSubsetOf function
	s.Run("DenomsSubsetOf", func() {
		// Test more denoms in a than b (should return false)
		a := sdk.DecCoins{
			sdk.NewDecCoinFromDec("atom", math.LegacyNewDec(5)),
			sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10)),
		}
		b := sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10))}
		result := xionfeeante.DenomsSubsetOf(a, b)
		s.Require().False(result)

		// Test denom in a not in b (should return false)
		a = sdk.DecCoins{sdk.NewDecCoinFromDec("atom", math.LegacyNewDec(5))}
		b = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10))}
		result = xionfeeante.DenomsSubsetOf(a, b)
		s.Require().False(result)

		// Test proper subset (should return true)
		a = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(5))}
		b = sdk.DecCoins{
			sdk.NewDecCoinFromDec("atom", math.LegacyNewDec(3)),
			sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10)),
		}
		result = xionfeeante.DenomsSubsetOf(a, b)
		s.Require().True(result)

		// Test equal sets (should return true)
		a = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(5))}
		b = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10))}
		result = xionfeeante.DenomsSubsetOf(a, b)
		s.Require().True(result)

		// Test empty a (should return true)
		a = sdk.DecCoins{}
		b = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDec(10))}
		result = xionfeeante.DenomsSubsetOf(a, b)
		s.Require().True(result)

		// Test both empty (should return true)
		a = sdk.DecCoins{}
		b = sdk.DecCoins{}
		result = xionfeeante.DenomsSubsetOf(a, b)
		s.Require().True(result)
	})
}

// PoC tests for bypass vulnerability #53694
func (s *IntegrationTestSuite) TestBypassGasCapNotEnforced() {
	// Test that bypass messages now properly enforce gas cap
	params := &globfeetypes.Params{
		MinimumGasPrices:                sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))},
		BypassMinFeeMsgTypes:            []string{},
		MaxTotalBypassMinFeeMsgGasUsage: 1_000, // small cap
	}

	feeDecorator, _ := s.SetupTestGlobalFeeStoreAndMinGasPrice([]sdk.DecCoin{}, params, bondDenom)

	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	// Create tx with gas that exceeds the bypass cap
	s.txBuilder.SetGasLimit(50_000)          // exceeds cap of 1,000
	s.txBuilder.SetFeeAmount(sdk.NewCoins()) // zero fees
	s.txBuilder.SetMsgs()                    // empty messages = bypass

	priv1, _, _ := testdata.KeyTestPubAddr()
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}

	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)

	ctx := s.ctx.WithIsCheckTx(true)

	// This should now fail because gas cap is enforced for bypass messages
	_, err = feeDecorator.AnteHandle(ctx, tx, false, NextFn)
	if err != nil {
		s.T().Logf("✅ Gas cap enforcement is working: %v", err)
		s.Require().Contains(err.Error(), "bypass messages cannot use more than")
	} else {
		s.T().Logf("❌ Gas cap enforcement is NOT working - bypass vulnerability still exists")
		s.Require().Fail("Expected error when gas exceeds bypass cap, but transaction was accepted")
	}
}

func (s *IntegrationTestSuite) TestBypassFeeDenomValidation() {
	// Test to demonstrate current behavior with fee denom validation for bypass messages
	params := &globfeetypes.Params{
		MinimumGasPrices:                sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))},
		BypassMinFeeMsgTypes:            []string{},
		MaxTotalBypassMinFeeMsgGasUsage: 1_000_000,
	}

	feeDecorator, _ := s.SetupTestGlobalFeeStoreAndMinGasPrice([]sdk.DecCoin{}, params, bondDenom)

	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	// Create tx with disallowed fee denom
	s.txBuilder.SetGasLimit(10_000)
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("uatom", math.NewInt(1)))) // disallowed denom
	s.txBuilder.SetMsgs()                                                        // empty messages = bypass

	priv1, _, _ := testdata.KeyTestPubAddr()
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}

	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)

	ctx := s.ctx.WithIsCheckTx(true)

	// This currently passes (demonstrating the remaining vulnerability)
	// but should ideally fail for disallowed fee denoms
	_, err = feeDecorator.AnteHandle(ctx, tx, false, NextFn)
	if err != nil {
		s.T().Logf("✅ Fee denom validation is working: %v", err)
		s.Require().Contains(err.Error(), "fee denom")
	} else {
		s.T().Logf("❌ Fee denom validation is NOT working - bypass vulnerability still exists")
		s.Require().Fail("Expected error for disallowed fee denom, but transaction was accepted")
	}
}
