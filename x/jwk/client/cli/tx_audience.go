package cli

import (
	"crypto/sha256"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/burnt-labs/xion/x/jwk/types"
)

const (
	FlagNewAdmin = "new-admin"
)

func CmdCreateAudienceClaim() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-audience-claim [aud]",
		Short: "Create a new audience claim",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			audStr := args[0]

			audHash := sha256.Sum256([]byte(audStr))

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgCreateAudienceClaim(clientCtx.GetFromAddress(), audHash[:])
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

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
		Use:   "update-audience [aud] [key] --new-admin [new-admin]",
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

			newAdmin, err := cmd.Flags().GetString(FlagNewAdmin)
			if err != nil {
				return err
			}
			if newAdmin == "" {
				newAdmin = clientCtx.GetFromAddress().String()
			}

			msg := types.NewMsgUpdateAudience(
				clientCtx.GetFromAddress().String(),
				newAdmin,
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
	cmd.Flags().String(FlagNewAdmin, "", "address to provide as the new admin")

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
