package integration_tests

import (
	"context"
	"crypto"
	cryptoRand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	mintTypes "github.com/burnt-labs/xion/x/mint/types"

	// ibclocalhost "github.com/cosmos/ibc-go/v10/modules/light-clients/09-localhost"

	"cosmossdk.io/math"
	wasmbinding "github.com/burnt-labs/xion/wasmbindings"
	"github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	authTx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncbor"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/icza/dyno"
	"github.com/strangelove-ventures/interchaintest/v10/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v10/ibc"
	"github.com/strangelove-ventures/interchaintest/v10/testutil"
	tokenfactorytypes "github.com/strangelove-ventures/tokenfactory/x/tokenfactory/types"
	"github.com/stretchr/testify/require"
)

func RawJSONMsgSend(t *testing.T, from, to, denom string) []byte {
	msg := fmt.Sprintf(`
{
  "body": {
    "messages": [
      {
        "@type": "/cosmos.bank.v1beta1.MsgSend",
        "from_address": "%s",
        "to_address": "%s",
        "amount": [
          {
            "denom": "%s",
            "amount": "100000"
          }
        ]
      }
    ],
    "memo": "",
    "timeout_height": "0",
    "extension_options": [],
    "non_critical_extension_options": []
  },
  "auth_info": {
    "signer_infos": [],
    "fee": {
      "amount": [],
      "gas_limit": "200000",
      "payer": "",
      "granter": ""
    },
    "tip": null
  },
  "signatures": []
}
	`, from, to, denom)
	var rawMsg json.RawMessage = []byte(msg)
	return rawMsg
}

func RawJSONMsgExecContractRemoveAuthenticator(sender string, contract string, index uint64) []byte {
	msg := fmt.Sprintf(`
{
  "body": {
    "messages": [
      {
        "@type": "/cosmwasm.wasm.v1.MsgExecuteContract",
        "sender": "%s",
        "contract": "%s",
        "msg": {
			"remove_auth_method": {
				"id": %d
			}
        },
        "funds": []
      }
    ],
    "memo": "",
    "timeout_height": "0",
    "extension_options": [],
    "non_critical_extension_options": []
  },
  "auth_info": {
    "signer_infos": [],
    "fee": {
      "amount": [],
      "gas_limit": "200000",
      "payer": "",
      "granter": ""
    },
    "tip": null
  },
  "signatures": []
}
	`, sender, contract, index)
	var rawMsg json.RawMessage = []byte(msg)
	return rawMsg
}

func RawJSONMsgMigrateContract(sender string, codeID string) []byte {
	msg := fmt.Sprintf(`

{
  "body": {
    "messages": [
    {
      "@type": "/cosmwasm.wasm.v1.MsgMigrateContract",
      "sender": "%s",
      "contract": "%s",
      "code_id": "%s",
      "msg": {}
    }
    ],
    "memo": "",
    "timeout_height": "0",
    "extension_options": [],
    "non_critical_extension_options": []
  },
  "auth_info": {
    "signer_infos": [],
    "fee": {
      "amount": [],
      "gas_limit": "200000",
      "payer": "",
      "granter": ""
    },
    "tip": null
  },
  "signatures": []
}
	`, sender, sender, codeID)
	var rawMsg json.RawMessage = []byte(msg)
	return rawMsg
}

// Helper method to retrieve the total token supply for a chain at some particular history denoted by the block height
func getTotalCoinSupplyInBank(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, denom string, blockHeight uint64) string {
	if blockHeight == 0 {
		// No history is required so use the most recent block height
		bHeight, err := xion.Height(ctx)
		require.NoError(t, err)
		blockHeight = uint64(bHeight)
		require.Greater(t, blockHeight, 0)
	}

	/*
	 * Response is of the structure
	 * {"supply":[{"denom":"uxion","amount":"110000002059725"}]}
	 */
	jsonRes := make(map[string]interface{})
	queryRes, _, err := xion.GetNode().ExecQuery(ctx, "bank", "total", "--height", strconv.FormatInt(int64(blockHeight), 10))
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(queryRes, &jsonRes))

	// Presuming we are the only denom on the chain then the returned array only has one Coin type ours (uxion)
	totalSupply, err := dyno.GetSlice(jsonRes, "supply")
	require.NoError(t, err)
	xionCoin := totalSupply[0]
	require.NotEmpty(t, xionCoin)
	// Make sure we selected the uxion denom
	xionCoinDenom, err := dyno.GetString(xionCoin, "denom")
	require.NoError(t, err)
	require.Equal(t, xionCoinDenom, denom)
	// Get the returned amount
	initialXionSupply, err := dyno.GetString(xionCoin, "amount")
	require.NoError(t, err)
	return initialXionSupply
}

