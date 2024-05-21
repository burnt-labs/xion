package cli

import (
	"encoding/base64"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/burnt-labs/xion/x/jwk/types"
)

func CmdListAudience() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-audience",
		Short: "list all audience",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllAudienceRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.AudienceAll(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddPaginationFlagsToCmd(cmd, cmd.Use)
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdShowAudience() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-audience [aud]",
		Short: "shows a audience",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			argAud := args[0]

			params := &types.QueryGetAudienceRequest{
				Aud: argAud,
			}

			res, err := queryClient.Audience(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdShowAudienceClaim() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-audience-claim [hash]",
		Short: "shows an audience claim",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			argAudHashStr := args[0]
			audHash, err := base64.StdEncoding.DecodeString(argAudHashStr)
			if err != nil {
				return err
			}

			params := &types.QueryGetAudienceClaimRequest{
				Hash: audHash,
			}

			res, err := queryClient.AudienceClaim(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
