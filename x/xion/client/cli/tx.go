package cli

import (
	"fmt"
	"github.com/burnt-labs/xion/x/xion/types"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

const (
	FlagSplit = "split"
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

// GetSignCommand returns the transaction sign command.
func GetSignCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign [file]",
		Short: "Sign a transaction generated offline",
		Long: `Sign a transaction created with the --generate-only flag.
It will read a transaction from [file], sign it, and print its JSON encoding.

If the --signature-only flag is set, it will output the signature parts only.

The --offline flag makes sure that the client will not reach out to full node.
As a result, the account and sequence number queries will not be performed and
it is required to set such parameters manually. Note, invalid values will cause
the transaction to fail.

The --multisig=<multisig_key> flag generates a signature on behalf of a multisig account
key. It implies --signature-only. Full multisig signed transactions may eventually
be generated via the 'multisign' command.
`,
		PreRun: preSignCmd,
		RunE:   makeSignCmd(),
		Args:   cobra.ExactArgs(1),
	}

	cmd.Flags().String(flagMultisig, "", "Address or key name of the multisig account on behalf of which the transaction shall be signed")
	cmd.Flags().Bool(flagOverwrite, false, "Overwrite existing signatures with a new one. If disabled, new signature will be appended")
	cmd.Flags().Bool(flagSigOnly, false, "Print only the signatures")
	cmd.Flags().String(flags.FlagOutputDocument, "", "The document will be written to the given file instead of STDOUT")
	cmd.Flags().Bool(flagAmino, false, "Generate Amino encoded JSON suitable for submiting to the txs REST endpoint")
	flags.AddTxFlagsToCmd(cmd)

	cmd.MarkFlagRequired(flags.FlagFrom)

	return cmd
}

func preSignCmd(cmd *cobra.Command, _ []string) {
	// Conditionally mark the account and sequence numbers required as no RPC
	// query will be done.
	if offline, _ := cmd.Flags().GetBool(flags.FlagOffline); offline {
		cmd.MarkFlagRequired(flags.FlagAccountNumber)
		cmd.MarkFlagRequired(flags.FlagSequence)
	}
}

func makeSignCmd() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) (err error) {
		var clientCtx client.Context

		clientCtx, err = client.GetClientTxContext(cmd)
		if err != nil {
			return err
		}

		clientCtx, txF, newTx, err := readTxAndInitContexts(clientCtx, cmd, args[0])
		if err != nil {
			return err
		}

		return signTx(cmd, clientCtx, txF, newTx)
	}
}

func signTx(cmd *cobra.Command, clientCtx client.Context, txF tx.Factory, newTx sdk.Tx) error {
	f := cmd.Flags()
	txCfg := clientCtx.TxConfig
	txBuilder, err := txCfg.WrapTxBuilder(newTx)
	if err != nil {
		return err
	}

	printSignatureOnly, err := cmd.Flags().GetBool(flagSigOnly)
	if err != nil {
		return err
	}

	multisig, err := cmd.Flags().GetString(flagMultisig)
	if err != nil {
		return err
	}

	from, err := cmd.Flags().GetString(flags.FlagFrom)
	if err != nil {
		return err
	}

	_, fromName, _, err := client.GetFromFields(clientCtx, txF.Keybase(), from)
	if err != nil {
		return fmt.Errorf("error getting account from keybase: %w", err)
	}

	overwrite, err := f.GetBool(flagOverwrite)
	if err != nil {
		return err
	}

	if multisig != "" {
		// Bech32 decode error, maybe it's a name, we try to fetch from keyring
		multisigAddr, multisigName, _, err := client.GetFromFields(clientCtx, txF.Keybase(), multisig)
		if err != nil {
			return fmt.Errorf("error getting account from keybase: %w", err)
		}
		multisigkey, err := getMultisigRecord(clientCtx, multisigName)
		if err != nil {
			return err
		}
		multisigPubKey, err := multisigkey.GetPubKey()
		if err != nil {
			return err
		}
		multisigLegacyPub := multisigPubKey.(*kmultisig.LegacyAminoPubKey)

		fromRecord, err := clientCtx.Keyring.Key(fromName)
		if err != nil {
			return fmt.Errorf("error getting account from keybase: %w", err)
		}
		fromPubKey, err := fromRecord.GetPubKey()
		if err != nil {
			return err
		}

		var found bool
		for _, pubkey := range multisigLegacyPub.GetPubKeys() {
			if pubkey.Equals(fromPubKey) {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("signing key is not a part of multisig key")
		}
		err = authclient.SignTxWithSignerAddress(
			txF, clientCtx, multisigAddr, fromName, txBuilder, clientCtx.Offline, overwrite)
		if err != nil {
			return err
		}
		printSignatureOnly = true
	} else {
		err = authclient.SignTx(txF, clientCtx, clientCtx.GetFromName(), txBuilder, clientCtx.Offline, overwrite)
	}
	if err != nil {
		return err
	}

	aminoJSON, err := f.GetBool(flagAmino)
	if err != nil {
		return err
	}

	bMode, err := f.GetString(flags.FlagBroadcastMode)
	if err != nil {
		return err
	}

	// set output
	closeFunc, err := setOutputFile(cmd)
	if err != nil {
		return err
	}

	defer closeFunc()
	clientCtx.WithOutput(cmd.OutOrStdout())

	var json []byte
	if aminoJSON {
		stdTx, err := tx.ConvertTxToStdTx(clientCtx.LegacyAmino, txBuilder.GetTx())
		if err != nil {
			return err
		}
		req := BroadcastReq{
			Tx:   stdTx,
			Mode: bMode,
		}
		json, err = clientCtx.LegacyAmino.MarshalJSON(req)
		if err != nil {
			return err
		}
	} else {
		json, err = marshalSignatureJSON(txCfg, txBuilder, printSignatureOnly)
		if err != nil {
			return err
		}
	}

	cmd.Printf("%s\n", json)

	return err
}

func marshalSignatureJSON(txConfig client.TxConfig, txBldr client.TxBuilder, signatureOnly bool) ([]byte, error) {
	parsedTx := txBldr.GetTx()
	if signatureOnly {
		sigs, err := parsedTx.GetSignaturesV2()
		if err != nil {
			return nil, err
		}
		return txConfig.MarshalSignatureJSON(sigs)
	}

	return txConfig.TxJSONEncoder()(parsedTx)
}
