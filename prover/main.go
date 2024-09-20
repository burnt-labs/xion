package prover

import (
	"bytes"
	"fmt"
	"github.com/burnt-labs/xion/x/xion/client/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"io"
	"net/mail"
	"os"
	"strings"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/emersion/go-msgauth/dkim"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
)

const (
	FlagIcicle          = "icicle"
	FlagR1CS            = "r1cs"
	FlagEmail           = "email"
	FlagVerifyProof     = "verify-proof"
	FlagVerifyDKIM      = "verify-dkim"
	FlagAccount         = "account"
	FlagAuthenticatorID = "authenticator-id"
)

func ProveCommand(_ module.BasicManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "prove",
		Aliases:                    []string{"p"},
		Short:                      "Proving subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		ProveZKEmailCmd(),
	)

	return cmd
}

type ZKEmailCircuit struct{}

func (circuit *ZKEmailCircuit) Define(_ frontend.API) error {
	// todo: fill from email body
	return nil
}

func ProveZKEmailCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zk-email",
		Short: "Create an abstract account signature via zk proof from a DMARC signed email body",
		Long:  `TODO: DESCRIBE`,
		Example: fmt.Sprintf(
			"$ %s prove zk-email --r1cs /path/to/r1cs --email path/to/file --icicle false",
			version.AppName,
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			emailBodyPath, err := cmd.Flags().GetString(FlagEmail)
			if err != nil {
				return err
			}

			rawEmailBody, err := readEmailFromFile(emailBodyPath)
			if err != nil {
				return err
			}

			// verify dkim records on email before expensive proving
			verifyDKIM, err := cmd.Flags().GetBool(FlagVerifyDKIM)
			if err != nil {
				return err
			}
			if verifyDKIM {
				verifications, err := dkim.Verify(strings.NewReader(rawEmailBody))
				if err != nil {
					return err
				}
				for _, v := range verifications {
					if v.Err != nil {
						return v.Err
					}
				}
			}

			// load the circuit
			r1csPath, err := cmd.Flags().GetString(FlagR1CS)
			if err != nil {
				return err
			}
			r1cs, err := readR1CSFromFile(r1csPath)
			if err != nil {
				return err
			}

			pk, vk, err := groth16.Setup(r1cs)
			if err != nil {
				return err
			}

			// convert the email into a witness
			assignment := ZKEmailCircuit{} // todo: fill circuit from email body
			witness, err := frontend.NewWitness(&assignment, ecc.BN254.ScalarField())
			if err != nil {
				return err
			}

			// create the proof, using GPU if configured
			var proof groth16.Proof
			icicle, err := cmd.Flags().GetBool(FlagIcicle)
			if err != nil {
				return err
			}
			if icicle {
				proof, err = groth16.Prove(r1cs, pk, witness, backend.WithIcicleAcceleration())
			} else {
				proof, err = groth16.Prove(r1cs, pk, witness)
			}
			if err != nil {
				return err
			}

			// verify proof output
			verifyProof, err := cmd.Flags().GetBool(FlagVerifyProof)
			if err != nil {
				return err
			}
			if verifyProof {
				publicWitness, err := witness.Public()
				if err != nil {
					return err
				}

				if err := groth16.Verify(proof, vk, publicWitness); err != nil {
					return err
				}
			}

			// load the email body and break it into components
			email, err := mail.ReadMessage(strings.NewReader(rawEmailBody))
			if err != nil {
				return err
			}
			var emailBody []byte
			_, err = email.Body.Read(emailBody)
			if err != nil {
				return err
			}

			emailComponents := bytes.Split(emailBody, []byte(`#`))
			if len(emailComponents) != 4 {
				return fmt.Errorf("expected 4 components, received %d", len(emailComponents))
			}

			// created the salted identifier from the sender and hash
			sender := email.Header.Get("From")
			emailSalt := string(emailComponents[2])
			emailHash, err := poseidonHash(strings.Join([]string{sender, emailSalt}, ""))
			if err != nil {
				return err
			}

			// todo: retrieve the meta account with this email identifier if not provided
			accountStr, err := cmd.Flags().GetString(FlagAccount)
			if err != nil {
				return err
			}
			accountAddr, err := sdk.AccAddressFromBech32(accountStr)
			if err != nil {
				return err
			}
			queryClient := authtypes.NewQueryClient(clientCtx)
			signerAcc, err := cli.GetSignerOfTx(queryClient, accountAddr)
			if err != nil {
				return err
			}
			authenticatorID, err := cmd.Flags().GetUint8(FlagAuthenticatorID)
			if err != nil {
				return err
			}

			// todo: construct the transaction
			txBody := emailComponents[1]
			stdTx, err := clientCtx.TxConfig.TxJSONDecoder()(txBody)
			if err != nil {
				return err
			}
			txBuilder, err := clientCtx.TxConfig.WrapTxBuilder(stdTx)
			if err != nil {
				return err
			}
			var proofBytes bytes.Buffer
			if _, err := proof.WriteTo(&proofBytes); err != nil {
				return err
			}
			sigBytes := append([]byte{authenticatorID}, proofBytes.Bytes()...)
			sigData := signing.SingleSignatureData{
				SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
				Signature: sigBytes,
			}

			sig := signing.SignatureV2{
				PubKey:   signerAcc.GetPubKey(),
				Data:     &sigData,
				Sequence: signerAcc.GetSequence(),
			}
			err = txBuilder.SetSignatures(sig)

			// output the transaction
			tx := txBuilder.GetTx()
			txJson, err := clientCtx.TxConfig.TxJSONEncoder()(tx)
			if err != nil {
				return err
			}

			return clientCtx.PrintBytes(txJson)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().Bool(FlagIcicle, false, "https://github.com/Consensys/gnark?tab=readme-ov-file#gpu-support")
	cmd.Flags().String(FlagR1CS, "path/to/r1cs", "the file that contains the r1cs circuit definitions")
	cmd.Flags().String(FlagEmail, "path/to/eml", "the file that contains the email raw body")
	cmd.Flags().Bool(FlagVerifyProof, false, "whether or not the prover should verify the generated proof, primarily for debugging")
	cmd.Flags().Bool(FlagVerifyDKIM, true, "whether or not the client should check dns dkim records for the email, disable primarily for debugging")
	cmd.Flags().String(FlagAccount, "", "that account that should submit the transaction")
	cmd.Flags().Uint8(FlagAuthenticatorID, 0, "the authenticator id for the account")
	// _ = cmd.MarkFlagRequired(FlagQuery)

	return cmd
}

func readR1CSFromFile(filename string) (r1cs constraint.ConstraintSystem, err error) {
	var buf []byte

	if filename == "-" {
		buf, err = io.ReadAll(os.Stdin)
	} else {
		buf, err = os.ReadFile(filename)
	}

	if err != nil {
		return
	}

	r1cs = groth16.NewCS(ecc.BN254)
	if _, err := r1cs.ReadFrom(bytes.NewBuffer(buf)); err != nil {
		return nil, err
	}

	return
}

func readEmailFromFile(filename string) (email string, err error) {
	var buf []byte

	if filename == "-" {
		buf, err = io.ReadAll(os.Stdin)
	} else {
		buf, err = os.ReadFile(filename)
	}

	if err != nil {
		return
	}

	return string(buf), nil
}

func poseidonHash(input string) (hash []byte, err error) {
	println(input)
	return
}
