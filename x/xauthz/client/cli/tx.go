package cli

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/burnt-labs/xion/x/xauthz/types"
)

const (
	flagExpiration = "expiration"
)

// GetTxCmd returns the transaction commands for this module.
func GetTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "xauthz transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
		SilenceUsage:               true,
	}
	txCmd.AddCommand(
		GrantCodeExecutionCmd(),
	)
	return txCmd
}

// GrantCodeExecutionCmd returns a CLI command to grant CodeExecutionAuthorization.
func GrantCodeExecutionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grant [grantee] [allowed_code_ids]",
		Short: "Grant authorization to execute contracts with specific code IDs",
		Long: `Grant authorization to an address to execute contracts on your behalf,
limited to contracts instantiated from the specified code IDs.

Examples:
  # Grant with expiration (Unix timestamp)
  $ xiond tx xauthz grant <grantee_addr> 1,2,3 --expiration 1893456000 --from mykey

  # Grant without expiration (never expires)
  $ xiond tx xauthz grant <grantee_addr> 1,2,3 --from mykey
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			grantee, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			codeIDs, err := parseCodeIDs(args[1])
			if err != nil {
				return err
			}

			authorization := types.NewCodeExecutionAuthorization(codeIDs)
			if err := authorization.ValidateBasic(); err != nil {
				return err
			}

			expire, err := getExpireTime(cmd)
			if err != nil {
				return err
			}

			grantMsg, err := authz.NewMsgGrant(clientCtx.GetFromAddress(), grantee, authorization, expire)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), grantMsg)
		},
		SilenceUsage: true,
	}

	cmd.Flags().Int64(flagExpiration, 0, "Unix timestamp for grant expiration (optional, 0 for no expiration)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// parseCodeIDs parses a comma-separated list of code IDs.
func parseCodeIDs(input string) ([]uint64, error) {
	if input == "" {
		return nil, errors.New("code IDs cannot be empty")
	}

	parts := strings.Split(input, ",")
	codeIDs := make([]uint64, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		codeID, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return nil, err
		}
		codeIDs = append(codeIDs, codeID)
	}

	if len(codeIDs) == 0 {
		return nil, errors.New("at least one code ID is required")
	}

	return codeIDs, nil
}

// getExpireTime parses the expiration flag and returns a time.Time pointer.
func getExpireTime(cmd *cobra.Command) (*time.Time, error) {
	exp, err := cmd.Flags().GetInt64(flagExpiration)
	if err != nil {
		return nil, err
	}
	if exp == 0 {
		return nil, nil
	}
	e := time.Unix(exp, 0)
	return &e, nil
}