// This function gets the bank balance for an address at some particular history denoted by the block height
func getAddressBankBalanceAtHeight(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, address string, denom string, blockHeight uint64) string {
	if blockHeight == 0 {
		bHeight, err := xion.Height(ctx)
		require.NoError(t, err)
		blockHeight = uint64(bHeight)
		require.Greater(t, blockHeight, 0)
	}

	/*
	 * Response is of the structure
	 * {"supply":[{"denom":"uxion","amount":"110000002059725"}]}
	 */
	jsonRes := make(map[string]interface{})
	queryRes, _, err := xion.GetNode().ExecQuery(ctx, "bank", "balances", address, "--height", strconv.FormatInt(int64(blockHeight), 10))
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(queryRes, &jsonRes))

	balances, err := dyno.GetSlice(jsonRes, "balances")
	require.NoError(t, err)
	if len(balances) == 0 {
		return "0"
	}
	// Make sure we selected the uxion denom
	balanceDenom, err := dyno.GetString(balances[0], "denom")
	require.NoError(t, err)
	require.Equal(t, balanceDenom, denom)
	balance, err := dyno.GetString(balances[0], "amount")
	require.NoError(t, err)
	t.Logf("Balance for address %s: %s", address, balance)
	return balance
}

// This function gets the module address for a module name
func GetModuleAddress(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, moduleName string) string {
	/*
		* Response is of the type
		* {
			"account": {
				"@type": "/cosmos.auth.v1beta1.ModuleAccount",
				"base_account": {
				"address": "xion1jv65s3grqf6v6jl3dp4t6c9t9rk99cd89k77l5",
				"pub_key": null,
				"account_number": "3",
				"sequence": "0"
				},
				"name": "distribution",
				"permissions": []
			}
			}
	*/
	jsonRes := make(map[string]interface{})
	queryRes, _, err := xion.GetNode().ExecQuery(ctx, "auth", "module-account", moduleName)
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(queryRes, &jsonRes))

	moduleAddress, err := dyno.GetString(jsonRes, "account", "value", "address")
	require.NoError(t, err)
	t.Logf("%s module address: %s", moduleName, moduleAddress)
	return moduleAddress
}

// Retrieve a block annual provision. This is the minted tokens for the block for validators and delegator
func GetBlockAnnualProvision(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, denom string, blockHeight uint64) math.LegacyDec {
	if blockHeight == 0 {
		bHeight, err := xion.Height(ctx)
		require.NoError(t, err)
		blockHeight = uint64(bHeight)
		require.Greater(t, blockHeight, 0)
	}

	// query the current block provision
	// response is a string
	var annualProvisionResponse mintTypes.QueryAnnualProvisionsResponse
	queryRes, _, err := xion.GetNode().ExecQuery(ctx, "mint", "annual-provisions", "--height", strconv.FormatInt(int64(blockHeight), 10))
	t.Logf("annual provission: %v", queryRes)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(queryRes, &annualProvisionResponse))

	// Query the block per year
	params := make(map[string]interface{})
	queryRes, _, err = xion.GetNode().ExecQuery(ctx, "mint", "params")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(queryRes, &params))
	fmt.Printf("%+v\n", params)
	blocksPerYear, err := dyno.GetInteger(params, "params", "blocks_per_year")
	require.NoError(t, err)

	// Calculate the block provision
	return math.LegacyMustNewDecFromStr(annualProvisionResponse.AnnualProvisions.String()).QuoInt(math.NewInt(blocksPerYear))
}

