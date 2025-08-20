package cli

import (
	"context"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"

	aatypes "github.com/burnt-labs/abstract-account/x/abstractaccount/types"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	// grpc options
	"google.golang.org/grpc"
)

// authQueryClientStub implements authtypes.QueryClient minimally for getSignerOfTx tests.
type authQueryClientStub struct {
	resp *authtypes.QueryAccountResponse
	err  error
}

func (a *authQueryClientStub) Account(ctx context.Context, in *authtypes.QueryAccountRequest, opts ...grpc.CallOption) (*authtypes.QueryAccountResponse, error) {
	return a.resp, a.err
}

// The remaining methods satisfy the interface but are unused in tests.
func (a *authQueryClientStub) Accounts(ctx context.Context, in *authtypes.QueryAccountsRequest, opts ...grpc.CallOption) (*authtypes.QueryAccountsResponse, error) {
	return nil, fmt.Errorf("unimplemented")
}
func (a *authQueryClientStub) Params(ctx context.Context, in *authtypes.QueryParamsRequest, opts ...grpc.CallOption) (*authtypes.QueryParamsResponse, error) {
	return nil, fmt.Errorf("unimplemented")
}
func (a *authQueryClientStub) ModuleAccounts(ctx context.Context, in *authtypes.QueryModuleAccountsRequest, opts ...grpc.CallOption) (*authtypes.QueryModuleAccountsResponse, error) {
	return nil, fmt.Errorf("unimplemented")
}
func (a *authQueryClientStub) ModuleAccountByName(ctx context.Context, in *authtypes.QueryModuleAccountByNameRequest, opts ...grpc.CallOption) (*authtypes.QueryModuleAccountByNameResponse, error) {
	return nil, fmt.Errorf("unimplemented")
}
func (a *authQueryClientStub) Bech32Prefix(ctx context.Context, in *authtypes.Bech32PrefixRequest, opts ...grpc.CallOption) (*authtypes.Bech32PrefixResponse, error) {
	return nil, fmt.Errorf("unimplemented")
}
func (a *authQueryClientStub) AddressBytesToString(ctx context.Context, in *authtypes.AddressBytesToStringRequest, opts ...grpc.CallOption) (*authtypes.AddressBytesToStringResponse, error) {
	return nil, fmt.Errorf("unimplemented")
}
func (a *authQueryClientStub) AddressStringToBytes(ctx context.Context, in *authtypes.AddressStringToBytesRequest, opts ...grpc.CallOption) (*authtypes.AddressStringToBytesResponse, error) {
	return nil, fmt.Errorf("unimplemented")
}
func (a *authQueryClientStub) AccountInfo(ctx context.Context, in *authtypes.QueryAccountInfoRequest, opts ...grpc.CallOption) (*authtypes.QueryAccountInfoResponse, error) {
	return nil, fmt.Errorf("unimplemented")
}
func (a *authQueryClientStub) AccountAddressByID(ctx context.Context, in *authtypes.QueryAccountAddressByIDRequest, opts ...grpc.CallOption) (*authtypes.QueryAccountAddressByIDResponse, error) {
	return nil, fmt.Errorf("unimplemented")
}

func TestGetSignerOfTx(t *testing.T) {
	addr := sdk.AccAddress([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19})

	mkResp := func(any *cdctypes.Any) *authtypes.QueryAccountResponse {
		return &authtypes.QueryAccountResponse{Account: any}
	}

	// wrong type any (base account)
	base := &authtypes.BaseAccount{}
	baseBz, _ := proto.Marshal(base)
	baseAny := &cdctypes.Any{TypeUrl: typeURL(base), Value: baseBz}

	// abstract account good
	aa := &aatypes.AbstractAccount{}
	aaBz, _ := proto.Marshal(aa)
	aaAny := &cdctypes.Any{TypeUrl: typeURL((*aatypes.AbstractAccount)(nil)), Value: aaBz}

	// abstract account bad bytes
	badAny := &cdctypes.Any{TypeUrl: typeURL((*aatypes.AbstractAccount)(nil)), Value: []byte("badbytes")}

	tests := []struct {
		name      string
		client    authtypes.QueryClient
		expectErr string
		ok        bool
	}{
		{"query error", &authQueryClientStub{err: fmt.Errorf("boom")}, "boom", false},
		{"wrong type", &authQueryClientStub{resp: mkResp(baseAny)}, "not an AbstractAccount", false},
		{"unmarshal", &authQueryClientStub{resp: mkResp(badAny)}, "unexpected EOF", false},
		{"success", &authQueryClientStub{resp: mkResp(aaAny)}, "", true},
	}

	for _, tc := range tests {
		acc, err := getSignerOfTx(tc.client, addr)
		if tc.ok {
			require.NoError(t, err, tc.name)
			require.NotNil(t, acc)
		} else {
			require.Error(t, err, tc.name)
			if tc.expectErr != "" {
				require.Contains(t, err.Error(), tc.expectErr)
			}
		}
	}
}
