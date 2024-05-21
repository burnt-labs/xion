package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/burnt-labs/xion/x/xion/types"
)

func CmdWebAuthNVerifyRegister() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webauthn-register [addr] [challenge] [rp] [data]",
		Short: "Test Webauthn Registration",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			reqAddr := args[0]
			reqChallenge := args[1]
			reqRP := args[2]
			reqData := []byte(args[3])

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryWebAuthNVerifyRegisterRequest{
				Addr:      reqAddr,
				Challenge: reqChallenge,
				Rp:        reqRP,
				Data:      reqData,
			}

			res, err := queryClient.WebAuthNVerifyRegister(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdWebAuthNVerifyAuthenticate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webauthn-authenticate [addr] [challenge] [rp] [credential] [data]",
		Short: "Test Webauthn Authentication",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			reqAddr := args[0]
			reqChallenge := args[1]
			reqRP := args[2]
			reqCredential := []byte(args[3])
			reqData := []byte(args[4])

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryWebAuthNVerifyAuthenticateRequest{
				Addr:       reqAddr,
				Challenge:  reqChallenge,
				Rp:         reqRP,
				Credential: reqCredential,
				Data:       reqData,
			}

			res, err := queryClient.WebAuthNVerifyAuthenticate(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
