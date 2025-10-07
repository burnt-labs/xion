package cli

import (
	"encoding/pem"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/burnt-labs/xion/x/dkim/types"
)

// !NOTE: Must enable in module.go (disabled in favor of autocli.go)

// NewTxCmd returns a root CLI command handler for certain modules
// transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      types.ModuleName + " subcommands.",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	txCmd.AddCommand(MsgRevokeDkimPubKey())
	return txCmd
}

// Returns a CLI command handler for registering a
// contract for the module.
// ParseAndValidateRevokeDkimMsg parses the private key and creates a revoke message.
// This function is extracted for testability.
func ParseAndValidateRevokeDkimMsg(signer, domain, privKeyStr string) (*types.MsgRevokeDkimPubKey, error) {
	pemKey := types.FormatToPemKey(privKeyStr, true)
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return nil, types.ErrParsingPrivKey
	}

	msg := &types.MsgRevokeDkimPubKey{
		Signer:  signer,
		Domain:  domain,
		PrivKey: pem.EncodeToMemory(block),
	}
	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func MsgRevokeDkimPubKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "revoke-dkim <domain> <priv_key>",
		Short:   "Revoke a Dkim pubkey without governance.",
		Long:    "Revoke a Dkim pubkey without governance. The private key is a PEM encoded private key without the headers and must be a contiguous string with no new line character.",
		Aliases: []string{"rdkim"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg, err := ParseAndValidateRevokeDkimMsg(
				cliCtx.GetFromAddress().String(),
				args[0],
				args[1],
			)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
