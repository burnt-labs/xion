package types_test

import (
	"context"
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/burnt-labs/xion/x/xauthz/types"
)

// mockQueryProviderForAccepter implements types.QueryProvider for testing
type mockQueryProviderForAccepter struct {
	contractInfo *wasmtypes.ContractInfo
	err          error
}

func (m *mockQueryProviderForAccepter) QueryContractInfo(_ context.Context, _ string) (*wasmtypes.ContractInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.contractInfo, nil
}

var _ types.QueryProvider = (*mockQueryProviderForAccepter)(nil)

// mockAuthorization implements authz.Authorization for testing non-stateful path
type mockAuthorization struct {
	acceptResponse authz.AcceptResponse
	acceptErr      error
}

func (m *mockAuthorization) MsgTypeURL() string {
	return sdk.MsgTypeURL(&banktypes.MsgSend{})
}

func (m *mockAuthorization) Accept(_ context.Context, _ sdk.Msg) (authz.AcceptResponse, error) {
	return m.acceptResponse, m.acceptErr
}

func (m *mockAuthorization) ValidateBasic() error {
	return nil
}

func (m *mockAuthorization) Reset()         {}
func (m *mockAuthorization) String() string { return "mockAuthorization" }
func (m *mockAuthorization) ProtoMessage()  {}

var _ authz.Authorization = (*mockAuthorization)(nil)

func TestNewAccepter(t *testing.T) {
	t.Run("creates accepter with query provider", func(t *testing.T) {
		mock := &mockQueryProviderForAccepter{}
		accepter := types.NewAccepter(mock)
		require.NotNil(t, accepter)
	})

	t.Run("creates accepter with nil query provider", func(t *testing.T) {
		accepter := types.NewAccepter(nil)
		require.NotNil(t, accepter)
	})
}

func TestAuthzAccepter_Accept(t *testing.T) {
	validContract := "xion1qg5ega6dykkxc307y25pecuufrjkxkaggkkxh7nad0vhyhtuhw3sqaa3c5"

	t.Run("delegates to StatefulAccepter for CodeExecutionAuthorization", func(t *testing.T) {
		mock := &mockQueryProviderForAccepter{
			contractInfo: &wasmtypes.ContractInfo{CodeID: 1},
		}
		accepter := types.NewAccepter(mock)

		auth := types.NewCodeExecutionAuthorization([]uint64{1, 2, 3})
		msg := &wasmtypes.MsgExecuteContract{
			Contract: validContract,
		}

		resp, err := accepter.Accept(context.Background(), msg, auth)
		require.NoError(t, err)
		require.True(t, resp.Accept)
	})

	t.Run("falls back to authorization.Accept for non-stateful authorization", func(t *testing.T) {
		mock := &mockQueryProviderForAccepter{}
		accepter := types.NewAccepter(mock)

		auth := &mockAuthorization{
			acceptResponse: authz.AcceptResponse{Accept: true},
		}
		msg := &banktypes.MsgSend{}

		resp, err := accepter.Accept(context.Background(), msg, auth)
		require.NoError(t, err)
		require.True(t, resp.Accept)
	})

	t.Run("returns error from stateful accepter", func(t *testing.T) {
		mock := &mockQueryProviderForAccepter{
			contractInfo: &wasmtypes.ContractInfo{CodeID: 99},
		}
		accepter := types.NewAccepter(mock)

		auth := types.NewCodeExecutionAuthorization([]uint64{1, 2, 3})
		msg := &wasmtypes.MsgExecuteContract{
			Contract: validContract,
		}

		resp, err := accepter.Accept(context.Background(), msg, auth)
		require.Error(t, err)
		require.False(t, resp.Accept)
	})

	t.Run("returns error when query provider is nil", func(t *testing.T) {
		accepter := types.NewAccepter(nil)

		auth := types.NewCodeExecutionAuthorization([]uint64{1, 2, 3})
		msg := &wasmtypes.MsgExecuteContract{
			Contract: validContract,
		}

		resp, err := accepter.Accept(context.Background(), msg, auth)
		require.Error(t, err)
		require.Contains(t, err.Error(), "requires a QueryProvider")
		require.False(t, resp.Accept)
	})

	t.Run("returns error from fallback authorization", func(t *testing.T) {
		mock := &mockQueryProviderForAccepter{}
		accepter := types.NewAccepter(mock)

		auth := &mockAuthorization{
			acceptResponse: authz.AcceptResponse{Accept: false},
			acceptErr:      authz.ErrNoAuthorizationFound,
		}
		msg := &banktypes.MsgSend{}

		resp, err := accepter.Accept(context.Background(), msg, auth)
		require.Error(t, err)
		require.False(t, resp.Accept)
	})
}
