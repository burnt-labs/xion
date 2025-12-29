package cli

import (
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/burnt-labs/xion/x/dkim/types"
)

// NewTxCmd returns a root CLI command handler for certain modules
// transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      types.ModuleName + " subcommands.",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	txCmd.AddCommand(MsgRevokeDkimPubKey(), NewTxDecodeCmd())
	return txCmd
}

// Returns a CLI command handler for registering a
// contract for the module.
// ParseAndValidateRevokeDkimMsg parses the private key and creates a revoke message.
// This function is extracted for testability.
func ParseAndValidateRevokeDkimMsg(signer, domain, privKeyStr string) (*types.MsgRevokeDkimPubKey, error) {
	pemKey := types.FormatToPemKey(privKeyStr, true)
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return nil, types.ErrParsingPrivKey
	}

	msg := &types.MsgRevokeDkimPubKey{
		Signer:  signer,
		Domain:  domain,
		PrivKey: pem.EncodeToMemory(block),
	}
	err := msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func MsgRevokeDkimPubKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "revoke-dkim <domain> <priv_key>",
		Short:   "Revoke a Dkim pubkey without governance.",
		Long:    "Revoke a Dkim pubkey without governance. The private key is a PEM encoded private key without the headers and must be a contiguous string with no new line character.",
		Aliases: []string{"rdkim"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg, err := ParseAndValidateRevokeDkimMsg(
				cliCtx.GetFromAddress().String(),
				args[0],
				args[1],
			)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewTxDecodeCmd returns a CLI command to decode a transaction from base64.
func NewTxDecodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decode [base64-tx]",
		Short: "Decode a transaction or sign doc from base64 encoding",
		Long: `Decode a base64 encoded transaction or sign document and display its contents.
The input can be either a full transaction or sign bytes (SignDoc).

Example:
  $ xiond tx decode CqIBCp8BChwvY29zbW9zLmJhbmsudjFiZXRhMS5Nc2dTZW5kEn8K...`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			return DecodeTx(clientCtx, args[0])
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// DecodeTx decodes a base64 encoded transaction or sign doc and outputs it as JSON.
func DecodeTx(clientCtx client.Context, txString string) error {
	// Get the transaction decoder from the client context
	if clientCtx.TxConfig == nil {
		return fmt.Errorf("tx config is not initialized")
	}

	// Validate input is not empty
	if len(txString) == 0 {
		return fmt.Errorf("failed to decode base64 string: input is empty")
	}

	// Normalize URL-safe base64 to standard base64
	// Replace URL-safe characters with standard ones
	normalizedString := strings.ReplaceAll(txString, "-", "+")
	normalizedString = strings.ReplaceAll(normalizedString, "_", "/")

	// Decode the base64 string
	txBytes, err := base64.StdEncoding.DecodeString(normalizedString)
	if err != nil {
		// Try without padding (raw encoding)
		txBytes, err = base64.RawStdEncoding.DecodeString(normalizedString)
		if err != nil {
			return fmt.Errorf("failed to decode base64 string: %w", err)
		}
	}

	// Try to decode as a full transaction first
	txDecoder := clientCtx.TxConfig.TxDecoder()
	decodedTx, err := txDecoder(txBytes)
	if err == nil {
		// Successfully decoded as a transaction
		jsonEncoder := clientCtx.TxConfig.TxJSONEncoder()
		jsonBytes, err := jsonEncoder(decodedTx)
		if err != nil {
			return fmt.Errorf("failed to encode transaction to JSON: %w", err)
		}
		return printPrettyJSON(clientCtx, jsonBytes)
	}

	// Try to decode as SignDoc (sign bytes)
	var signDoc txtypes.SignDoc
	if err := signDoc.Unmarshal(txBytes); err == nil && len(signDoc.BodyBytes) > 0 {
		return decodeSignDoc(clientCtx, &signDoc)
	}

	return fmt.Errorf("failed to decode: input is neither a valid transaction nor sign document")
}

// decodeSignDoc decodes and prints a SignDoc structure
func decodeSignDoc(clientCtx client.Context, signDoc *txtypes.SignDoc) error {
	// Decode the body bytes
	var txBody txtypes.TxBody
	if err := txBody.Unmarshal(signDoc.BodyBytes); err != nil {
		return fmt.Errorf("failed to decode sign doc body: %w", err)
	}

	// Decode the auth info bytes
	var authInfo txtypes.AuthInfo
	if err := authInfo.Unmarshal(signDoc.AuthInfoBytes); err != nil {
		return fmt.Errorf("failed to decode sign doc auth info: %w", err)
	}

	// Build output structure
	output := map[string]interface{}{
		"body":           nil,
		"auth_info":      nil,
		"chain_id":       signDoc.ChainId,
		"account_number": fmt.Sprintf("%d", signDoc.AccountNumber),
	}

	// Marshal body to JSON using codec for proper type URLs
	bodyJSON, err := clientCtx.Codec.MarshalJSON(&txBody)
	if err != nil {
		return fmt.Errorf("failed to marshal body to JSON: %w", err)
	}
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(bodyJSON, &bodyMap); err != nil {
		return fmt.Errorf("failed to parse body JSON: %w", err)
	}
	output["body"] = bodyMap

	// Marshal auth info to JSON using codec
	authInfoJSON, err := clientCtx.Codec.MarshalJSON(&authInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal auth info to JSON: %w", err)
	}
	var authInfoMap map[string]interface{}
	if err := json.Unmarshal(authInfoJSON, &authInfoMap); err != nil {
		return fmt.Errorf("failed to parse auth info JSON: %w", err)
	}
	output["auth_info"] = authInfoMap

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to marshal output to JSON: %w", err)
	}

	return printPrettyJSON(clientCtx, jsonBytes)
}

// printPrettyJSON prints JSON with indentation
func printPrettyJSON(clientCtx client.Context, jsonBytes []byte) error {
	var prettyJSON map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &prettyJSON); err != nil {
		// If pretty printing fails, just output the raw JSON
		return clientCtx.PrintString(string(jsonBytes) + "\n")
	}

	prettyBytes, err := json.MarshalIndent(prettyJSON, "", "  ")
	if err != nil {
		return clientCtx.PrintString(string(jsonBytes) + "\n")
	}

	return clientCtx.PrintString(string(prettyBytes) + "\n")
}
