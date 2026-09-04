package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

const (
	flagSalt  = "salt"
	flagFunds = "funds"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "abstract-account",
		Short:                      "Abstract account transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
		SilenceUsage:               true,
	}

	cmd.AddCommand(
		registerCmd(),
		updateParamsCmd(),
	)

	return cmd
}

func registerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "register [code-id] [msg] --salt [string] --funds [coins,optional]",
		Short:        "Register an abstract account",
		Args:         cobra.ExactArgs(2),
		RunE:         runRegisterCmd,
		SilenceUsage: true,
	}

	flags.AddTxFlagsToCmd(cmd)

	cmd.Flags().String(flagSalt, "", "Salt value used in determining account address")
	cmd.Flags().String(flagFunds, "", "Coins to send to the account during instantiation")

	return cmd
}

func runRegisterCmd(cmd *cobra.Command, args []string) error {
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		return err
	}

	return RegisterAccount(clientCtx, cmd.Flags(), args)
}

func RegisterAccount(clientCtx client.Context, flagSet *pflag.FlagSet, args []string) error {
	codeID, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return err
	}

	salt, err := flagSet.GetString(flagSalt)
	if err != nil {
		return fmt.Errorf("salt: %s", err)
	}

	amountStr, err := flagSet.GetString(flagFunds)
	if err != nil {
		return fmt.Errorf("amount: %s", err)
	}

	amount, err := sdk.ParseCoinsNormalized(amountStr)
	if err != nil {
		return fmt.Errorf("amount: %s", err)
	}

	msg := &types.MsgRegisterAccount{
		Sender: clientCtx.GetFromAddress().String(),
		CodeID: codeID,
		Msg:    []byte(args[1]),
		Funds:  amount,
		Salt:   []byte(salt),
	}

	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	return tx.GenerateOrBroadcastTxCLI(clientCtx, flagSet, msg)
}

func updateParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "update-params [json-encoded-params]",
		Short:        "Update the module's parameters",
		Args:         cobra.ExactArgs(1),
		RunE:         runUpdateParamsCmd,
		SilenceUsage: true,
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func runUpdateParamsCmd(cmd *cobra.Command, args []string) error {
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		return err
	}

	return UpdateParams(clientCtx, cmd.Flags(), args)
}

func UpdateParams(clientCtx client.Context, flagSet *pflag.FlagSet, args []string) error {
	var params types.Params
	if err := json.Unmarshal([]byte(args[0]), &params); err != nil {
		return err
	}

	msg := &types.MsgUpdateParams{
		Sender: clientCtx.GetFromAddress().String(),
		Params: &params,
	}

	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	return tx.GenerateOrBroadcastTxCLI(clientCtx, flagSet, msg)
}
