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
					Addr:      "xion1ynu5zu77pjyuj9ueepqw0vveq2fpd2xp6jgx0s7m2rlcguxldxvq8akzpz",
					Rp:        "https://xion-dapp-example-git-feat-faceid-burntfinance.vercel.app",
					Challenge: "eGlvbjF5bnU1enU3N3BqeXVqOXVlZXBxdzB2dmVxMmZwZDJ4cDZqZ3gwczdtMnJsY2d1eGxkeHZxOGFrenB6",
					Data:      []byte(`{"type":"public-key","id":"y0zUQQMndks_wh4naaNRL_PZJOFgwusbO2LYVVhHvZg","rawId":"y0zUQQMndks_wh4naaNRL_PZJOFgwusbO2LYVVhHvZg","authenticatorAttachment":"platform","response":{"clientDataJSON":"eyJ0eXBlIjoid2ViYXV0aG4uY3JlYXRlIiwiY2hhbGxlbmdlIjoiZUdsdmJqRjViblUxZW5VM04zQnFlWFZxT1hWbFpYQnhkekIyZG1WeE1tWndaREo0Y0RacVozZ3djemR0TW5Kc1kyZDFlR3hrZUhaeE9HRnJlbkI2Iiwib3JpZ2luIjoiaHR0cHM6Ly94aW9uLWRhcHAtZXhhbXBsZS1naXQtZmVhdC1mYWNlaWQtYnVybnRmaW5hbmNlLnZlcmNlbC5hcHAiLCJjcm9zc09yaWdpbiI6ZmFsc2V9","attestationObject":"o2NmbXRkbm9uZWdhdHRTdG10oGhhdXRoRGF0YViksGMBiDcEppiMfxQ10TPCe2-FaKrLeTkvpzxczngTMw1BAAAAAK3OAAI1vMYKZIsLJfHwVQMAIMtM1EEDJ3ZLP8IeJ2mjUS_z2SThYMLrGzti2FVYR72YpQECAyYgASFYIP00VX-FAxs2eClWYI-wgmmBwSt5qPduwIC6JqaVeEwwIlggzFwyKvRH0UvJTLzZQa0fKPr0gCdbT2A-nuNa0Jcp1_k","transports":["internal"]},"clientExtensionResults":{}}`),
				})
				suite.Require().NoError(err)
				return bz
			},
			responseProtoStruct: &xiontypes.QueryWebAuthNVerifyRegisterResponse{},
		},
		{
			name: "WebAuthNVerifyAuthenticate",
			path: "/xion.v1.Query/WebAuthNVerifyAuthenticate",
			requestData: func() []byte {

				bz, err := proto.Marshal(&xiontypes.QueryWebAuthNVerifyAuthenticateRequest{
					Addr:       "xion1ynu5zu77pjyuj9ueepqw0vveq2fpd2xp6jgx0s7m2rlcguxldxvq8akzpz",
					Challenge:  "eGlvbjF5bnU1enU3N3BqeXVqOXVlZXBxdzB2dmVxMmZwZDJ4cDZqZ3gwczdtMnJsY2d1eGxkeHZxOGFrenB6",
					Rp:         "https://xion-dapp-example-git-feat-faceid-burntfinance.vercel.app",
					Credential: []byte(`{"ID":"y0zUQQMndks/wh4naaNRL/PZJOFgwusbO2LYVVhHvZg=","PublicKey":"pQECAyYgASFYIP00VX+FAxs2eClWYI+wgmmBwSt5qPduwIC6JqaVeEwwIlggzFwyKvRH0UvJTLzZQa0fKPr0gCdbT2A+nuNa0Jcp1/k=","AttestationType":"none","Transport":["internal"],"Flags":{"UserPresent":true,"UserVerified":false,"BackupEligible":false,"BackupState":false},"Authenticator":{"AAGUID":"rc4AAjW8xgpkiwsl8fBVAw==","SignCount":0,"CloneWarning":false,"Attachment":"platform"}}`),
					Data:       []byte(`{"type":"public-key","id":"y0zUQQMndks_wh4naaNRL_PZJOFgwusbO2LYVVhHvZg","rawId":"y0zUQQMndks_wh4naaNRL_PZJOFgwusbO2LYVVhHvZg","authenticatorAttachment":"platform","response":{"clientDataJSON":"eyJ0eXBlIjoid2ViYXV0aG4uZ2V0IiwiY2hhbGxlbmdlIjoiZUdsdmJqRjViblUxZW5VM04zQnFlWFZxT1hWbFpYQnhkekIyZG1WeE1tWndaREo0Y0RacVozZ3djemR0TW5Kc1kyZDFlR3hrZUhaeE9HRnJlbkI2Iiwib3JpZ2luIjoiaHR0cHM6Ly94aW9uLWRhcHAtZXhhbXBsZS1naXQtZmVhdC1mYWNlaWQtYnVybnRmaW5hbmNlLnZlcmNlbC5hcHAiLCJjcm9zc09yaWdpbiI6ZmFsc2UsIm90aGVyX2tleXNfY2FuX2JlX2FkZGVkX2hlcmUiOiJkbyBub3QgY29tcGFyZSBjbGllbnREYXRhSlNPTiBhZ2FpbnN0IGEgdGVtcGxhdGUuIFNlZSBodHRwczovL2dvby5nbC95YWJQZXgifQ","authenticatorData":"sGMBiDcEppiMfxQ10TPCe2-FaKrLeTkvpzxczngTMw0BAAAAAA","signature":"MEUCIQC7pTqOWJ5zm40pJOr9W6Bi3xW27fs07mfr6LF_KSOUhgIgBDC3o0P1-7XjMsVFMLtI1a94i1-lkxwYN0W8T_bMxKs","userHandle":"eGlvbjF5bnU1enU3N3BqeXVqOXVlZXBxdzB2dmVxMmZwZDJ4cDZqZ3gwczdtMnJsY2d1eGxkeHZxOGFrenB6"},"clientExtensionResults":{}}`),
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
