package wasmbinding_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	"github.com/golang-jwt/jwt/v5"
	proto "github.com/golang/protobuf/proto" //nolint:staticcheck // we're intentionally using this deprecated package to be compatible with cosmos protos
	jwk "github.com/lestrrat-go/jwx/jwk"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	xionapp "github.com/burnt-labs/xion/app"
	wasmbinding "github.com/burnt-labs/xion/wasmbindings"
	jwkMsgServer "github.com/burnt-labs/xion/x/jwk/keeper"
	jwktypes "github.com/burnt-labs/xion/x/jwk/types"
	xiontypes "github.com/burnt-labs/xion/x/xion/types"
)

type StargateTestSuite struct {
	suite.Suite

	ctx sdk.Context
	app *xionapp.WasmApp
}

var admin = "cosmos1e2fuwe3uhq8zd9nkkk876nawrwdulgv4cxkq74"

func (suite *StargateTestSuite) SetupTest() {
	suite.app = xionapp.Setup(suite.T())
	suite.ctx = suite.app.NewContext(true).WithBlockTime(time.Now())
	suite.app.Configurator()
}

func TestStargateTestSuite(t *testing.T) {
	suite.Run(t, new(StargateTestSuite))
}

func SetUpAudience(suite *StargateTestSuite) {
	privKey, err := wasmbinding.SetupKeys()
	suite.Require().NoError(err)
	jwkPrivKey, err := jwk.New(privKey)
	suite.Require().NoError(err)
	pubKey, err := jwkPrivKey.PublicKey()
	suite.NoError(err)
	err = pubKey.Set("alg", "RS256")
	suite.NoError(err)
	pubKeyJSON, err := json.Marshal(pubKey)
	suite.NoError(err)
	msgServer := jwkMsgServer.NewMsgServerImpl(suite.app.JwkKeeper)
	sum := sha256.Sum256([]byte("test-aud"))
	_, err = msgServer.CreateAudienceClaim(suite.ctx, &jwktypes.MsgCreateAudienceClaim{
		Admin:   admin,
		AudHash: sum[:],
	})
	suite.NoError(err)
	_, err = msgServer.CreateAudience(suite.ctx, &jwktypes.MsgCreateAudience{
		Admin: admin,
		Aud:   "test-aud",
		Key:   string(pubKeyJSON),
	})
	suite.NoError(err)
}

