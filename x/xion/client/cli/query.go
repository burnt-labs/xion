package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/burnt-labs/xion/x/xion/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd() *cobra.Command {
	// Group jwk queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdWebAuthNVerifyRegister())
	cmd.AddCommand(CmdWebAuthNVerifyAuthenticate())
	cmd.AddCommand(CmdPlatformPercentage())
	cmd.AddCommand(CmdPlatformMinimum())

	// this line is used by starport scaffolding # 1

	return cmd
}
