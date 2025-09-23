package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/burnt-labs/xion/x/dkim/types"
)

// !NOTE: Must enable in module.go (disabled in favor of autocli.go)

func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for " + types.ModuleName,
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	queryCmd.AddCommand(GetDkimPublicKey(), GetDkimPublicKeys(), GenerateDkimPublicKey())
	return queryCmd
}

func GetCmdParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Show module params",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetDkimPublicKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dkim-pubkey <domain> <selector> [flag]",
		Short:   "Get a DKIM public key",
		Aliases: []string{"qdkim"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.DkimPubKey(cmd.Context(), &types.QueryDkimPubKeyRequest{
				Domain:   args[0],
				Selector: args[1],
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

func GetDkimPublicKeys() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dkim-pubkeys [flag] [domain] [selector | poseidon_hash]",
		Short: "Get a DKIM public key matching filter parameters",
		Long: `Get a DKIM public key matching filter parameters. 
				If domain and selector are provided, it will return the DKIM public key for that domain and selector. 
				If domain and poseidon hash are provided, it will return the DKIM public key for that domain and poseidon hash. 
				If no filter parameters are provided, it will return all DKIM public keys.`,
		Example: "dkim-pubkeys --domain x.com --selector dkim-202308 \n dkim-pubkeys --domain x.com --poseidon-hash 1234567890",
		Aliases: []string{"qdkims"},
		Args:    cobra.RangeArgs(0, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			domain, err := cmd.Flags().GetString("domain")
			if err != nil {
				return err
			}
			selector, err := cmd.Flags().GetString("selector")
			if err != nil {
				return err
			}
			poseidonHash, err := cmd.Flags().GetString("hash")
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.DkimPubKeys(cmd.Context(), &types.QueryDkimPubKeysRequest{
				Domain:       domain,
				Selector:     selector,
				PoseidonHash: []byte(poseidonHash),
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	cmd.Flags().String("domain", "", "Filter by domain")
	cmd.Flags().String("selector", "", "Filter by selector. If selector is provided, domain is required")
	cmd.Flags().String("hash", "", "Filter by poseidon hash. If poseidon hash is provided, domain is required")

	return cmd
}

func GenerateDkimPublicKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate-dkim-pubkey [flag] <domain> <selector>",
		Example: "gen-dkim-pubkey x.com dkim-202308",
		Short:   "Generate a DKIM msg to create a new DKIM public key",
		Long:    "This command generates a DKIM msg to create a new DKIM public key. The command will query dns for the public key and then compute the poseidon hash of the public key. The returned DKIM msg can be used to create a new DKIM public key using the AddDkimPubkey command.",
		Aliases: []string{"gdkim"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			pubKey, err := GetDKIMPublicKey(args[1], args[0])
			if err != nil {
				return err
			}
			hash, err := types.ComputePoseidonHash(pubKey)
			if err != nil {
				return err
			}
			dkimPubKey := types.DkimPubKey{
				Domain:       args[0],
				PubKey:       pubKey,
				Selector:     args[1],
				PoseidonHash: []byte(hash.String()),
				Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
				KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
			}

			return clientCtx.PrintProto(&dkimPubKey)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
