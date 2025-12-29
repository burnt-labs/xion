package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/burnt-labs/xion/x/zk/types"
)

// GetTxCmd returns the transaction commands for the zk module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdAddVKey(),
		GetCmdUpdateVKey(),
		GetCmdRemoveVKey(),
	)

	return cmd
}

// GetCmdAddVKey implements the add verification key command
func GetCmdAddVKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-vkey [name] [vkey-json-file] [description]",
		Short: "Add a new verification key",
		Long: `Add a new verification key to the blockchain.
The vkey-json-file should contain the JSON-encoded verification key from SnarkJS.
Only the governance module can add verification keys.`,
		Args: cobra.ExactArgs(3),
		Example: fmt.Sprintf(
			`$ %s tx zk add-vkey email_auth ./vkey.json "Email authentication circuit" --from mykey
$ %s tx zk add-vkey rollup_batch ./rollup_vkey.json "Rollup batch verification" --from mykey --chain-id xion-1`,
			"xiond", "xiond",
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			name := args[0]
			description := args[2]

			// Read vkey JSON file
			vkeyBytes, err := os.ReadFile(args[1])
			if err != nil {
				return fmt.Errorf("failed to read vkey file: %w", err)
			}

			// Validate the vkey
			if err := types.ValidateVKeyBytes(vkeyBytes); err != nil {
				return fmt.Errorf("invalid verification key: %w", err)
			}

			msg := &types.MsgAddVKey{
				Authority:   clientCtx.GetFromAddress().String(),
				Name:        name,
				Description: description,
				VkeyBytes:   vkeyBytes,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// GetCmdUpdateVKey implements the update verification key command
func GetCmdUpdateVKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-vkey [name] [vkey-json-file] [description]",
		Short: "Update an existing verification key",
		Long: `Update an existing verification key on the blockchain.
The vkey-json-file should contain the JSON-encoded verification key from SnarkJS.
Only the governance module can update verification keys.`,
		Args: cobra.ExactArgs(3),
		Example: fmt.Sprintf(
			`$ %s tx zk update-vkey email_auth ./new_vkey.json "Updated email authentication circuit" --from mykey
$ %s tx zk update-vkey rollup_batch ./new_rollup_vkey.json "Updated rollup verification" --from mykey`,
			"xiond", "xiond",
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			name := args[0]
			description := args[2]

			// Read vkey JSON file
			vkeyBytes, err := os.ReadFile(args[1])
			if err != nil {
				return fmt.Errorf("failed to read vkey file: %w", err)
			}

			// Validate the vkey
			if err := types.ValidateVKeyBytes(vkeyBytes); err != nil {
				return fmt.Errorf("invalid verification key: %w", err)
			}

			msg := &types.MsgUpdateVKey{
				Authority:   clientCtx.GetFromAddress().String(),
				Name:        name,
				Description: description,
				VkeyBytes:   vkeyBytes,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// GetCmdRemoveVKey implements the remove verification key command
func GetCmdRemoveVKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-vkey [name]",
		Short: "Remove a verification key",
		Long: `Remove a verification key from the blockchain.
Only the governance module can remove verification keys.`,
		Args: cobra.ExactArgs(1),
		Example: fmt.Sprintf(
			`$ %s tx zk remove-vkey email_auth --from mykey
$ %s tx zk remove-vkey old_circuit --from mykey --chain-id xion-1`,
			"xiond", "xiond",
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			name := args[0]

			msg := &types.MsgRemoveVKey{
				Authority: clientCtx.GetFromAddress().String(),
				Name:      name,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