// This test confirms the property of the module described at
// https://www.notion.so/burntlabs/Mint-Module-Blog-Post-78f59fb108c04e9ea5fa826dda30a340
/*
 * The mint harness test logic is as follows
 * Given a particular block height, we compute the minted or burned tokens at that block using the following algorithm

 * Get the total token supply at the previous height
 * Get the total token supply at the current height
 * Compute the change in the total supply (d)
 * Get the balance of the distribution module account in the previous block
 * Get the balance of the distribution module account at the current block
 * The difference (f) is the sum of the total fees and minted token at the current block
 * The difference ( d - f )is the total fees accrued at the current block
 * Get the block provision at the current height (this is the number of tokens to be minted for validators and delegators at that block
 * Perform the following checks
 * If we have more fees accrued than the block provision, we burn the excess tokens
 * If instead there are fewer fees than block provision, we only mint (provision - fees) tokens to meet expectations
 * Otherwise, we had equal fees and provisions. We do not mint or burn any tokens
 * If these checks passes then the mint module functions as expected
 */
func MintModuleTestHarness(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, blockHeight int, assertion func(*testing.T, math.LegacyDec, int64, int64)) {
	// Get bank supply at previous height
	previousXionBankSupply, err := strconv.ParseInt(getTotalCoinSupplyInBank(t, xion, ctx, xion.Config().Denom, uint64(blockHeight-1)), 10, 64)
	t.Logf("Previous Xion bank supply: %d", previousXionBankSupply)
	require.NoError(t, err, "bank supply should be convertible to an int64")
	// Get bank supply at current height
	currentXionBankSupply, err := strconv.ParseInt(getTotalCoinSupplyInBank(t, xion, ctx, xion.Config().Denom, uint64(blockHeight)), 10, 64)
	t.Logf("Current Xion bank supply: %d", currentXionBankSupply)
	require.NoError(t, err, "bank supply should be convertible to an int64")
	tokenChange := currentXionBankSupply - previousXionBankSupply

	// Get the distribution module account address
	distributionModuleAddress := GetModuleAddress(t, xion, ctx, "distribution")
	// Get distribution module account balance in previous height
	previousDistributionModuleBalance, err := strconv.ParseInt(getAddressBankBalanceAtHeight(t, xion, ctx, distributionModuleAddress, xion.Config().Denom, uint64(blockHeight-1)), 10, 64)
	require.NoError(t, err, "distribution module balance should be convertible to an int64")
	// Get distribution module account balance in current height
	currentDistributionModuleBalance, err := strconv.ParseInt(getAddressBankBalanceAtHeight(t, xion, ctx, distributionModuleAddress, xion.Config().Denom, uint64(blockHeight)), 10, 64)
	require.NoError(t, err, "distribution module balance should be convertible to an int64")

	delta := currentDistributionModuleBalance - previousDistributionModuleBalance

	// Fees Accrued is the total fees in a block. Since the tokens in the distribution module account
	// are the fees and the minted tokens, we can compute the fees accrued by subtracting the token change
	// from the delta
	feesAccrued := delta - tokenChange
	t.Logf("Fees accrued: %d", feesAccrued)

	// Get the block provision. This is the minted tokens for the block for validators and delegator
	blockProvision := GetBlockAnnualProvision(t, xion, ctx, xion.Config().Denom, uint64(blockHeight))

	assertion(t, blockProvision, feesAccrued, tokenChange)
	/*
		if blockProvision.TruncateInt().GT(math.NewInt(feesAccrued)) {
			// We have minted tokens because the fees accrued is less than the block provision
			mintedTokens := blockProvision.TruncateInt().Sub(math.NewInt(feesAccrued))
			t.Logf("Minted tokens: %d and Token change: %d", mintedTokens.Int64(), int64(tokenChange))
			require.Equal(t, mintedTokens, math.NewInt(int64(tokenChange)))
		} else if blockProvision.TruncateInt().LT(math.NewInt(feesAccrued)) {
			// We have burned tokens because the fees accrued is greater than the block provision so the fees
			// accrued are used to pay validators and the remaining is burned
			burnedTokens := math.NewInt(feesAccrued).Sub(blockProvision.TruncateInt())
			t.Logf("Burned tokens: %d and Token change: %d", burnedTokens.Int64(), tokenChange)
			require.Equal(t, burnedTokens, math.NewInt(tokenChange).Abs())
		} else {
			// We have not minted or burned tokens but just used all fees accrued to pay validators
			require.Equal(t, math.NewInt(0), math.NewInt(tokenChange))
			t.Logf("No minted or Burned tokens. Token change: %d", tokenChange)
		}
	*/
}

