package antetest

import (
	"context"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/params/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	xionapp "github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/x/globalfee"
	xionfeeante "github.com/burnt-labs/xion/x/globalfee/ante"
	globfeetypes "github.com/burnt-labs/xion/x/globalfee/types"
)

type IntegrationTestSuite struct {
	suite.Suite

	app       *xionapp.WasmApp
	ctx       sdk.Context
	clientCtx client.Context
	txBuilder client.TxBuilder
}

var testBondDenom = "uxion"

func (s *IntegrationTestSuite) SetupTest() {
	app := xionapp.Setup(s.T())
	ctx := app.BaseApp.NewContext(false)

	encodingConfig := testutil.MakeTestEncodingConfig()
	encodingConfig.Amino.RegisterConcrete(&testdata.TestMsg{}, "testdata.TestMsg", nil)
	testdata.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	s.app = app
	s.ctx = ctx
	s.clientCtx = client.Context{}.WithTxConfig(encodingConfig.TxConfig)
}

func bondDenom(_ sdk.Context) string {
	return testBondDenom
}

func noBondDenom(_ sdk.Context) string {
	return ""
}

func (s *IntegrationTestSuite) SetupTestGlobalFeeStoreAndMinGasPrice(minGasPrice []sdk.DecCoin, globalFeeParams *globfeetypes.Params, bondDenom func(sdk.Context) string) (xionfeeante.FeeDecorator, sdk.AnteHandler) {
	subspace := s.app.GetSubspace(globalfee.ModuleName)
	subspace.SetParamSet(s.ctx, globalFeeParams)
	s.ctx = s.ctx.WithMinGasPrices(minGasPrice).WithIsCheckTx(true)

	// set staking params
	stakingParam := stakingtypes.DefaultParams()
	stakingParam.BondDenom = testBondDenom

	// build fee decorator
	feeDecorator := xionfeeante.NewFeeDecorator(subspace, bondDenom)

	// chain fee decorator to antehandler
	antehandler := sdk.ChainAnteDecorators(feeDecorator)

	return feeDecorator, antehandler
}

// SetupTestStakingSubspace sets uatom as bond denom for the fee tests.
func (s *IntegrationTestSuite) SetupTestStakingSubspace(params stakingtypes.Params) types.Subspace {
	s.app.GetSubspace(stakingtypes.ModuleName).SetParamSet(s.ctx, &params)
	return s.app.GetSubspace(stakingtypes.ModuleName)
}

func (s *IntegrationTestSuite) CreateTestTx(privs []cryptotypes.PrivKey, accNums []uint64, accSeqs []uint64, chainID string) (xauthsigning.Tx, error) {
	var sigsV2 []signing.SignatureV2
	for i, priv := range privs {
		sigV2 := signing.SignatureV2{
			PubKey: priv.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
				Signature: nil,
			},
			Sequence: accSeqs[i],
		}

		sigsV2 = append(sigsV2, sigV2)
	}

	if err := s.txBuilder.SetSignatures(sigsV2...); err != nil {
		return nil, err
	}

	sigsV2 = []signing.SignatureV2{}
	for i, priv := range privs {
		signerData := xauthsigning.SignerData{
			ChainID:       chainID,
			AccountNumber: accNums[i],
			Sequence:      accSeqs[i],
		}
		sigV2, err := tx.SignWithPrivKey(
			context.Background(),
			signing.SignMode_SIGN_MODE_DIRECT,
			signerData,
			s.txBuilder,
			priv,
			s.clientCtx.TxConfig,
			accSeqs[i],
		)
		if err != nil {
			return nil, err
		}

		sigsV2 = append(sigsV2, sigV2)
	}

	if err := s.txBuilder.SetSignatures(sigsV2...); err != nil {
		return nil, err
	}

	return s.txBuilder.GetTx(), nil
}
