package cli

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	aatypes "github.com/larry0x/abstract-account/x/abstractaccount/types"
	"github.com/spf13/cobra"

	"github.com/cosmos/gogoproto/proto"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	"cosmossdk.io/math"
	signing2 "cosmossdk.io/x/tx/signing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	cdcTypes "github.com/cosmos/cosmos-sdk/codec/types"
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
	flagAudience        = "aud"
	flagToken           = "token"
	flagSubject         = "sub"
)

type ExplicitAny struct {
	TypeURL string `json:"type_url"`
	Value   []byte `json:"value"`
}

type GrantConfig struct {
	Description   string      `json:"description"`
	Authorization interface{} `json:"authorization"`
	Optional      bool        `json:"optional"`
}

type UpdateGrantConfig struct {
	MsgTypeURL  string      `json:"msg_type_url"`
	GrantConfig GrantConfig `json:"grant_config"`
}

type FeeConfig struct {
	Description string      `json:"description"`
	Allowance   interface{} `json:"allowance,omitempty"`
	Expiration  int32       `json:"expiration,omitempty"`
}

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
		NewUpdateConfigsCmd(),
		NewUpdateParamsCmd(),
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

			totalAddrs := math.NewInt(int64(len(args) - 2))
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
		Use:   "register [code-id] [keyname] --salt [string] --funds [coins,optional] --authenticator [Seckp256|Jwt,required] --authenticator-id [uint8] --aud [string] --sub [string] --token [string]",
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

			signature, pubKey, err := clientCtx.Keyring.SignByAddress(
				clientCtx.GetFromAddress(),
				[]byte(predictedAddr.String()),
				signMode,
			)
			if err != nil {
				return fmt.Errorf("error signing predicted address : %s", err)
			}
			// TODO: Split authenticator types using switch,
			var instantiateMsg string
			switch authenticatorType {
			case "Jwt":
				sub, err := cmd.Flags().GetString(flagSubject)
				if err != nil {
					return fmt.Errorf("subject: %s", err)
				}

				aud, err := cmd.Flags().GetString(flagAudience)
				if err != nil {
					return fmt.Errorf("audience: %s", err)
				}

				token, err := cmd.Flags().GetString(flagToken)
				if err != nil {
					return fmt.Errorf("token: %s", err)
				}

				instantiateMsg, err = newInstantiateJwtMsg(token, authenticatorType, sub, aud, authenticatorID)
				if err != nil {
					return err
				}
			default:
				instantiateMsg, err = newInstantiateMsg(authenticatorType, authenticatorID, signature, pubKey.Bytes())
				if err != nil {
					return err
				}
			}

			msg := registerMsg(clientCtx.GetFromAddress().String(), salt, instantiateMsg, codeID, amount)
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
	cmd.Flags().String(flagAudience, "", "Recipient for the token")
	cmd.Flags().String(flagToken, "", "Pre signed JWT")
	cmd.Flags().String(flagSubject, "", "Principal for the token")

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

			signMode := signing.SignMode_SIGN_MODE_UNSPECIFIED
			switch clientCtx.SignModeStr {
			case flags.SignModeDirect:
				signMode = signing.SignMode_SIGN_MODE_DIRECT
			case flags.SignModeLegacyAminoJSON:
				signMode = signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON
			case flags.SignModeDirectAux:
				signMode = signing.SignMode_SIGN_MODE_DIRECT_AUX
			case flags.SignModeTextual:
				signMode = signing.SignMode_SIGN_MODE_TEXTUAL
			case flags.SignModeEIP191:
				signMode = signing.SignMode_SIGN_MODE_EIP_191
			}

			signature, pubKey, err := clientCtx.Keyring.SignByAddress(clientCtx.GetFromAddress(), []byte(contractAddr), signMode)
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
		Use:   "sign [keyname] [signer_account] [path/to/tx.json]",
		Short: "sign a transaction",
		Long:  `Sign transaction by retrieving the Smart Contract Account signer.`,
		Args:  cobra.ExactArgs(3),
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

			signerAddr, err := sdk.AccAddressFromBech32(args[1])
			if err != nil {
				return err
			}

			txBz, err := os.ReadFile(args[2])
			if err != nil {
				return err
			}

			stdTx, err := clientCtx.TxConfig.TxJSONDecoder()(txBz)
			if err != nil {
				return err
			}

			queryClient := authtypes.NewQueryClient(clientCtx)

			signerAcc, err := getSignerOfTx(queryClient, signerAddr)
			if err != nil {
				return err
			}

			signerData := signing2.SignerData{
				Address:       signerAcc.GetAddress().String(),
				ChainID:       clientCtx.ChainID,
				AccountNumber: signerAcc.GetAccountNumber(),
				Sequence:      signerAcc.GetSequence(),
				PubKey:        nil, // NOTE: NilPubKey
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

			adaptableTx, ok := txBuilder.GetTx().(authsigning.V2AdaptableTx)
			if !ok {
				return fmt.Errorf("expected tx to implement V2AdaptableTx, got %T", txBuilder.GetTx())
			}

			txData := adaptableTx.GetSigningTxData()
			signBytes, err := clientCtx.TxConfig.SignModeHandler().GetSignBytes(
				clientCtx.CmdContext,
				signingv1beta1.SignMode_SIGN_MODE_DIRECT,
				signerData,
				txData,
			)
			if err != nil {
				panic(err)
			}
			signedBytes, _, err := clientCtx.Keyring.Sign(
				clientCtx.GetFromName(), signBytes,
				signMode,
			)
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

func NewUpdateConfigsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-configs [contract] [config_path_or_url]",
		Short: "Batch update grant configs and fee config for the treasury",
		Long:  "Batch update grant configs and fee config for the treasury. To read from a local file, use the --local flag otherwise the config_path_or_url is treated as a URL",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			cdc := clientCtx.Codec

			contract := args[0]
			configSource := args[1]

			// Determine source type (local file or URL)
			localSource, err := cmd.Flags().GetBool("local")
			if err != nil {
				return fmt.Errorf("failed to parse local flag: %w", err)
			}

			var configData struct {
				GrantConfig []UpdateGrantConfig `json:"grant_config"`
				FeeConfig   FeeConfig           `json:"fee_config"`
			}

			if localSource {
				// Read from local file
				fileData, err := os.ReadFile(configSource)
				if err != nil {
					return fmt.Errorf("failed to read configuration file: %w", err)
				}
				err = json.Unmarshal(fileData, &configData)
				if err != nil {
					return fmt.Errorf("failed to unmarshal local configuration file: %w", err)
				}
			} else {
				// Fetch JSON from URI
				parsedURL, err := url.Parse(configSource)
				if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
					return fmt.Errorf("invalid URL: %s", configSource)
				}
				// #nosec G107 - URL is controlled and safe in this context
				resp, err := http.Get(configSource)
				if err != nil {
					return fmt.Errorf("failed to fetch configuration from URI: %w", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("received non-200 response code: %d", resp.StatusCode)
				}

				err = json.NewDecoder(resp.Body).Decode(&configData)
				if err != nil {
					return fmt.Errorf("failed to decode JSON response: %w", err)
				}
			}

			var msgs []sdk.Msg
			// Process Grant Configs
			for _, grant := range configData.GrantConfig {
				auth := grant.GrantConfig.Authorization
				authM, ok := auth.(map[string]interface{})
				if !ok {
					return fmt.Errorf("failed to parse authorization from grant config")
				}
				grantConfig, err := ConvertJSONToAny(cdc, authM)
				if err != nil {
					return fmt.Errorf("failed to convert grant config to Any: %w", err)
				}
				grant.GrantConfig.Authorization = grantConfig
				executeMsg := map[string]interface{}{
					"update_grant_config": map[string]interface{}{
						"msg_type_url": grant.MsgTypeURL,
						"grant_config": grant.GrantConfig,
					},
				}
				msgBz, err := json.Marshal(executeMsg)
				if err != nil {
					return fmt.Errorf("failed to marshal execute message for grant: %w", err)
				}
				msg := &wasmtypes.MsgExecuteContract{
					Sender:   clientCtx.GetFromAddress().String(),
					Contract: contract,
					Msg:      msgBz,
					Funds:    sdk.Coins{},
				}
				msgs = append(msgs, msg)
			}

			// Process Fee Config
			allowance := configData.FeeConfig.Allowance
			allowanceM := allowance.(map[string]interface{})
			feeConfig, err := ConvertJSONToAny(cdc, allowanceM)
			if err != nil {
				return fmt.Errorf("failed to convert fee config to Any: %w", err)
			}
			configData.FeeConfig.Allowance = feeConfig

			feeExecuteMsg := map[string]interface{}{
				"update_fee_config": map[string]interface{}{
					"fee_config": configData.FeeConfig,
				},
			}
			feeMsgBz, err := json.Marshal(feeExecuteMsg)
			if err != nil {
				return fmt.Errorf("failed to marshal execute message for fee config: %w", err)
			}
			feeMsg := &wasmtypes.MsgExecuteContract{
				Sender:   clientCtx.GetFromAddress().String(),
				Contract: contract,
				Msg:      feeMsgBz,
				Funds:    sdk.Coins{},
			}
			msgs = append(msgs, feeMsg)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msgs...)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().Bool("local", false, "Specify if the config source is a local file instead of a URL")
	return cmd
}

func NewUpdateParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-params <contract> <display_url> <redirect_url> <icon_url>",
		Short: "Update treasury contract parameters",
		Long: `Updates a treasury contract's display URL, redirect URL, and icon URL.
		Example:
		update-params <contract_address> "https://example.com/display" "https://example.com/redirect" "https://example.com/icon.png"
		`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			contract := args[0]
			displayURL := args[1]
			redirectURL := args[2]
			iconURL := args[3]

			_, err = url.ParseRequestURI(displayURL)
			if err != nil {
				return fmt.Errorf("invalid display URL: %w", err)
			}
			_, err = url.ParseRequestURI(redirectURL)
			if err != nil {
				return fmt.Errorf("invalid redirect URL: %w", err)
			}
			_, err = url.ParseRequestURI(iconURL)
			if err != nil {
				return fmt.Errorf("invalid icon URL: %w", err)
			}

			// Construct the execute message
			updateMsg := map[string]interface{}{
				"update_params": map[string]interface{}{
					"params": map[string]string{
						"display_url":  displayURL,
						"redirect_url": redirectURL,
						"icon_url":     iconURL,
					},
				},
			}

			// Serialize the message to JSON
			msgBz, err := json.Marshal(updateMsg)
			if err != nil {
				return fmt.Errorf("failed to marshal execute message: %w", err)
			}

			// Create a MsgExecuteContract message
			msg := &wasmtypes.MsgExecuteContract{
				Sender:   clientCtx.GetFromAddress().String(),
				Contract: contract,
				Msg:      msgBz,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func getSignerOfTx(queryClient authtypes.QueryClient, address sdk.AccAddress) (*aatypes.AbstractAccount, error) {
	res, err := queryClient.Account(context.Background(), &authtypes.QueryAccountRequest{Address: address.String()})
	if err != nil {
		return nil, err
	}

	if res.Account.TypeUrl != typeURL((*aatypes.AbstractAccount)(nil)) {
		return nil, fmt.Errorf("signer %s is not an AbstractAccount", address.String())
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

func registerMsg(sender, salt, instantiateMsg string, codeID uint64, amount sdk.Coins) *aatypes.MsgRegisterAccount {
	msg := &aatypes.MsgRegisterAccount{
		Sender: sender,
		CodeID: codeID,
		Msg:    []byte(instantiateMsg),
		Funds:  amount,
		Salt:   []byte(salt),
	}
	return msg
}

func newInstantiateMsg(authenticatorType string, authenticatorID uint8, signature, pubKey []byte) (string, error) {
	instantiateMsg := map[string]interface{}{}
	authenticatorDetails := map[string]interface{}{}
	authenticator := map[string]interface{}{}

	authenticatorDetails["id"] = authenticatorID
	authenticatorDetails["pubkey"] = pubKey
	authenticatorDetails["signature"] = signature
	authenticator[authenticatorType] = authenticatorDetails

	instantiateMsg["authenticator"] = authenticator
	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	if err != nil {
		return "", fmt.Errorf("error signing contract msg : %s", err)
	}
	return string(instantiateMsgStr), nil
}

func newInstantiateJwtMsg(token, authenticatorType, sub, aud string, authenticatorID uint8) (string, error) {
	instantiateMsg := map[string]interface{}{}
	authenticatorDetails := map[string]interface{}{}
	authenticator := map[string]interface{}{}

	authenticatorDetails["sub"] = sub
	authenticatorDetails["aud"] = aud
	authenticatorDetails["id"] = authenticatorID
	authenticator[authenticatorType] = authenticatorDetails

	instantiateMsg["authenticator"] = authenticator
	authenticatorDetails["token"] = []byte(token)
	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	if err != nil {
		return "", err
	}

	return string(instantiateMsgStr), nil

	/*
		auds := jwt.ClaimStrings{aud}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"iss":              aud,
			"sub":              sub,
			"aud":              auds,
			"exp":              inFive.Unix(),
			"nbf":              fiveAgo.Unix(),
			"iat":              fiveAgo.Unix(),
			"transaction_hash": signature,
		})
		t.Logf("jwt claims: %v", token)

		// sign the JWT with the predefined key
		output, err := token.SignedString(privateKey)
		require.NoError(t, err)
		t.Logf("signed token: %s", output)
	*/
}

func ConvertJSONToAny(cdc codec.Codec, jsonInput map[string]interface{}) (ExplicitAny, error) {
	typeURL, ok := jsonInput["@type"].(string)
	if !ok {
		return ExplicitAny{}, fmt.Errorf("failed to parse type URL from JSON")
	}
	delete(jsonInput, "@type")
	// Resolve the concrete type for the given typeURL
	protoMsg, err := cdc.InterfaceRegistry().Resolve(typeURL)
	if err != nil {
		return ExplicitAny{}, fmt.Errorf("failed to resolve type URL %s: %w", typeURL, err)
	}
	jsonInputBz, err := json.Marshal(jsonInput)
	if err != nil {
		return ExplicitAny{}, fmt.Errorf("failed to marshal JSON input: %w", err)
	}
	// Unmarshal the JSON into the Protobuf message
	err = cdc.UnmarshalJSON(jsonInputBz, protoMsg)
	if err != nil {
		return ExplicitAny{}, fmt.Errorf("failed to unmarshal JSON into proto.Message: %w", err)
	}

	// Marshal the Protobuf message into Any
	val, err := cdcTypes.NewAnyWithValue(protoMsg)
	if err != nil {
		return ExplicitAny{}, fmt.Errorf("failed to marshal proto.Message into Any: %w", err)
	}

	res := ExplicitAny{
		TypeURL: val.TypeUrl,
		Value:   val.Value,
	}

	protoMsg.Reset()

	return res, nil
}