func (suite *StargateTestSuite) TestWebauthNStargateQuerier() {
	testCases := []struct {
		name                   string
		testSetup              func()
		path                   string
		requestData            func() []byte
		responseProtoStruct    proto.Message
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
				expJSONResp, err := wasmbinding.ConvertProtoToJSONMarshal(tc.responseProtoStruct, expectedResponse, suite.app.AppCodec())
				suite.Require().NoError(err)
				suite.Require().Equal(expJSONResp, stargateResponse)
			}

			suite.Require().NoError(err)

			protoResponse := tc.responseProtoStruct

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

func (suite *StargateTestSuite) TestJWKStargateQuerier() {
	privKey, err := wasmbinding.SetupKeys()
	suite.Require().NoError(err)
	jwkPrivKey, err := jwk.New(privKey)
	suite.Require().NoError(err)
	publicKey, err := jwkPrivKey.PublicKey()
	suite.NoError(err)
	err = publicKey.Set("alg", "RS256")
	suite.NoError(err)
	publicKeyJSON, err := json.Marshal(publicKey)
	suite.NoError(err)

	testCases := []struct {
		name                   string
		testSetup              func()
		path                   string
		requestData            func() []byte
		responseProtoStruct    proto.Message
		expectedQuerierError   bool
		expectedUnMarshalError bool
		resendRequest          bool
		checkResponseStruct    bool
	}{
		{
			name: "JWKAudience",
			path: "/xion.jwk.v1.Query/Audience",
			requestData: func() []byte {
				bz, err := proto.Marshal(&jwktypes.QueryGetAudienceRequest{
					Aud: "test-aud",
				})
				suite.Require().NoError(err)
				return bz
			},
			testSetup: func() {
				SetUpAudience(suite)
			},
			responseProtoStruct: &jwktypes.QueryGetAudienceResponse{
				Audience: jwktypes.Audience{
					Admin: admin,
					Aud:   "test-aud",
					Key:   string(publicKeyJSON),
				},
			},
		},
		{
			name: "JWKAllAudience",
			path: "/xion.jwk.v1.Query/AudienceAll",
			requestData: func() []byte {
				bz, err := proto.Marshal(&jwktypes.QueryAllAudienceRequest{
					Pagination: &query.PageRequest{
						CountTotal: true,
					},
				})
				suite.Require().NoError(err)
				return bz
			},
			testSetup: func() {
				SetUpAudience(suite)
			},
			responseProtoStruct: &jwktypes.QueryAllAudienceResponse{
				Audience: []jwktypes.Audience{
					{
						Admin: admin,
						Aud:   "test-aud",
						Key:   string(publicKeyJSON),
					},
				},
				Pagination: &query.PageResponse{
					Total: 1,
				},
			},
		},
		{
			name: "JWKValidateJWT",
			path: "/xion.jwk.v1.Query/ValidateJWT",
			requestData: func() []byte {
				now := time.Now()
				fiveAgo := now.Add(-time.Second * 5)
				inFive := now.Add(time.Minute * 5)
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
					"iss":              "test-aud",
					"sub":              "subject",
					"aud":              "test-aud",
					"exp":              inFive.Unix(),
					"nbf":              fiveAgo.Unix(),
					"iat":              fiveAgo.Unix(),
					"transaction_hash": "test-tx-hash",
				})
				signedToken, err := token.SignedString(privKey)
				suite.Require().NoError(err)
				suite.NotEmpty(signedToken)
				bz, err := proto.Marshal(&jwktypes.QueryValidateJWTRequest{
					Aud:      "test-aud",
					Sub:      "subject",
					SigBytes: signedToken,
				})
				suite.Require().NoError(err)
				return bz
			},
			testSetup: func() {
				SetUpAudience(suite)
			},
			responseProtoStruct: &jwktypes.QueryValidateJWTResponse{},
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
				suite.NoError(err)
				expJSONResp, err := wasmbinding.ConvertProtoToJSONMarshal(tc.responseProtoStruct, expectedResponse, suite.app.AppCodec())
				suite.Require().NoError(err)
				suite.Require().Equal(expJSONResp, stargateResponse)
			}

			suite.Require().NoError(err)

			protoResponse := tc.responseProtoStruct

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

func createAuthzGrants(suite *StargateTestSuite) {
	authzKeeper := suite.app.AuthzKeeper

	authorization, err := types.NewAnyWithValue(&authztypes.GenericAuthorization{
		Msg: "/" + string(proto.MessageReflect(&banktypes.MsgSend{}).Descriptor().FullName()),
	})
	suite.NoError(err)
	grantMsg := &authztypes.MsgGrant{
		Granter: "cosmos1ynu5zu77pjyuj9ueepqw0vveq2fpd2xp6jgx0s7m2rlcguxldxvqag9wce",
		Grantee: "cosmos1e2fuwe3uhq8zd9nkkk876nawrwdulgv4cxkq74",
		Grant: authztypes.Grant{
			Authorization: authorization,
		},
	}
	_, err = authzKeeper.Grant(suite.ctx, grantMsg)
	suite.NoError(err)
}

func (suite *StargateTestSuite) TestAuthzStargateQuerier() {
	testCases := []struct {
		name                   string
		testSetup              func()
		path                   string
		requestData            func() []byte
		responseProtoStruct    func() proto.Message
		expectedQuerierError   bool
		expectedUnMarshalError bool
		resendRequest          bool
		checkResponseStruct    bool
	}{
		{
			name: "AuthzGrants",
			path: "/cosmos.authz.v1beta1.Query/Grants",
			testSetup: func() {
				createAuthzGrants(suite)
			},
			responseProtoStruct: func() proto.Message {
				authorization, err := types.NewAnyWithValue(&authztypes.GenericAuthorization{
					Msg: "/" + string(proto.MessageReflect(&banktypes.MsgSend{}).Descriptor().FullName()),
				})
				suite.NoError(err)
				return &authztypes.QueryGrantsResponse{
					Grants: []*authztypes.Grant{
						{Authorization: authorization},
					},
					Pagination: &query.PageResponse{
						Total:   1,
						NextKey: nil,
					},
				}
			},
			requestData: func() []byte {
				bz, err := proto.Marshal(&authztypes.QueryGrantsRequest{
					Granter: "cosmos1ynu5zu77pjyuj9ueepqw0vveq2fpd2xp6jgx0s7m2rlcguxldxvqag9wce",
					Grantee: "cosmos1e2fuwe3uhq8zd9nkkk876nawrwdulgv4cxkq74",
				})
				if err != nil {
					panic(err)
				}
				return bz
			},
			checkResponseStruct: true,
		},
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
				expectedResponse, err := proto.Marshal(tc.responseProtoStruct())
				suite.NoError(err)
				expJSONResp, err := wasmbinding.ConvertProtoToJSONMarshal(&authztypes.QueryGrantsResponse{}, expectedResponse, suite.app.AppCodec())
				suite.Require().NoError(err)
				suite.Require().Equal(expJSONResp, stargateResponse)
			}

			suite.Require().NoError(err)

			protoResponse := tc.responseProtoStruct()

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
