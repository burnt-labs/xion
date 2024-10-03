package wasmbinding_test

import (
	"fmt"
	"testing"
	"time"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	xionapp "github.com/burnt-labs/xion/app"
	wasmbinding "github.com/burnt-labs/xion/wasmbindings"
)

type GrpcTestSuite struct {
	suite.Suite

	ctx sdk.Context
	app *xionapp.WasmApp
}

func (suite *GrpcTestSuite) SetupTest() {
	suite.app = xionapp.Setup(suite.T())
	suite.ctx = suite.app.NewContext(true).WithBlockTime(time.Now())
	suite.app.Configurator()
}

func TestGrpcTestSuite(t *testing.T) {
	suite.Run(t, new(GrpcTestSuite))
}

func createAuthzGrantsGrpc(suite *GrpcTestSuite) {
	authzKeeper := suite.app.AuthzKeeper

	authorization, err := types.NewAnyWithValue(&authztypes.GenericAuthorization{
		Msg: "/" + proto.MessageName(&banktypes.MsgSend{}),
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

func (suite *GrpcTestSuite) TestAuthzGrpcQuerier() {
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
			name: "AuthzGrantsGrpc",
			path: "/cosmos.authz.v1beta1.Query/Grants",
			testSetup: func() {
				createAuthzGrantsGrpc(suite)
			},
			responseProtoStruct: func() proto.Message {
				authorization, err := types.NewAnyWithValue(&authztypes.GenericAuthorization{
					Msg: "/" + proto.MessageName(&banktypes.MsgSend{}),
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

			grpcQuerier := wasmbinding.GrpcQuerier(*suite.app.GRPCQueryRouter())
			grpcRequest := &wasmvmtypes.GrpcQuery{
				Path: tc.path,
				Data: tc.requestData(),
			}
			grpcResponse, err := grpcQuerier(suite.ctx, grpcRequest)
			if tc.expectedQuerierError {
				suite.Require().Error(err)
				return
			}
			suite.Require().NoError(err)

			if tc.checkResponseStruct {
				reponseBz, err := proto.Marshal(grpcResponse)
				suite.Require().NoError(err)
				expectedResponseBz, err := proto.Marshal(tc.responseProtoStruct())
				suite.Require().NoError(err)
				suite.Require().Equal(reponseBz, expectedResponseBz)
			}

			if tc.resendRequest {
				grpcQuerier = wasmbinding.GrpcQuerier(*suite.app.GRPCQueryRouter())
				grpcRequest = &wasmvmtypes.GrpcQuery{
					Path: tc.path,
					Data: tc.requestData(),
				}
				resendResponse, err := grpcQuerier(suite.ctx, grpcRequest)
				suite.Require().NoError(err)
				suite.Require().Equal(grpcResponse, resendResponse)
			}
		})
	}
}
