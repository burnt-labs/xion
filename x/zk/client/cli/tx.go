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

// parseProofSystem converts the user-supplied string ("groth16", "gnark", or "ultrahonk") to the typed enum.
func parseProofSystem(s string) (types.ProofSystem, error) {
	switch s {
	case "groth16":
		return types.ProofSystem_PROOF_SYSTEM_GROTH16, nil
	case "gnark":
		return types.ProofSystem_PROOF_SYSTEM_GROTH16_GNARK, nil
	case "ultrahonk":
		return types.ProofSystem_PROOF_SYSTEM_ULTRA_HONK_ZK, nil
	default:
		return 0, fmt.Errorf("proof_system must be %q, %q, or %q, got %q", "groth16", "gnark", "ultrahonk", s)
	}
}

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
		Use:   "add-vkey [name] [vkey-file] [description] [proof-system]",
		Short: "Add a new verification key",
		Long: `Add a new verification key to the blockchain.
The vkey-file should contain the verification key: JSON for groth16 (SnarkJS/Circom), binary for gnark (gnark Groth16), or binary for ultrahonk (Barretenberg).
proof-system must be "groth16", "gnark", or "ultrahonk". Any account can add verification keys.`,
		Args: cobra.ExactArgs(4),
		Example: fmt.Sprintf(
			`$ %s tx zk add-vkey email_auth ./vkey.json "Email authentication circuit" groth16 --from mykey
$ %s tx zk add-vkey zkml_model ./model_vkey.bin "ZKML model verification" gnark --from mykey
$ %s tx zk add-vkey rollup_batch ./rollup_vkey.bin "Rollup batch verification" ultrahonk --from mykey --chain-id xion-1`,
			"xiond", "xiond", "xiond",
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			name := args[0]
			description := args[2]
			ps, err := parseProofSystem(args[3])
			if err != nil {
				return err
			}

			// Read vkey file (Groth16 JSON or UltraHonk binary; validation in ValidateBasic)
			vkeyBytes, err := os.ReadFile(args[1])
			if err != nil {
				return fmt.Errorf("failed to read vkey file: %w", err)
			}

			msg := &types.MsgAddVKey{
				Authority:   clientCtx.GetFromAddress().String(),
				Name:        name,
				Description: description,
				VkeyBytes:   vkeyBytes,
				ProofSystem: ps,
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
		Use:   "update-vkey [name] [vkey-file] [description] [proof-system]",
		Short: "Update an existing verification key",
		Long: `Update an existing verification key on the blockchain.
The vkey-file should contain the verification key: JSON for groth16 (SnarkJS/Circom), binary for gnark (gnark Groth16), or binary for ultrahonk (Barretenberg).
proof-system must be "groth16", "gnark", or "ultrahonk". Any account can update verification keys.`,
		Args: cobra.ExactArgs(4),
		Example: fmt.Sprintf(
			`$ %s tx zk update-vkey email_auth ./new_vkey.json "Updated email authentication circuit" groth16 --from mykey
$ %s tx zk update-vkey zkml_model ./new_model_vkey.bin "Updated ZKML model verification" gnark --from mykey
$ %s tx zk update-vkey rollup_batch ./new_rollup_vkey.bin "Updated rollup verification" ultrahonk --from mykey`,
			"xiond", "xiond", "xiond",
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			name := args[0]
			description := args[2]
			ps, err := parseProofSystem(args[3])
			if err != nil {
				return err
			}

			// Read vkey file (Groth16 JSON or UltraHonk binary; validation in ValidateBasic)
			vkeyBytes, err := os.ReadFile(args[1])
			if err != nil {
				return fmt.Errorf("failed to read vkey file: %w", err)
			}

			msg := &types.MsgUpdateVKey{
				Authority:   clientCtx.GetFromAddress().String(),
				Name:        name,
				Description: description,
				VkeyBytes:   vkeyBytes,
				ProofSystem: ps,
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
Any account can remove verification keys.`,
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
