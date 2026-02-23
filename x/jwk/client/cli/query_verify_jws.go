package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/burnt-labs/xion/x/jwk/types"
)

var _ = strconv.Itoa(0)

func CmdVerifyJWS() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify-jws [aud] [sig-bytes]",
		Short: "Query VerifyJWS",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			reqAud := args[0]
			reqSigBytes := args[1]

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryVerifyJWSRequest{
				Aud:      reqAud,
				SigBytes: reqSigBytes,
			}

			res, err := queryClient.VerifyJWS(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
