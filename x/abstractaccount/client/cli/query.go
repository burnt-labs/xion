package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "abstract-account",
		Short:                      "Querying commands for the abstract-account module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
	}

	cmd.AddCommand(
		paramsCmd(),
		accountAddressCmd(),
	)

	return cmd
}

func accountAddressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account-address [sender] --salt [string]",
		Short: "Query the registered or predicted abstract account address",
		Args:  cobra.ExactArgs(1),
		RunE:  queryAccountAddress,
	}

	cmd.Flags().String(flagSalt, "", "Salt value used in determining account address")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func queryAccountAddress(cmd *cobra.Command, args []string) error {
	clientCtx, err := client.GetClientQueryContext(cmd)
	if err != nil {
		return err
	}
	salt, err := cmd.Flags().GetString(flagSalt)
	if err != nil {
		return fmt.Errorf("salt: %s", err)
	}

	res, err := types.NewQueryClient(clientCtx).AccountAddress(cmd.Context(), &types.QueryAccountAddressRequest{
		Sender: args[0],
		Salt:   []byte(salt),
	})
	if err != nil {
		return err
	}

	return clientCtx.PrintProto(res)
}

func paramsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the module's parameters",
		Args:  cobra.NoArgs,
		RunE:  queryParams,
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func queryParams(cmd *cobra.Command, _ []string) error {
	clientCtx, err := client.GetClientQueryContext(cmd)
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	queryClient := types.NewQueryClient(clientCtx)

	res, err := queryClient.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		return err
	}

	return clientCtx.PrintProto(res.Params)
}