// Run Mint module test harness over some random block height
// Chain must have at least 12 blocks
func VerifyMintModuleTestRandomBlocks(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, assertion func(*testing.T, math.LegacyDec, int64, int64)) {
	currentBlockHeight, err := xion.Height(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, currentBlockHeight, int64(12))
	// Get a random number from 1 to the (currentBlockHeight - 10)
	randomHeight := rand.Intn(int(currentBlockHeight)-11) + 2 // we start from 2 because we need at least 2 blocks to run the test

	for i := randomHeight; i < randomHeight+10; i++ {
		t.Logf("Current random height: %d", i)
		MintModuleTestHarness(t, xion, ctx, i, assertion)
	}
}

// Run Mint module test over some txHash
func VerifyMintModuleTest(t *testing.T, xion *cosmos.CosmosChain, ctx context.Context, txHashes []string, assertion func(*testing.T, math.LegacyDec, int64, int64)) {
	for i, txHash := range txHashes {
		txResp, err := authTx.QueryTx(xion.GetNode().CliContext(), txHash)
		require.NoError(t, err)
		t.Logf("Bank send msg %d BH: %d", i, txResp.Height)
		MintModuleTestHarness(t, xion, ctx, int(txResp.Height)+1, assertion) // check my block and the next one
	}
}

func TxCommandOverrideGas(t *testing.T, tn *cosmos.ChainNode, keyName, gas string, command ...string) []string {
	command = append([]string{"tx"}, command...)
	return tn.NodeCommand(append(command,
		"--from", keyName,
		"--gas-prices", gas,
		"--gas-adjustment", fmt.Sprint(tn.Chain.Config().GasAdjustment),
		"--gas", "auto",
		"--keyring-backend", keyring.BackendTest,
		"--output", "json",
		"-y",
	)...)
}

func ExecTx(t *testing.T, ctx context.Context, tn *cosmos.ChainNode, keyName string, command ...string) (string, error) {
	cmd := TxCommandOverrideGas(t, tn, keyName, tn.Chain.Config().GasPrices, command...)
	t.Logf("cmd: %s", cmd)
	stdout, _, err := tn.Exec(ctx, cmd, nil)
	if err != nil {
		return "", err
	}
	output := cosmos.CosmosTx{}
	err = json.Unmarshal(stdout, &output)
	if err != nil {
		return "", err
	}
	if output.Code != 0 {
		return output.TxHash, fmt.Errorf("transaction failed with code %d: %s", output.Code, output.RawLog)
	}
	if err := testutil.WaitForBlocks(ctx, 2, tn); err != nil {
		return "", err
	}
	return output.TxHash, nil
}

func ExecTxWithGas(t *testing.T, ctx context.Context, tn *cosmos.ChainNode, keyName string, gas string, command ...string) (string, error) {
	cmd := TxCommandOverrideGas(t, tn, keyName, gas, command...)
	t.Logf("cmd: %s", cmd)
	stdout, _, err := tn.Exec(ctx, cmd, nil)
	if err != nil {
		return "", err
	}
	output := cosmos.CosmosTx{}
	err = json.Unmarshal(stdout, &output)
	if err != nil {
		return "", err
	}
	if output.Code != 0 {
		return output.TxHash, fmt.Errorf("transaction failed with code %d: %s", output.Code, output.RawLog)
	}
	if err := testutil.WaitForBlocks(ctx, 2, tn); err != nil {
		return "", err
	}
	return output.TxHash, nil
}

