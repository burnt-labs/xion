package cli

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	aatypes "github.com/larry0x/abstract-account/x/abstractaccount/types"
	"github.com/spf13/cobra"

	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

const (
	FlagSplit           = "split"
	signMode            = signing.SignMode_SIGN_MODE_DIRECT
	flagSalt            = "salt"
	flagFunds           = "funds"
	flagAuthenticator   = "authenticator"
	flagAuthenticatorID = "authenticator-id"
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
		NewAddAuthenticatorCmd(),
		NewRegisterCmd(),
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
			if err := cmd.Flags().Set(flags.FlagFrom, args[0]); err != nil {
				return err
			}
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
			if err := cmd.Flags().Set(flags.FlagFrom, args[0]); err != nil {
				return err
			}
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

func NewRegisterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register [code-id] [keyname] --salt [string] --funds [coins,optional] --authenticator [Seckp256|Jwt,required] --authenticator-id [uint8]",
		Short: "Register an abstract account",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.Flags().Set(flags.FlagFrom, args[1]); err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			codeID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			authenticatorID, err := cmd.Flags().GetUint8(flagAuthenticatorID)
			if err != nil {
				return err
			}

			salt, err := cmd.Flags().GetString(flagSalt)
			if err != nil {
				return fmt.Errorf("salt: %s", err)
			}

			amountStr, err := cmd.Flags().GetString(flagFunds)
			if err != nil {
				return fmt.Errorf("amount: %s", err)
			}

			authenticatorType, err := cmd.Flags().GetString(flagAuthenticator)
			if err != nil {
				return fmt.Errorf("authenticator: %s", err)
			}

			amount, err := sdk.ParseCoinsNormalized(amountStr)
			if err != nil {
				return fmt.Errorf("amount: %s", err)
			}
			queryClient := wasmtypes.NewQueryClient(clientCtx)

			codeResp, err := queryClient.Code(
				context.Background(),
				&wasmtypes.QueryCodeRequest{
					CodeId: codeID,
				},
			)
			if err != nil {
				return err
			}
			creatorAddr := clientCtx.GetFromAddress()
			codeHash, err := hex.DecodeString(codeResp.DataHash.String())
			if err != nil {
				return err
			}
			predictedAddr := wasmkeeper.BuildContractAddressPredictable(codeHash, creatorAddr, []byte(salt), []byte{})

			signature, pubKey, err := clientCtx.Keyring.SignByAddress(clientCtx.GetFromAddress(), []byte(predictedAddr.String()))
			if err != nil {
				return fmt.Errorf("error signing predicted address : %s", err)
			}

			instantiateMsg := map[string]interface{}{}
			authenticatorDetails := map[string]interface{}{}
			authenticator := map[string]interface{}{}

			authenticatorDetails["id"] = authenticatorID
			authenticatorDetails["pubkey"] = pubKey.Bytes()
			authenticatorDetails["signature"] = signature
			authenticator[authenticatorType] = authenticatorDetails

			instantiateMsg["authenticator"] = authenticator
			instantiateMsgStr, err := json.Marshal(instantiateMsg)
			if err != nil {
				return fmt.Errorf("error signing contract msg : %s", err)
			}

			msg := &aatypes.MsgRegisterAccount{
				Sender: clientCtx.GetFromAddress().String(),
				CodeID: codeID,
				Msg:    []byte(string(instantiateMsgStr)),
				Funds:  amount,
				Salt:   []byte(salt),
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
		SilenceUsage: true,
	}

	flags.AddTxFlagsToCmd(cmd)

	cmd.Flags().String(flagSalt, "", "Salt value used in determining account address")
	cmd.Flags().String(flagAuthenticator, "", "Authenticator type: Seckp256K1|JWT")
	cmd.Flags().String(flagFunds, "", "Coins to send to the account during instantiation")
	cmd.Flags().Uint8(flagAuthenticatorID, 0, "Authenticator index locator")

	return cmd
}

func NewAddAuthenticatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-authenticator [contract-addr] --authenticator-id [uint8]",
		Short: "Add the signing key as an authenticator to an abstract account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			authenticatorID, err := cmd.Flags().GetUint8(flagAuthenticatorID)
			if err != nil {
				return err
			}

			contractAddr := args[0]

			signature, pubKey, err := clientCtx.Keyring.SignByAddress(clientCtx.GetFromAddress(), []byte(contractAddr))
			if err != nil {
				return fmt.Errorf("error signing address : %s", err)
			}

			secp256k1 := map[string]interface{}{}
			secp256k1["id"] = authenticatorID
			secp256k1["pubkey"] = pubKey.Bytes()
			secp256k1["signature"] = signature

			addAuthenticator := map[string]interface{}{}
			addAuthenticator["Secp256K1"] = secp256k1

			addAuthMethod := map[string]interface{}{}
			addAuthMethod["add_authenticator"] = addAuthenticator

			msg := map[string]interface{}{}
			msg["add_auth_method"] = addAuthMethod

			jsonMsg, err := json.Marshal(msg)
			if err != nil {
				return err
			}

			rawMsg := wasmtypes.RawContractMessage{}
			err = json.Unmarshal(jsonMsg, &rawMsg)
			if err != nil {
				return err
			}

			wasmMsg := &wasmtypes.MsgExecuteContract{
				Sender:   contractAddr,
				Contract: contractAddr,
				Msg:      rawMsg,
				Funds:    nil,
			}
			if err := wasmMsg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), wasmMsg)
		},
		SilenceUsage: true,
	}

	flags.AddTxFlagsToCmd(cmd)

	cmd.Flags().Uint8(flagAuthenticatorID, 0, "Authenticator index locator")

	return cmd
}

// NewSignCmd returns a CLI command to sign a Tx with the Smart Contract Account signer
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
			authenticatorID, err := cmd.Flags().GetUint8(flagAuthenticatorID)
			if err != nil {
				return err
			}

			txBz, err := os.ReadFile(args[1])
			if err != nil {
				return err
			}

			stdTx, err := clientCtx.TxConfig.TxJSONDecoder()(txBz)
			if err != nil {
				return err
			}

			queryClient := authtypes.NewQueryClient(clientCtx)

			signerAcc, err := getSignerOfTx(queryClient, stdTx)
			if err != nil {
				return err
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
				PubKey:   signerAcc.GetPubKey(),
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
			signedBytes, _, err := clientCtx.Keyring.Sign(clientCtx.GetFromName(), signBytes)
			if err != nil {
				panic(err)
			}

			sigBytes := append([]byte{authenticatorID}, signedBytes...)
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
	cmd.Flags().Uint8(flagAuthenticatorID, 0, "Authenticator index locator")
	return cmd
}

func getSignerOfTx(queryClient authtypes.QueryClient, stdTx sdk.Tx) (*aatypes.AbstractAccount, error) {
	var signerAddr sdk.AccAddress
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

	acc := &aatypes.AbstractAccount{}
	if err = proto.Unmarshal(res.Account.Value, acc); err != nil {
		return nil, err
	}

	return acc, nil
}

func typeURL(x proto.Message) string {
	return "/" + proto.MessageName(x)
}
