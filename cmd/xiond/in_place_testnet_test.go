package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The in-place-testnet command (server.AddTestnetCreatorCommand) must be
// registered in initRootCmd and honor addModuleInitFlags; rehearsal workflows
// (restored mainnet snapshot -> in-place-testnet) depend on both.
func TestInPlaceTestnetRegistered(t *testing.T) {
	root, _ := NewRootCmd()
	cmd, _, err := root.Find([]string{"in-place-testnet"})
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "in-place-testnet", cmd.Name())
	require.Contains(t, cmd.Use, "[newChainID] [newOperatorAddress]")
	require.NotNil(t, cmd.Flags().Lookup("trigger-testnet-upgrade"), "upgrade-handler trigger flag must exist")
	require.NotNil(t, cmd.Flags().Lookup("consensus.timeout_commit"), "addModuleInitFlags must apply")

	// The command must be registered through server.AddCommands with the
	// production app wiring (newApp/appExport), not a test fixture: rehearsal
	// chains must exercise the real upgrade handler and indexer startup.
	_, _, err = root.Find([]string{"start"})
	require.NoError(t, err)
	require.NotNil(t, cmd.RunE, "in-place-testnet must execute through RunE like start")
}
