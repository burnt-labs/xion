package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	aatypes "github.com/larry0x/abstract-account/x/abstractaccount/types"
)

const (
	FlagSplit = "split"
	signMode  = signing.SignMode_SIGN_MODE_DIRECT
)

// NewTxCmd returns a root CLI command handler for all x/xion transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Xion transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewSendTxCmd(),
		NewMultiSendTxCmd(),
		NewSignCmd(),
	)

	return txCmd
}

// NewSendTxCmd returns a CLI command handler for creating a MsgSend transaction.
func NewSendTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send [from_key_or_address] [to_address] [amount]",
		Short: "Send funds from one account to another.",
		Long: `Send funds from one account to another.
Note, the '--from' flag is ignored as it is implied from [from_key_or_address].
When using '--dry-run' a key name cannot be used, only a bech32 address.
`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Flags().Set(flags.FlagFrom, args[0])
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			toAddr, err := sdk.AccAddressFromBech32(args[1])
			if err != nil {
				return err
			}

			coins, err := sdk.ParseCoinsNormalized(args[2])
			if err != nil {
				return err
			}

			msg := types.NewMsgSend(clientCtx.GetFromAddress(), toAddr, coins)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewMultiSendTxCmd returns a CLI command handler for creating a MsgMultiSend transaction.
// For a better UX this command is limited to send funds from one account to two or more accounts.
func NewMultiSendTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multi-send [from_key_or_address] [to_address_1, to_address_2, ...] [amount]",
		Short: "Send funds from one account to two or more accounts.",
		Long: `Send funds from one account to two or more accounts.
By default, sends the [amount] to each address of the list.
Using the '--split' flag, the [amount] is split equally between the addresses.
Note, the '--from' flag is ignored as it is implied from [from_key_or_address].
When using '--dry-run' a key name cannot be used, only a bech32 address.
`,
		Args: cobra.MinimumNArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Flags().Set(flags.FlagFrom, args[0])
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			coins, err := sdk.ParseCoinsNormalized(args[len(args)-1])
			if err != nil {
				return err
			}

			if coins.IsZero() {
				return fmt.Errorf("must send positive amount")
			}

			split, err := cmd.Flags().GetBool(FlagSplit)
			if err != nil {
				return err
			}

			totalAddrs := sdk.NewInt(int64(len(args) - 2))
			// coins to be received by the addresses
			sendCoins := coins
			if split {
				sendCoins = coins.QuoInt(totalAddrs)
			}

			var output []banktypes.Output
			for _, arg := range args[1 : len(args)-1] {
				toAddr, err := sdk.AccAddressFromBech32(arg)
				if err != nil {
					return err
				}

				output = append(output, banktypes.NewOutput(toAddr, sendCoins))
			}

			// amount to be send from the from address
			var amount sdk.Coins
			if split {
				// user input: 1000stake to send to 3 addresses
				// actual: 333stake to each address (=> 999stake actually sent)
				amount = sendCoins.MulInt(totalAddrs)
			} else {
				amount = coins.MulInt(totalAddrs)
			}

			msg := types.NewMsgMultiSend([]banktypes.Input{banktypes.NewInput(clientCtx.FromAddress, amount)}, output)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Bool(FlagSplit, false, "Send the equally split token amount to each address")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewSendTxCmd returns a CLI command handler for creating a MsgSend transaction.
func NewSignCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign [keyname] [path/to/tx.json]",
		Short: "sign a transaction",
		Long:  `Sign transaction by retrieving the Smart Contract Account signer.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.Flags().Set(flags.FlagFrom, args[0]); err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txBz, err := os.ReadFile(args[1])
			if err != nil {
				panic(err)
			}

			stdTx, err := clientCtx.TxConfig.TxJSONDecoder()(txBz)
			if err != nil {
				panic(err)
			}

			queryClient := authtypes.NewQueryClient(clientCtx) // TODO: determine if the ClientCtx has a qurey client

			signerAcc, err := getSignerOfTx(queryClient, stdTx)
			if err != nil {
				panic(err)
			}

			signerData := authsigning.SignerData{
				Address:       signerAcc.GetAddress().String(),
				ChainID:       clientCtx.ChainID,
				AccountNumber: signerAcc.GetAccountNumber(),
				Sequence:      signerAcc.GetSequence(),
				PubKey:        signerAcc.GetPubKey(), // NOTE: NilPubKey
			}

			txBuilder, err := clientCtx.TxConfig.WrapTxBuilder(stdTx)
			if err != nil {
				panic(err)
			}

			sigData := signing.SingleSignatureData{
				SignMode:  signMode,
				Signature: nil,
			}

			sig := signing.SignatureV2{
				PubKey:   signerAcc.GetPubKey(), // NOTE: NilPubKey
				Data:     &sigData,
				Sequence: signerAcc.GetSequence(),
			}

			if err := txBuilder.SetSignatures(sig); err != nil {
				panic(err)
			}

			signBytes, err := clientCtx.TxConfig.SignModeHandler().GetSignBytes(signMode, signerData, txBuilder.GetTx())
			if err != nil {
				panic(err)
			}
			sigBytes, _, err := clientCtx.Keyring.Sign(clientCtx.GetFromName(), signBytes)
			if err != nil {
				panic(err)
			}

			sigData = signing.SingleSignatureData{
				SignMode:  signMode,
				Signature: sigBytes,
			}

			sig = signing.SignatureV2{
				PubKey:   signerAcc.GetPubKey(),
				Data:     &sigData,
				Sequence: signerAcc.GetSequence(),
			}

			if err := txBuilder.SetSignatures(sig); err != nil {
				panic(err)
			}

			bz, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
			if err != nil {
				panic(err)
			}
			res, err := clientCtx.BroadcastTx(bz)
			if err != nil {
				panic(err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func getSignerOfTx(queryClient authtypes.QueryClient, stdTx sdk.Tx) (*aatypes.AbstractAccount, error) {
	var signerAddr sdk.AccAddress = nil
	for i, msg := range stdTx.GetMsgs() {
		signers := msg.GetSigners()
		if len(signers) != 1 {
			return nil, fmt.Errorf("msg %d has more than one signers", i)
		}

		if signerAddr != nil && !signerAddr.Equals(signers[0]) {
			return nil, errors.New("tx has more than one signers")
		}

		signerAddr = signers[0]
	}

	res, err := queryClient.Account(context.Background(), &authtypes.QueryAccountRequest{Address: signerAddr.String()})
	if err != nil {
		return nil, err
	}

	if res.Account.TypeUrl != typeURL((*aatypes.AbstractAccount)(nil)) {
		return nil, fmt.Errorf("signer %s is not an AbstractAccount", signerAddr.String())
	}

	var acc = &aatypes.AbstractAccount{}
	if err = proto.Unmarshal(res.Account.Value, acc); err != nil {
		return nil, err
	}
	//panic(fmt.Sprintf("Account: %s\nTypeURL: %s\nAcc: %+v", res.Account.TypeUrl, typeURL((*aatypes.AbstractAccount)(nil)), acc))

	return acc, nil
}

func typeURL(x proto.Message) string {
	return "/" + proto.MessageName(x)
}
