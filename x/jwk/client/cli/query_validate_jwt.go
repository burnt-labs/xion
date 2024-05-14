package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/burnt-labs/xion/x/jwk/types"
)

var _ = strconv.Itoa(0)

func CmdValidateJWT() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate-jwt [aud] [sub] [sig-bytes]",
		Short: "Query ValidateJWT",
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

			params := &types.QueryValidateJWTRequest{
				Aud:      reqAud,
				Sub:      reqSub,
				SigBytes: reqSigBytes,
			}

			fmt.Printf("request: %s", params)

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