func GenerateTx(t *testing.T, ctx context.Context, tn *cosmos.ChainNode, keyName string, command ...string) (string, error) {
	cmd := append([]string{"tx"}, command...)
	cmd = tn.NodeCommand(append(cmd,
		"--from", keyName,
		"--keyring-backend", keyring.BackendTest,
		"--output", "json",
		"--generate-only",
	)...)
	t.Logf("cmd: %s", cmd)
	stdout, _, err := tn.Exec(ctx, cmd, nil)
	if err != nil {
		return "", err
	}
	return string(stdout), nil
}

func ExecQuery(t *testing.T, ctx context.Context, tn *cosmos.ChainNode, command ...string) (map[string]interface{}, error) {
	jsonRes := make(map[string]interface{})
	t.Logf("querying with cmd: %s", command)
	output, _, err := tn.ExecQuery(ctx, command...)
	if err != nil {
		return jsonRes, err
	}
	require.NoError(t, json.Unmarshal(output, &jsonRes))

	return jsonRes, nil
}

func ExecBin(t *testing.T, ctx context.Context, tn *cosmos.ChainNode, command ...string) (map[string]interface{}, error) {
	jsonRes := make(map[string]interface{})
	output, _, err := tn.ExecBin(ctx, command...)
	if err != nil {
		return jsonRes, err
	}
	require.NoError(t, json.Unmarshal(output, &jsonRes))

	return jsonRes, nil
}

func ExecBinStr(t *testing.T, ctx context.Context, tn *cosmos.ChainNode, command ...string) (string, error) {
	output, _, err := tn.ExecBin(ctx, command...)
	require.NoError(t, err)
	return string(output), nil
}

func ExecBinRaw(t *testing.T, ctx context.Context, tn *cosmos.ChainNode, command ...string) ([]byte, error) {
	output, _, err := tn.ExecBin(ctx, command...)
	if err != nil {
		return output, err
	}

	return output, nil
}

func ExecBroadcast(_ *testing.T, ctx context.Context, tn *cosmos.ChainNode, tx []byte) (string, error) {
	if err := tn.WriteFile(ctx, tx, "tx.json"); err != nil {
		return "", err
	}

	cmd := tn.NodeCommand("tx", "broadcast", path.Join(tn.HomeDir(), "tx.json"), "--output", "json")

	stdout, _, err := tn.Exec(ctx, cmd, nil)
	if err != nil {
		return "", err
	}
	return string(stdout), err
}

func ExecBroadcastWithFlags(_ *testing.T, ctx context.Context, tn *cosmos.ChainNode, tx []byte, command ...string) (string, error) {
	if err := tn.WriteFile(ctx, tx, "tx.json"); err != nil {
		return "", err
	}
	c := append([]string{"tx", "broadcast", path.Join(tn.HomeDir(), "tx.json")}, command...)
	cmd := tn.NodeCommand(c...)

	stdout, _, err := tn.Exec(ctx, cmd, nil)
	if err != nil {
		return "", err
	}

	output := cosmos.CosmosTx{}
	err = json.Unmarshal(stdout, &output)
	if err != nil {
		return "", err
	}
	if output.Code != 0 {
		return output.TxHash, fmt.Errorf("transaction failed with code %d: %s", output.Code, output.RawLog)
	}
	if err := testutil.WaitForBlocks(ctx, 2, tn); err != nil {
		return "", err
	}
	return output.TxHash, err
}

func UploadFileToContainer(t *testing.T, ctx context.Context, tn *cosmos.ChainNode, file *os.File) error {
	content, err := os.ReadFile(file.Name())
	if err != nil {
		return err
	}
	path := strings.Split(file.Name(), "/")
	return tn.WriteFile(ctx, content, path[len(path)-1])
}

type signOpts struct{}

