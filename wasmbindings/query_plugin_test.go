package wasmbinding_test

import (
	"fmt"
	"testing"
	"time"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	xionapp "github.com/burnt-labs/xion/app"
	wasmbinding "github.com/burnt-labs/xion/wasmbindings"
	xiontypes "github.com/burnt-labs/xion/x/xion/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/golang/protobuf/proto" //nolint:staticcheck // we're intentionally using this deprecated package to be compatible with cosmos protos

	"github.com/stretchr/testify/suite"
)

type StargateTestSuite struct {
	suite.Suite

	ctx sdk.Context
	app *xionapp.WasmApp
}

func (suite *StargateTestSuite) SetupTest() {
	suite.app = xionapp.Setup(suite.T())
	suite.ctx = suite.app.BaseApp.NewContext(false, tmproto.Header{Height: 1, ChainID: "xion-1", Time: time.Now().UTC()})
}

func TestStargateTestSuite(t *testing.T) {
	suite.Run(t, new(StargateTestSuite))
}

func (suite *StargateTestSuite) TestStargateQuerier() {
	testCases := []struct {
		name                   string
		testSetup              func()
		path                   string
		requestData            func() []byte
		responseProtoStruct    codec.ProtoMarshaler
		expectedQuerierError   bool
		expectedUnMarshalError bool
		resendRequest          bool
		checkResponseStruct    bool
	}{
		{
			name: "WebAuthNVerifyRegister",
			path: "/xion.v1.Query/WebAuthNVerifyRegister",
			requestData: func() []byte {

				bz, err := proto.Marshal(&xiontypes.QueryWebAuthNVerifyRegisterRequest{
					Addr:      "cosmos1fl48vsnmsdzcv85q5d2q4z5ajdha8yu34mf0eh",
					Rp:        "https://xion-dapp-example-git-feat-faceid-burntfinance.vercel.app",
					Challenge: "dGVzdC1jaGFsbGVuZ2U",
					Data:      []byte(`{"type":"public-key","id":"8ofgr8BFk_HalAiGi6tBxAJez4d7lq0iVi7Gi7_SN5E","rawId":"8ofgr8BFk_HalAiGi6tBxAJez4d7lq0iVi7Gi7_SN5E","authenticatorAttachment":"platform","response":{"clientDataJSON":"eyJ0eXBlIjoid2ViYXV0aG4uY3JlYXRlIiwiY2hhbGxlbmdlIjoiZEdWemRDMWphR0ZzYkdWdVoyVSIsIm9yaWdpbiI6Imh0dHBzOi8veGlvbi1kYXBwLWV4YW1wbGUtZ2l0LWZlYXQtZmFjZWlkLWJ1cm50ZmluYW5jZS52ZXJjZWwuYXBwIiwiY3Jvc3NPcmlnaW4iOmZhbHNlfQ","attestationObject":"o2NmbXRkbm9uZWdhdHRTdG10oGhhdXRoRGF0YViksGMBiDcEppiMfxQ10TPCe2-FaKrLeTkvpzxczngTMw1BAAAAAK3OAAI1vMYKZIsLJfHwVQMAIPKH4K_ARZPx2pQIhourQcQCXs-He5atIlYuxou_0jeRpQECAyYgASFYILbJBGn3gOiKXecsLGRvLfOEVic9KiQJ55Tbz5BBNFffIlggSrIMryGmxIEl9p1z0uXvuPnH-T7GMeF_hrwJS6bWMKQ","transports":["internal"]},"clientExtensionResults":{}}`),
				})
				suite.Require().NoError(err)
				return bz
			},
			responseProtoStruct: &xiontypes.QueryWebAuthNVerifyRegisterResponse{},
		},

		// TODO: errors in wrong query in state machine
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest()
			if tc.testSetup != nil {
				tc.testSetup()
			}

			stargateQuerier := wasmbinding.StargateQuerier(*suite.app.GRPCQueryRouter(), suite.app.AppCodec())
			stargateRequest := &wasmvmtypes.StargateQuery{
				Path: tc.path,
				Data: tc.requestData(),
			}
			stargateResponse, err := stargateQuerier(suite.ctx, stargateRequest)
			if tc.expectedQuerierError {
				suite.Require().Error(err)
				return
			}
			if tc.checkResponseStruct {
				expectedResponse, err := proto.Marshal(tc.responseProtoStruct)
				suite.Require().NoError(err)
				expJsonResp, err := wasmbinding.ConvertProtoToJSONMarshal(tc.responseProtoStruct, expectedResponse, suite.app.AppCodec())
				suite.Require().NoError(err)
				suite.Require().Equal(expJsonResp, stargateResponse)
			}

			suite.Require().NoError(err)

			protoResponse, ok := tc.responseProtoStruct.(proto.Message)
			suite.Require().True(ok)

			// test correctness by unmarshalling json response into proto struct
			err = suite.app.AppCodec().UnmarshalJSON(stargateResponse, protoResponse)
			if tc.expectedUnMarshalError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(protoResponse)
			}

			if tc.resendRequest {
				stargateQuerier = wasmbinding.StargateQuerier(*suite.app.GRPCQueryRouter(), suite.app.AppCodec())
				stargateRequest = &wasmvmtypes.StargateQuery{
					Path: tc.path,
					Data: tc.requestData(),
				}
				resendResponse, err := stargateQuerier(suite.ctx, stargateRequest)
				suite.Require().NoError(err)
				suite.Require().Equal(stargateResponse, resendResponse)
			}
		})
	}
}
