package antetest

import (
	"fmt"
	"testing"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibcchanneltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/stretchr/testify/suite"

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
	high := sdk.NewDec(400).Quo(sdk.NewDec(denominator)) // 0.004
	med := sdk.NewDec(200).Quo(sdk.NewDec(denominator))  // 0.002
	low := sdk.NewDec(100).Quo(sdk.NewDec(denominator))  // 0.001

	highFeeAmt := sdk.NewInt(high.MulInt64(int64(2) * denominator).RoundInt64())
	medFeeAmt := sdk.NewInt(med.MulInt64(int64(2) * denominator).RoundInt64())
	lowFeeAmt := sdk.NewInt(low.MulInt64(int64(2) * denominator).RoundInt64())

	globalfeeParamsEmpty := []sdk.DecCoin{}
	minGasPriceEmpty := []sdk.DecCoin{}
	globalfeeParams0 := []sdk.DecCoin{
		sdk.NewDecCoinFromDec("photon", sdk.NewDec(0)),
		sdk.NewDecCoinFromDec("uxion", sdk.NewDec(0)),
	}
	minGasPrice0 := []sdk.DecCoin{
		sdk.NewDecCoinFromDec("stake", sdk.NewDec(0)),
		sdk.NewDecCoinFromDec("uxion", sdk.NewDec(0)),
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
		gasLimit    sdk.Gas
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
			networkFee:  false,
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
			gasPrice:    sdk.NewCoins(sdk.NewCoin("stake", sdk.ZeroInt())),
			gasLimit:    testGasLimit,
			txMsg:       testdata.NewTestMsg(addr1),
			txCheck:     true,
			expErr:      false,
		},
		// zero min_gas_price and empty  global fee
		"zero min_gas_price, empty global fee, zero fee in min_gas_price_denom": {
			minGasPrice: minGasPrice0,
			globalFee:   globalfeeParamsEmpty,
			gasPrice:    sdk.NewCoins(sdk.NewCoin("stake", sdk.ZeroInt())),
			gasLimit:    testGasLimit,
			txMsg:       testdata.NewTestMsg(addr1),
			txCheck:     true,
			expErr:      false,
		},
		// empty min_gas_price, zero global fee
		"empty min_gas_price, zero global fee": {
			minGasPrice: minGasPriceEmpty,
			globalFee:   globalfeeParams0,
			gasPrice:    sdk.NewCoins(sdk.NewCoin("uxion", sdk.ZeroInt())),
			gasLimit:    testGasLimit,
			txMsg:       testdata.NewTestMsg(addr1),
			txCheck:     true,
			expErr:      false,
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
		},
		"fee lower than globalfee and min_gas_price": {
			minGasPrice: minGasPrice,
			globalFee:   globalfeeParamsHigh,
			gasPrice:    sdk.NewCoins(sdk.NewCoin("uxion", lowFeeAmt)),
			gasLimit:    testGasLimit,
			txMsg:       testdata.NewTestMsg(addr1),
			txCheck:     true,
			expErr:      false,
		},
		"does not bypass msg type: ibc.core.channel.v1.MsgRecvPacket": {
			minGasPrice: minGasPrice,
			globalFee:   globalfeeParamsLow,
			gasPrice:    sdk.NewCoins(sdk.NewCoin("uxion", sdk.ZeroInt())),
			gasLimit:    testGasLimit,
			txMsg: ibcchanneltypes.NewMsgRecvPacket(
				ibcchanneltypes.Packet{}, nil, ibcclienttypes.Height{}, ""),
			txCheck: true,
			expErr:  false,
		},
		"bypass msg type: xion.v1.MsgSend.": {
			minGasPrice: minGasPrice,
			globalFee:   globalfeeParamsLow,
			gasPrice:    sdk.NewCoins(sdk.NewCoin("uxion", sdk.ZeroInt())),
			gasLimit:    testGasLimit,
			txMsg:       &xiontypes.MsgSend{ToAddress: addr1.String(), FromAddress: addr1.String()},
			txCheck:     true,
			expErr:      false,
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
			// TODO: add final fee evaluation, for msgs that are bypassed vs msgs that are not
			if !tc.expErr {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
			g := tcCtx.MinGasPrices()
			// Calculate expected Min Gas Fee
			// Combine(local, network)
			// where:
			// Network = tc.globalFee * GasPrice
			// Local = tc.minGasPrice
			expected, _ := xionfeeante.CombinedFeeRequirement(getFee(tc.minGasPrice, tc.gasLimit), getFee(tc.minGasPrice, tc.gasLimit))
			s.Require().Equal(sdk.NewDecCoinsFromCoins(expected...), g)
		})
	}
}

func getFee(networkFee sdk.DecCoins, gasLimit uint64) sdk.Coins {
	requiredGlobalFees := make(sdk.Coins, len(networkFee))
	// Determine the required fees by multiplying each required minimum gas
	// price by the gas limit, where fee = ceil(minGasPrice * gasLimit).
	glDec := sdk.NewDec(int64(gasLimit))
	for i, gp := range networkFee {
		fee := gp.Amount.Mul(glDec)
		requiredGlobalFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
	}

	return requiredGlobalFees.Sort()
}

func (s *IntegrationTestSuite) TestGetTxFeeRequired() {
	// create global fee params
	globalfeeParamsEmpty := &globfeetypes.Params{MinimumGasPrices: []sdk.DecCoin{}}

	// setup tests with default global fee i.e. "0uxion" and empty local min gas prices
	feeDecorator, _ := s.SetupTestGlobalFeeStoreAndMinGasPrice([]sdk.DecCoin{}, globalfeeParamsEmpty, noBondDenom)

	// set a subspace that doesn't have the stakingtypes.KeyBondDenom key registred
	//feeDecorator.StakingSubspace = s.app.GetSubspace(globfeetypes.ModuleName)

	// check that an error is returned when staking bond denom is empty
	_, err := feeDecorator.GetTxFeeRequired(s.ctx, nil)
	s.Require().Equal(err.Error(), "empty staking bond denomination")

	// set non-zero local min gas prices
	localMinGasPrices := sdk.NewCoins(sdk.NewCoin("uxion", sdk.NewInt(1)))

	// setup tests with non-empty local min gas prices
	feeDecorator, _ = s.SetupTestGlobalFeeStoreAndMinGasPrice(
		sdk.NewDecCoinsFromCoins(localMinGasPrices...),
		globalfeeParamsEmpty,
		bondDenom,
	)

	// mock tx data
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}

	//privfee, accNums, accSeqs := []cryptotypes.PrivKey{priv2}, []uint64{0}, []uint64{0}
	s.Require().NoError(s.txBuilder.SetMsgs(testdata.NewTestMsg(addr1)))
	s.txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("uxion", sdk.ZeroInt())))

	s.txBuilder.SetGasLimit(uint64(1))
	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)

	// check that the required fees returned in CheckTx mode are equal to
	// local min gas prices since they're greater than the default global fee values.
	s.Require().True(s.ctx.IsCheckTx())
	res, err := feeDecorator.GetTxFeeRequired(s.ctx, tx)
	s.Require().True(res.IsEqual(localMinGasPrices))
	s.Require().NoError(err)

	// check that the global fee is returned in DeliverTx mode.
	globalFee, err := feeDecorator.GetGlobalFee(s.ctx, tx)
	s.Require().NoError(err)

	ctx := s.ctx.WithIsCheckTx(false)
	res, err = feeDecorator.GetTxFeeRequired(ctx, tx)
	s.Require().NoError(err)
	s.Require().True(res.IsEqual(globalFee))
}
