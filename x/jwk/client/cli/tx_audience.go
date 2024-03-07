package cli

import (
	"github.com/burnt-labs/xion/x/jwk/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
)

func CmdCreateAudience() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-audience [aud] [key] [admin | optional]",
		Short: "Create a new audience",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Get indexes
			indexAud := args[0]

			// Get value arguments
			argKey := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// if admin provided, use it. if not, use `from`
			var admin string
			if len(args) == 3 {
				admin = args[2]
			} else {
				admin = clientCtx.GetFromAddress().String()
			}

			msg := types.NewMsgCreateAudience(
				admin,
				indexAud,
				argKey,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdUpdateAudience() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-audience [aud] [key]",
		Short: "Update a audience",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Get indexes
			indexAud := args[0]

			// Get value arguments
			argKey := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgUpdateAudience(
				clientCtx.GetFromAddress().String(),
				clientCtx.GetFromAddress().String(),
				indexAud,
				argKey,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdDeleteAudience() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete-audience [aud]",
		Short: "Delete a audience",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			indexAud := args[0]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgDeleteAudience(
				clientCtx.GetFromAddress().String(),
				indexAud,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