func (*signOpts) HashFunc() crypto.Hash {
	return crypto.SHA256
}

var (
	credentialID = []byte("UWxY-yRdIls8IT-vyMS6la1ZiqESOAff7bWZ_LWV0Pg")
	AAGUID       = []byte("AAGUIDAAGUIDAA==")
)

func getWebAuthNKeys(t *testing.T) (*rsa.PrivateKey, []byte, webauthncose.RSAPublicKeyData) {
	privateKey, _, err := wasmbinding.SetupPublicKeys("./integration_tests/testdata/keys/jwtRS256.key")
	require.NoError(t, err)
	publicKey := privateKey.PublicKey
	publicKeyModulus := publicKey.N.Bytes()
	require.NoError(t, err)
	pubKeyData := webauthncose.RSAPublicKeyData{
		PublicKeyData: webauthncose.PublicKeyData{
			KeyType:   int64(webauthncose.RSAKey),
			Algorithm: int64(webauthncose.AlgRS256),
		},
		Modulus:  publicKeyModulus,
		Exponent: big.NewInt(int64(publicKey.E)).Bytes(),
	}
	publicKeyBuf, err := webauthncbor.Marshal(pubKeyData)
	require.NoError(t, err)
	return privateKey, publicKeyBuf, pubKeyData
}

func CreateWebAuthn(t *testing.T) (webauthn.Config, *webauthn.WebAuthn, error) {
	webAuthnConfig := webauthn.Config{
		RPDisplayName:         "Xion",
		RPID:                  "xion-dapp-example-git-feat-faceid-burntfinance.vercel.app",
		RPOrigins:             []string{"https://xion-dapp-example-git-feat-faceid-burntfinance.vercel.app"},
		AttestationPreference: "none",
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.Platform,
			UserVerification:        protocol.VerificationPreferred,
		},
	}
	webAuthn, err := webauthn.New(&webAuthnConfig)
	require.NoError(t, err)

	return webAuthnConfig, webAuthn, nil
}

func CreateWebAuthNAttestationCred(t *testing.T, challenge string) []byte {
	webAuthnConfig, _, err := CreateWebAuthn(t)
	require.NoError(t, err)
	clientData := protocol.CollectedClientData{
		Type:      "webauthn.create",
		Challenge: challenge,
		Origin:    "https://xion-dapp-example-git-feat-faceid-burntfinance.vercel.app",
	}

	_, publicKeyBuf, _ := getWebAuthNKeys(t)

	clientDataJSON, err := json.Marshal(clientData)
	require.NoError(t, err)

	RPIDHash := sha256.Sum256([]byte(webAuthnConfig.RPID))
	authData := protocol.AuthenticatorData{
		RPIDHash: RPIDHash[:],
		Counter:  0,
		AttData: protocol.AttestedCredentialData{
			AAGUID:              AAGUID,
			CredentialID:        credentialID,
			CredentialPublicKey: publicKeyBuf,
		},
		Flags: 69,
	}
	authDataBz := append(RPIDHash[:], big.NewInt(69).Bytes()[:]...)
	counterBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(counterBytes, 0)
	authDataBz = append(authDataBz, counterBytes...)

	var attData []byte

	// Concatenate AAGUID
	attData = append(attData, AAGUID...)

	// Add CredentialIDLength
	credentialIDLengthBytes := make([]byte, 2)
	credentialIDLength := uint16(len(credentialID))
	binary.BigEndian.PutUint16(credentialIDLengthBytes, credentialIDLength)
	attData = append(attData, credentialIDLengthBytes...)

	// Add CredentialID
	attData = append(attData, credentialID...)

	// Add CredentialPublicKey
	attData = append(attData, publicKeyBuf...)

	authDataBz = append(authDataBz, attData...)

	attestationObject := protocol.AttestationObject{
		AuthData:    authData,
		RawAuthData: authDataBz,
		Format:      "none",
	}
	attestationObjectJSON, err := webauthncbor.Marshal(attestationObject)
	require.NoError(t, err)

	attestationResponse := protocol.AuthenticatorAttestationResponse{
		AuthenticatorResponse: protocol.AuthenticatorResponse{
			ClientDataJSON: protocol.URLEncodedBase64(clientDataJSON),
		},
		AttestationObject: attestationObjectJSON,
	}
	_, err = attestationResponse.Parse()
	require.NoError(t, err)

	credentialCreationResponse := protocol.CredentialCreationResponse{
		PublicKeyCredential: protocol.PublicKeyCredential{
			Credential: protocol.Credential{
				ID:   string(credentialID),
				Type: "public-key",
			},
			RawID:                   credentialID,
			AuthenticatorAttachment: string(protocol.Platform),
		},
		AttestationResponse: attestationResponse,
	}
	_, err = credentialCreationResponse.Parse()
	require.NoError(t, err)

	credCreationRes, err := json.Marshal(credentialCreationResponse)
	require.NoError(t, err)
	return credCreationRes
}

