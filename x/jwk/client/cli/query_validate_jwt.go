package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/burnt-labs/xion/x/jwk/types"
)

var _ = strconv.Itoa(0)

// Deprecated: Use CmdDecodeJWT instead.
func CmdValidateJWT() *cobra.Command {
	cmd := &cobra.Command{
		Use:        "validate-jwt [aud] [sub] [sig-bytes]",
		Short:      "Query ValidateJWT (deprecated: use decode-jwt)",
		Deprecated: "use decode-jwt instead",
		Args:       cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqAud := args[0]
			reqSub := args[1]
			reqSigBytes := args[2]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryValidateJWTRequest{
				Aud:      reqAud,
				Sub:      reqSub,
				SigBytes: reqSigBytes,
			}

			res, err := queryClient.ValidateJWT(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdDecodeJWT() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decode-jwt [aud] [sub] [sig-bytes]",
		Short: "Validate a JWT and return all claims (standard and private)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqAud := args[0]
			reqSub := args[1]
			reqSigBytes := args[2]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryDecodeJWTRequest{
				Aud:      reqAud,
				Sub:      reqSub,
				SigBytes: reqSigBytes,
			}

			res, err := queryClient.DecodeJWT(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
