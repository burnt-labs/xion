package main

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	tmcfg "github.com/cometbft/cometbft/config"

	"github.com/cosmos/cosmos-sdk/server"
)

func TestSetValidatorTimeout(t *testing.T) {
	serverCtx := server.NewDefaultContext()
	serverCtx.Config = tmcfg.DefaultConfig()

	setValidatorTimeout(serverCtx, 2500*time.Millisecond)

	require.Equal(t, 2500*time.Millisecond, serverCtx.Config.Consensus.TimeoutCommit)
}

func TestValidatorTimeoutCommitFlag(t *testing.T) {
	tests := map[string]struct {
		args     []string
		expected time.Duration
	}{
		"defaults to one second": {
			expected: time.Second,
		},
		"accepts a CLI override": {
			args:     []string{"--consensus.timeout_commit=2500ms"},
			expected: 2500 * time.Millisecond,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := &cobra.Command{}
			addModuleInitFlags(cmd)
			require.NoError(t, cmd.Flags().Parse(tc.args))

			timeoutCommit, err := cmd.Flags().GetDuration("consensus.timeout_commit")
			require.NoError(t, err)
			require.Equal(t, tc.expected, timeoutCommit)
		})
	}
}
