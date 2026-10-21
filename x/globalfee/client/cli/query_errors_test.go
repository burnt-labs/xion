package cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"

	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"

	"github.com/burnt-labs/xion/x/globalfee/client/cli"
)

func TestGetCmdShowGlobalFeeParamsErrors(t *testing.T) {
	t.Run("client context cannot be built", func(t *testing.T) {
		_, err := clitestutil.ExecTestCLICmd(newMockCtx(t), cli.GetCmdShowGlobalFeeParams(), []string{"--node", "tcp://[::1"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing ']' in host")
	})

	t.Run("query failure is returned", func(t *testing.T) {
		ctx := newMockCtx(t).WithClient(clitestutil.NewMockCometRPC(abci.ResponseQuery{Code: 1, Log: "boom"}))
		_, err := clitestutil.ExecTestCLICmd(ctx, cli.GetCmdShowGlobalFeeParams(), []string{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "boom")
	})
}
