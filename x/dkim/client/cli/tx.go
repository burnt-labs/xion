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
func MsgUpdateParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-params [some-value]",
		Short: "Update the params (must be submitted from the authority)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, _ []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			senderAddress := cliCtx.GetFromAddress()

			msg := &types.MsgUpdateParams{
				Authority: senderAddress.String(),
				Params:    types.Params{},
			}

			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// Returns a CLI command handler for registering a
// contract for the module.
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

			domain := args[0]
			pemKey := types.FormatToPemKey(args[1], true)
			block, _ := pem.Decode([]byte(pemKey))
			if block == nil {
				return types.ErrParsingPrivKey
			}

			msg := &types.MsgRevokeDkimPubKey{
				Signer: cliCtx.GetFromAddress().String(),
				Domain: domain,
				PrivKey: pem.EncodeToMemory(
					block,
				),
			}
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