func CreateWebAuthNSignature(t *testing.T, challenge string, address string) []byte {
	webAuthnConfig, _, err := CreateWebAuthn(t)
	require.NoError(t, err)
	privateKey, publicKeyBuf, pubKeyData := getWebAuthNKeys(t)

	webAuthnUser := types.SmartContractUser{
		Address: address,
		Credential: &webauthn.Credential{
			ID:              credentialID,
			AttestationType: "none",
			PublicKey:       publicKeyBuf,
			Transport:       []protocol.AuthenticatorTransport{protocol.Internal},
			Flags: webauthn.CredentialFlags{
				UserPresent:  false,
				UserVerified: false,
			},
			Authenticator: webauthn.Authenticator{
				AAGUID:     AAGUID,
				SignCount:  0,
				Attachment: protocol.Platform,
			},
		},
	}

	RPIDHash := sha256.Sum256([]byte(webAuthnConfig.RPID))
	clientData := protocol.CollectedClientData{
		Type:      "webauthn.get",
		Challenge: challenge,
		Origin:    "https://xion-dapp-example-git-feat-faceid-burntfinance.vercel.app",
	}
	clientDataJSON, err := json.Marshal(clientData)
	require.NoError(t, err)
	clientDataHash := sha256.Sum256(clientDataJSON)
	authDataBz := append(RPIDHash[:], big.NewInt(69).Bytes()[:]...)
	counterBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(counterBytes, 0)
	authDataBz = append(authDataBz, counterBytes...)

	var attData []byte

	// Concatenate AAGUID
	attData = append(attData, AAGUID...)

	// Add CredentialIDLength
	credentialIDLengthBytes := make([]byte, 2)
	credentialIDLength := uint16(len(credentialID))
	binary.BigEndian.PutUint16(credentialIDLengthBytes, credentialIDLength)
	attData = append(attData, credentialIDLengthBytes...)

	// Add CredentialID
	attData = append(attData, credentialID...)

	// Add CredentialPublicKey
	attData = append(attData, publicKeyBuf...)

	authDataBz = append(authDataBz, attData...)
	require.NoError(t, err)

	signData := append(authDataBz[:], clientDataHash[:]...)
	signHash := sha256.Sum256(signData)
	signature, err := privateKey.Sign(cryptoRand.Reader, signHash[:], &signOpts{})
	require.NoError(t, err)
	verified, err := pubKeyData.Verify(signData[:], signature)
	require.NoError(t, err)
	require.True(t, verified)

	credentialAssertionData := &protocol.CredentialAssertionResponse{
		PublicKeyCredential: protocol.PublicKeyCredential{
			Credential: protocol.Credential{
				ID:   string(credentialID),
				Type: "public-key",
			},
			RawID:                   credentialID,
			AuthenticatorAttachment: string(protocol.Platform),
		},
		AssertionResponse: protocol.AuthenticatorAssertionResponse{
			AuthenticatorResponse: protocol.AuthenticatorResponse{
				ClientDataJSON: protocol.URLEncodedBase64(clientDataJSON),
			},
			AuthenticatorData: authDataBz,
			Signature:         signature,
			UserHandle:        webAuthnUser.WebAuthnID(),
		},
	}
	// validate signature
	assertionResponse, err := json.Marshal(credentialAssertionData)
	require.NoError(t, err)
	return assertionResponse
}

