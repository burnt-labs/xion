// client/cli/query.go
package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/burnt-labs/xion/x/zk/types"
)

// GetQueryCmd returns the cli query commands for the zk module
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdQueryVKey(),
		GetCmdQueryVKeyByName(),
		GetCmdQueryVKeys(),
		GetCmdQueryHasVKey(),
		GetCmdQueryVerifyProof(),
		GetCmdQueryVerifyUltraHonk(),
		GetCmdQueryParams(),
	)

	return cmd
}

// GetCmdQueryVKey queries a verification key by ID
func GetCmdQueryVKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vkey [id]",
		Short: "Query a verification key by ID",
		Args:  cobra.ExactArgs(1),
		Example: fmt.Sprintf(
			`$ %s query zk vkey 0
$ %s q zk vkey 5`,
			"xiond", "xiond",
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid vkey ID: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.VKey(context.Background(), &types.QueryVKeyRequest{
				Id: id,
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

// GetCmdQueryVKeyByName queries a verification key by name
func GetCmdQueryVKeyByName() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vkey-by-name [name]",
		Short: "Query a verification key by name",
		Args:  cobra.ExactArgs(1),
		Example: fmt.Sprintf(
			`$ %s query zk vkey-by-name email_auth
$ %s q zk vkey-by-name rollup_circuit`,
			"xiond", "xiond",
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.VKeyByName(context.Background(), &types.QueryVKeyByNameRequest{
				Name: args[0],
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

// GetCmdQueryVKeys queries all verification keys
func GetCmdQueryVKeys() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vkeys",
		Short: "Query all verification keys with pagination",
		Example: fmt.Sprintf(
			`$ %s query zk vkeys
$ %s q zk vkeys --limit 10 --offset 0
$ %s q zk vkeys --page 2 --limit 20`,
			"xiond", "xiond", "xiond",
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.VKeys(context.Background(), &types.QueryVKeysRequest{
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "vkeys")
	return cmd
}

// GetCmdQueryHasVKey checks if a verification key exists
func GetCmdQueryHasVKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "has-vkey [name]",
		Short: "Check if a verification key exists by name",
		Args:  cobra.ExactArgs(1),
		Example: fmt.Sprintf(
			`$ %s query zk has-vkey email_auth
$ %s q zk has-vkey rollup_circuit`,
			"xiond", "xiond",
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.HasVKey(context.Background(), &types.QueryHasVKeyRequest{
				Name: args[0],
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

// GetCmdQueryVerifyProof verifies a zero-knowledge proof
func GetCmdQueryVerifyProof() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify-proof [proof-file]",
		Short: "Verify a zero-knowledge proof using a stored verification key",
		Long: `Verify a zero-knowledge proof using a stored verification key.
The proof file should contain the JSON-encoded proof.
You must specify either --vkey-name or --vkey-id.
Public inputs should be provided via --public-inputs as a comma-separated list.`,
		Args: cobra.ExactArgs(1),
		Example: fmt.Sprintf(
			`$ %s query zk verify-proof proof.json --vkey-name email_auth --public-inputs "1,2,3,4"
$ %s q zk verify-proof proof.json --vkey-id 0 --public-inputs "1,2,3,4"
$ %s q zk verify-proof ./proofs/email.json --vkey-name email_auth --public-inputs "$(cat inputs.txt)"`,
			"xiond", "xiond", "xiond",
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			// Read proof from file
			proofBytes, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("failed to read proof file: %w", err)
			}

			// Get flags
			vkeyName, _ := cmd.Flags().GetString("vkey-name")
			vkeyID, _ := cmd.Flags().GetUint64("vkey-id")
			publicInputsStr, _ := cmd.Flags().GetString("public-inputs")

			// Validate inputs: at least one of vkey-name or vkey-id (server prefers name when both set)
			if vkeyName == "" && vkeyID == 0 {
				return fmt.Errorf("either --vkey-name or --vkey-id must be specified")
			}

			if publicInputsStr == "" {
				return fmt.Errorf("--public-inputs must be specified")
			}

			// Parse public inputs (comma-separated)
			publicInputs := ParsePublicInputs(publicInputsStr)

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.ProofVerify(context.Background(), &types.QueryVerifyRequest{
				Proof:        proofBytes,
				PublicInputs: publicInputs,
				VkeyName:     vkeyName,
				VkeyId:       vkeyID,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().String("vkey-name", "", "Name of the verification key to use")
	cmd.Flags().Uint64("vkey-id", 0, "ID of the verification key to use")
	cmd.Flags().String("public-inputs", "", "Comma-separated list of public inputs")

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryVerifyUltraHonk verifies an UltraHonk (Barretenberg) proof.
func GetCmdQueryVerifyUltraHonk() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify-ultrahonk [proof-file]",
		Short: "Verify an UltraHonk (Barretenberg) proof using a stored verification key",
		Long: `Verify an UltraHonk (Barretenberg) proof using a stored verification key.
The proof file should contain raw binary proof bytes (e.g. from bb prove -o proof.bin).
Public inputs must be provided via --public-inputs-file as raw binary (32-byte field elements concatenated).
You must specify either --vkey-name or --vkey-id (the vkey must be an UltraHonk key).`,
		Args: cobra.ExactArgs(1),
		Example: fmt.Sprintf(
			`$ %s query zk verify-ultrahonk proof.bin --vkey-name my_circuit --public-inputs-file public_inputs.bin
$ %s q zk verify-ultrahonk proof.bin --vkey-id 1 --public-inputs-file inputs.bin`,
			"xiond", "xiond",
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			proofBytes, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("failed to read proof file: %w", err)
			}

			vkeyName, _ := cmd.Flags().GetString("vkey-name")
			vkeyID, _ := cmd.Flags().GetUint64("vkey-id")
			inputsPath, _ := cmd.Flags().GetString("public-inputs-file")

			// At least one of vkey-name or vkey-id (server prefers name when both set, same as verify-proof)
			if vkeyName == "" && vkeyID == 0 {
				return fmt.Errorf("either --vkey-name or --vkey-id must be specified")
			}
			if inputsPath == "" {
				return fmt.Errorf("--public-inputs-file must be specified")
			}

			publicInputs, err := os.ReadFile(inputsPath)
			if err != nil {
				return fmt.Errorf("failed to read public inputs file: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.ProofVerifyUltraHonk(context.Background(), &types.QueryVerifyUltraHonkRequest{
				Proof:        proofBytes,
				PublicInputs: publicInputs,
				VkeyName:     vkeyName,
				VkeyId:       vkeyID,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().String("vkey-name", "", "Name of the UltraHonk verification key to use")
	cmd.Flags().Uint64("vkey-id", 0, "ID of the UltraHonk verification key to use")
	cmd.Flags().String("public-inputs-file", "", "Path to binary file containing public inputs (32-byte field elements concatenated)")

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query zk module parameters",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			if clientCtx.Output == nil {
				return fmt.Errorf("client context not configured: missing output writer (run within a proper app client)")
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Params(context.Background(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// ParsePublicInputs parses a comma-separated string into a slice of strings
func ParsePublicInputs(input string) []string {
	if input == "" {
		return []string{}
	}

	var inputs []string
	var current string

	for _, char := range input {
		if char == ',' {
			if current != "" {
				inputs = append(inputs, current)
				current = ""
			}
		} else if char != ' ' && char != '\t' && char != '\n' {
			current += string(char)
		}
	}

	if current != "" {
		inputs = append(inputs, current)
	}

	return inputs
}
