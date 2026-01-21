package types_test

import (
	"context"
	"fmt"
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/burnt-labs/xion/x/xauthz/types"
)

// mockQueryProvider implements types.QueryProvider for testing
type mockQueryProvider struct {
	contractInfo *wasmtypes.ContractInfo
	err          error
}

func (m *mockQueryProvider) QueryContractInfo(_ context.Context, _ string) (*wasmtypes.ContractInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.contractInfo, nil
}

var _ types.QueryProvider = (*mockQueryProvider)(nil)

func TestNewCodeExecutionAuthorization(t *testing.T) {
	t.Run("creates authorization with allowed code IDs", func(t *testing.T) {
		auth := types.NewCodeExecutionAuthorization([]uint64{1, 2, 3})
		require.NotNil(t, auth)
		require.Equal(t, []uint64{1, 2, 3}, auth.AllowedCodeIds)
	})

	t.Run("creates authorization with empty code IDs", func(t *testing.T) {
		auth := types.NewCodeExecutionAuthorization([]uint64{})
		require.NotNil(t, auth)
		require.Empty(t, auth.AllowedCodeIds)
	})

	t.Run("creates authorization with nil code IDs", func(t *testing.T) {
		auth := types.NewCodeExecutionAuthorization(nil)
		require.NotNil(t, auth)
		require.Nil(t, auth.AllowedCodeIds)
	})
}

func TestCodeExecutionAuthorization_MsgTypeURL(t *testing.T) {
	auth := types.NewCodeExecutionAuthorization([]uint64{1})
	msgTypeURL := auth.MsgTypeURL()
	require.Equal(t, sdk.MsgTypeURL(&wasmtypes.MsgExecuteContract{}), msgTypeURL)
}

func TestCodeExecutionAuthorization_ValidateBasic(t *testing.T) {
	t.Run("valid with allowed code IDs", func(t *testing.T) {
		auth := types.NewCodeExecutionAuthorization([]uint64{1, 2, 3})
		err := auth.ValidateBasic()
		require.NoError(t, err)
	})

	t.Run("invalid with empty code IDs", func(t *testing.T) {
		auth := types.NewCodeExecutionAuthorization([]uint64{})
		err := auth.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "allowed_code_ids cannot be empty")
	})

	t.Run("invalid with nil code IDs", func(t *testing.T) {
		auth := types.NewCodeExecutionAuthorization(nil)
		err := auth.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "allowed_code_ids cannot be empty")
	})
}

func TestCodeExecutionAuthorization_Accept(t *testing.T) {
	t.Run("returns error requiring stateful execution", func(t *testing.T) {
		auth := types.NewCodeExecutionAuthorization([]uint64{1})
		resp, err := auth.Accept(context.Background(), &wasmtypes.MsgExecuteContract{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "requires stateful execution")
		require.False(t, resp.Accept)
	})
}

func TestCodeExecutionAuthorization_AcceptWith(t *testing.T) {
	validContract := "xion1qg5ega6dykkxc307y25pecuufrjkxkaggkkxh7nad0vhyhtuhw3sqaa3c5"

	t.Run("accepts when code ID is in allowed list", func(t *testing.T) {
		auth := types.NewCodeExecutionAuthorization([]uint64{1, 2, 3})
		mock := &mockQueryProvider{
			contractInfo: &wasmtypes.ContractInfo{CodeID: 2},
		}

		msg := &wasmtypes.MsgExecuteContract{
			Contract: validContract,
		}

		resp, err := auth.AcceptWith(context.Background(), msg, mock)
		require.NoError(t, err)
		require.True(t, resp.Accept)
	})

	t.Run("rejects when code ID is not in allowed list", func(t *testing.T) {
		auth := types.NewCodeExecutionAuthorization([]uint64{1, 2, 3})
		mock := &mockQueryProvider{
			contractInfo: &wasmtypes.ContractInfo{CodeID: 99},
		}

		msg := &wasmtypes.MsgExecuteContract{
			Contract: validContract,
		}

		resp, err := auth.AcceptWith(context.Background(), msg, mock)
		require.Error(t, err)
		require.Contains(t, err.Error(), "contract code_id 99 not in allowed list")
		require.False(t, resp.Accept)
	})

	t.Run("returns error for wrong message type", func(t *testing.T) {
		auth := types.NewCodeExecutionAuthorization([]uint64{1})
		mock := &mockQueryProvider{}

		msg := &banktypes.MsgSend{}

		resp, err := auth.AcceptWith(context.Background(), msg, mock)
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected MsgExecuteContract")
		require.False(t, resp.Accept)
	})

	t.Run("returns error when query fails", func(t *testing.T) {
		auth := types.NewCodeExecutionAuthorization([]uint64{1})
		mock := &mockQueryProvider{
			err: fmt.Errorf("contract not found"),
		}

		msg := &wasmtypes.MsgExecuteContract{
			Contract: validContract,
		}

		resp, err := auth.AcceptWith(context.Background(), msg, mock)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to query contract info")
		require.False(t, resp.Accept)
	})

	t.Run("accepts first code ID in list", func(t *testing.T) {
		auth := types.NewCodeExecutionAuthorization([]uint64{5, 10, 15})
		mock := &mockQueryProvider{
			contractInfo: &wasmtypes.ContractInfo{CodeID: 5},
		}

		msg := &wasmtypes.MsgExecuteContract{
			Contract: validContract,
		}

		resp, err := auth.AcceptWith(context.Background(), msg, mock)
		require.NoError(t, err)
		require.True(t, resp.Accept)
	})

	t.Run("accepts last code ID in list", func(t *testing.T) {
		auth := types.NewCodeExecutionAuthorization([]uint64{5, 10, 15})
		mock := &mockQueryProvider{
			contractInfo: &wasmtypes.ContractInfo{CodeID: 15},
		}

		msg := &wasmtypes.MsgExecuteContract{
			Contract: validContract,
		}

		resp, err := auth.AcceptWith(context.Background(), msg, mock)
		require.NoError(t, err)
		require.True(t, resp.Accept)
	})

	t.Run("accepts single code ID authorization", func(t *testing.T) {
		auth := types.NewCodeExecutionAuthorization([]uint64{42})
		mock := &mockQueryProvider{
			contractInfo: &wasmtypes.ContractInfo{CodeID: 42},
		}

		msg := &wasmtypes.MsgExecuteContract{
			Contract: validContract,
		}

		resp, err := auth.AcceptWith(context.Background(), msg, mock)
		require.NoError(t, err)
		require.True(t, resp.Accept)
	})
}