func CreateTokenFactoryDenom(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet, subDenomName, feeCoin string) (fullDenom string) {
	// TF gas to create cost 2mil, so we set to 2.5 to be safe
	cmd := []string{
		"xiond", "tx", "tokenfactory", "create-denom", subDenomName,
		"--node", chain.GetRPCAddress(),
		"--home", chain.HomeDir(),
		"--chain-id", chain.Config().ChainID,
		"--from", user.KeyName(),
		"--gas", "2500000",
		"--keyring-dir", chain.HomeDir(),
		"--keyring-backend", keyring.BackendTest,
		"-y",
	}

	if feeCoin != "" {
		cmd = append(cmd, "--fees", feeCoin)
	}

	_, _, err := chain.Exec(ctx, cmd, nil)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 2, chain)
	require.NoError(t, err)

	return "factory/" + user.FormattedAddress() + "/" + subDenomName
}

func MintTokenFactoryDenom(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, admin ibc.Wallet, amount uint64, fullDenom string) {
	denom := strconv.FormatUint(amount, 10) + fullDenom

	// mint new tokens to the account
	cmd := []string{
		"xiond", "tx", "tokenfactory", "mint", denom,
		"--node", chain.GetRPCAddress(),
		"--home", chain.HomeDir(),
		"--chain-id", chain.Config().ChainID,
		"--from", admin.KeyName(),
		"--keyring-dir", chain.HomeDir(),
		"--keyring-backend", keyring.BackendTest,
		"-y",
	}
	_, _, err := chain.Exec(ctx, cmd, nil)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 2, chain)
	require.NoError(t, err)
}

func MintToTokenFactoryDenom(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, admin ibc.Wallet, toWallet ibc.Wallet, amount uint64, fullDenom string) {
	denom := strconv.FormatUint(amount, 10) + fullDenom

	receiver := toWallet.FormattedAddress()

	t.Log("minting", denom, "to", receiver)

	// mint new tokens to the account
	cmd := []string{
		"xiond", "tx", "tokenfactory", "mint-to", receiver, denom,
		"--node", chain.GetRPCAddress(),
		"--home", chain.HomeDir(),
		"--chain-id", chain.Config().ChainID,
		"--from", admin.KeyName(),
		"--keyring-dir", chain.HomeDir(),
		"--keyring-backend", keyring.BackendTest,
		"-y",
	}
	_, _, err := chain.Exec(ctx, cmd, nil)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 2, chain)
	require.NoError(t, err)
}

func TransferTokenFactoryAdmin(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, currentAdmin ibc.Wallet, newAdminBech32 string, fullDenom string) {
	cmd := []string{
		"xiond", "tx", "tokenfactory", "change-admin", fullDenom, newAdminBech32,
		"--node", chain.GetRPCAddress(),
		"--home", chain.HomeDir(),
		"--chain-id", chain.Config().ChainID,
		"--from", currentAdmin.KeyName(),
		"--keyring-dir", chain.HomeDir(),
		"--keyring-backend", keyring.BackendTest,
		"-y",
	}
	_, _, err := chain.Exec(ctx, cmd, nil)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 2, chain)
	require.NoError(t, err)
}

func GetTokenFactoryAdmin(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, fullDenom string) string {
	cmd := []string{
		"xiond", "query", "tokenfactory", "denom-authority-metadata", fullDenom,
		"--node", chain.GetRPCAddress(),
		//"--chain-id", chain.Config().ChainID,
		"--output", "json",
	}
	stdout, _, err := chain.Exec(ctx, cmd, nil)
	require.NoError(t, err)

	results := &tokenfactorytypes.QueryDenomAuthorityMetadataResponse{}
	err = json.Unmarshal(stdout, results)
	require.NoError(t, err)

	t.Log(results)

	return results.AuthorityMetadata.Admin
}
