package cli_test

/*
// Mock QueryClient for testing
type mockQueryClient struct {
	types.QueryClient
	paramsFunc func(ctx context.Context, req *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error)
}

func (m *mockQueryClient) Params(ctx context.Context, req *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error) {
	if m.paramsFunc != nil {
		return m.paramsFunc(ctx, req, opts...)
	}
	params := types.DefaultParams()
	return &types.QueryParamsResponse{Params: &params}, nil
}

func TestQueryParams(t *testing.T) {
	mockClient := &mockQueryClient{}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	res, err := cli.QueryParams(mockClient, cmd)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Params)
}
*/
