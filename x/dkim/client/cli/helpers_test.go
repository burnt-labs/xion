package cli_test

import (
	"context"

	"google.golang.org/grpc"

	"github.com/burnt-labs/xion/x/dkim/types"
)

// Mock QueryClient for testing
type MockQueryClient struct {
	types.QueryClient
	dkimPubKeyFunc  func(ctx context.Context, req *types.QueryDkimPubKeyRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeyResponse, error)
	dkimPubKeysFunc func(ctx context.Context, req *types.QueryDkimPubKeysRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeysResponse, error)
}

func (m *MockQueryClient) DkimPubKey(ctx context.Context, req *types.QueryDkimPubKeyRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeyResponse, error) {
	if m.dkimPubKeyFunc != nil {
		return m.dkimPubKeyFunc(ctx, req, opts...)
	}
	return &types.QueryDkimPubKeyResponse{
		DkimPubKey: &types.DkimPubKey{
			Domain:   req.Domain,
			Selector: req.Selector,
		},
	}, nil
}

func (m *MockQueryClient) DkimPubKeys(ctx context.Context, req *types.QueryDkimPubKeysRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeysResponse, error) {
	if m.dkimPubKeysFunc != nil {
		return m.dkimPubKeysFunc(ctx, req, opts...)
	}
	return &types.QueryDkimPubKeysResponse{
		DkimPubKeys: []*types.DkimPubKey{
			{
				Domain:       req.Domain,
				Selector:     req.Selector,
				PoseidonHash: req.PoseidonHash,
			},
		},
	}, nil
}
