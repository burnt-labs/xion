package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/burnt-labs/xion/x/tee/types"
)

// GetQueryCmd returns the cli query commands for the tee module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdVerifyQuote(),
	)

	return cmd
}

// GetCmdVerifyQuote returns the command to verify a TDX quote.
func GetCmdVerifyQuote() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify-quote [quote-file]",
		Short: "Verify a TDX quote from a binary file",
		Long:  `Verify a TDX attestation quote. The quote file should contain the raw TDX quote in Intel ABI format.`,
		Args:  cobra.ExactArgs(1),
		Example: fmt.Sprintf(
			`$ %s query tee verify-quote ./quote.bin
$ %s q tee verify-quote /path/to/tdx_quote.dat`,
			"xiond", "xiond",
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			quoteBytes, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("failed to read quote file: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.QuoteVerify(context.Background(), &types.QueryQuoteVerifyRequest{
				Quote: quoteBytes,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
