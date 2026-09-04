package types_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

type capturingQueryClient struct {
	request *types.QueryAccountAddressRequest
}

func (c *capturingQueryClient) Params(
	context.Context,
	*types.QueryParamsRequest,
	...grpc.CallOption,
) (*types.QueryParamsResponse, error) {
	return &types.QueryParamsResponse{}, nil
}

func (c *capturingQueryClient) AccountAddress(
	_ context.Context,
	req *types.QueryAccountAddressRequest,
	_ ...grpc.CallOption,
) (*types.QueryAccountAddressResponse, error) {
	c.request = req
	return &types.QueryAccountAddressResponse{}, nil
}

func TestAccountAddressGatewayPopulatesSaltQueryParameter(t *testing.T) {
	mux := runtime.NewServeMux()
	queryClient := &capturingQueryClient{}
	require.NoError(t, types.RegisterQueryHandlerClient(context.Background(), mux, queryClient))

	salt := []byte("account namespace")
	encodedSalt := url.QueryEscape(base64.StdEncoding.EncodeToString(salt))
	req := httptest.NewRequest(
		http.MethodGet,
		"/abstractaccount/v1/account_address/xion1sender?salt="+encodedSalt,
		nil,
	)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	require.Equal(t, 200, res.Code, res.Body.String())
	require.NotNil(t, queryClient.request)
	require.Equal(t, "xion1sender", queryClient.request.Sender)
	require.Equal(t, salt, queryClient.request.Salt)
}
