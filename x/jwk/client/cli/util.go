package cli

import (
	"encoding/json"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/spf13/cobra"
)

func CmdConvertPemToJson() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert-pem [file]",
		Short: "Convery PEM to JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			publicKeyBz, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			publicKey, err := jwk.ParseKey(publicKeyBz, jwk.WithPEM(true))
			if err != nil {
				return err
			}
			publicKeyJSON, err := json.Marshal(publicKey)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			return clientCtx.PrintBytes(publicKeyJSON)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
