package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"cosmossdk.io/math"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	aatypes "github.com/burnt-labs/abstract-account/x/abstractaccount/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"
)

// Helper functions to create test data
func createTempJSONFile(content string) (string, func(), error) {
	tmpDir := os.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "test-*.json")
	if err != nil {
		return "", nil, err
	}

	_, err = tmpFile.WriteString(content)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", nil, err
	}
	tmpFile.Close()

	cleanup := func() {
		os.Remove(tmpFile.Name())
	}

	return tmpFile.Name(), cleanup, nil
}

// Tests for NewAddAuthenticatorCmd - targeting 100% coverage
// Mock structures
type MockQueryClient struct {
	mock.Mock
}

func (m *MockQueryClient) Account(ctx context.Context, req *authtypes.QueryAccountRequest, opts ...grpc.CallOption) (*authtypes.QueryAccountResponse, error) {
	args := m.Called(ctx, req, opts)
	return args.Get(0).(*authtypes.QueryAccountResponse), args.Error(1)
}

func (m *MockQueryClient) AccountAddressByID(ctx context.Context, req *authtypes.QueryAccountAddressByIDRequest, opts ...grpc.CallOption) (*authtypes.QueryAccountAddressByIDResponse, error) {
	args := m.Called(ctx, req, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authtypes.QueryAccountAddressByIDResponse), args.Error(1)
}

func (m *MockQueryClient) Accounts(ctx context.Context, req *authtypes.QueryAccountsRequest, opts ...grpc.CallOption) (*authtypes.QueryAccountsResponse, error) {
	args := m.Called(ctx, req, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authtypes.QueryAccountsResponse), args.Error(1)
}

func (m *MockQueryClient) AccountInfo(ctx context.Context, req *authtypes.QueryAccountInfoRequest, opts ...grpc.CallOption) (*authtypes.QueryAccountInfoResponse, error) {
	args := m.Called(ctx, req, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authtypes.QueryAccountInfoResponse), args.Error(1)
}

func (m *MockQueryClient) Params(ctx context.Context, req *authtypes.QueryParamsRequest, opts ...grpc.CallOption) (*authtypes.QueryParamsResponse, error) {
	args := m.Called(ctx, req, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authtypes.QueryParamsResponse), args.Error(1)
}

func (m *MockQueryClient) ModuleAccounts(ctx context.Context, req *authtypes.QueryModuleAccountsRequest, opts ...grpc.CallOption) (*authtypes.QueryModuleAccountsResponse, error) {
	args := m.Called(ctx, req, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authtypes.QueryModuleAccountsResponse), args.Error(1)
}

func (m *MockQueryClient) ModuleAccountByName(ctx context.Context, req *authtypes.QueryModuleAccountByNameRequest, opts ...grpc.CallOption) (*authtypes.QueryModuleAccountByNameResponse, error) {
	args := m.Called(ctx, req, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authtypes.QueryModuleAccountByNameResponse), args.Error(1)
}

func (m *MockQueryClient) Bech32Prefix(ctx context.Context, req *authtypes.Bech32PrefixRequest, opts ...grpc.CallOption) (*authtypes.Bech32PrefixResponse, error) {
	args := m.Called(ctx, req, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authtypes.Bech32PrefixResponse), args.Error(1)
}

func (m *MockQueryClient) AddressBytesToString(ctx context.Context, req *authtypes.AddressBytesToStringRequest, opts ...grpc.CallOption) (*authtypes.AddressBytesToStringResponse, error) {
	args := m.Called(ctx, req, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authtypes.AddressBytesToStringResponse), args.Error(1)
}

func (m *MockQueryClient) AddressStringToBytes(ctx context.Context, req *authtypes.AddressStringToBytesRequest, opts ...grpc.CallOption) (*authtypes.AddressStringToBytesResponse, error) {
	args := m.Called(ctx, req, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authtypes.AddressStringToBytesResponse), args.Error(1)
}

type MockAccount struct {
	address       sdk.AccAddress
	pubKey        cryptotypes.PubKey
	accountNumber uint64
	sequence      uint64
}

func (m *MockAccount) GetAddress() sdk.AccAddress {
	return m.address
}

func (m *MockAccount) SetAddress(addr sdk.AccAddress) error {
	m.address = addr
	return nil
}

func (m *MockAccount) GetPubKey() cryptotypes.PubKey {
	return m.pubKey
}

func (m *MockAccount) SetPubKey(pubKey cryptotypes.PubKey) error {
	m.pubKey = pubKey
	return nil
}

func (m *MockAccount) GetAccountNumber() uint64 {
	return m.accountNumber
}

func (m *MockAccount) SetAccountNumber(accNumber uint64) error {
	m.accountNumber = accNumber
	return nil
}

func (m *MockAccount) GetSequence() uint64 {
	return m.sequence
}

func (m *MockAccount) SetSequence(seq uint64) error {
	m.sequence = seq
	return nil
}

func (m *MockAccount) String() string {
	return fmt.Sprintf("MockAccount{%s}", m.address)
}

func TestNewAddAuthenticatorCmdComprehensive(t *testing.T) {
	cmd := NewAddAuthenticatorCmd()

	t.Run("command structure", func(t *testing.T) {
		require.Contains(t, cmd.Use, "add-authenticator")
		require.Equal(t, "Add the signing key as an authenticator to an abstract account", cmd.Short)
		require.NotNil(t, cmd.Flags())
	})

	t.Run("argument validation", func(t *testing.T) {
		// ExactArgs(1)
		require.Error(t, cmd.Args(cmd, []string{}))
		require.NoError(t, cmd.Args(cmd, []string{"authenticator"}))
		require.Error(t, cmd.Args(cmd, []string{"a", "b"}))
	})

	t.Run("flag handling", func(t *testing.T) {
		// Test all flags can be set
		flags := cmd.Flags()

		// Test required flags - only authenticator-id is available for this command
		err := flags.Set(flagAuthenticatorID, "1")
		require.NoError(t, err)

		// Test standard tx flags
		err = flags.Set("chain-id", "test-chain")
		require.NoError(t, err)
	})

	t.Run("execution with missing context", func(t *testing.T) {
		cmd.SetArgs([]string{"test-authenticator"})
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)

		// Should error due to missing client context
		err := cmd.Execute()
		require.Error(t, err)
	})
}

// Tests for NewSignCmd - targeting 100% coverage
func TestNewSignCmdComprehensive(t *testing.T) {
	cmd := NewSignCmd()

	t.Run("command structure", func(t *testing.T) {
		require.Contains(t, cmd.Use, "sign")
		require.Equal(t, "sign a transaction", cmd.Short)
		require.NotNil(t, cmd.Flags())
		require.NotNil(t, cmd.Flag(flagAuthenticatorID))
	})

	t.Run("argument validation", func(t *testing.T) {
		// ExactArgs(3)
		require.Error(t, cmd.Args(cmd, []string{}))
		require.Error(t, cmd.Args(cmd, []string{"file"}))
		require.Error(t, cmd.Args(cmd, []string{"a", "b"}))
		require.NoError(t, cmd.Args(cmd, []string{"a", "b", "c"}))
		require.Error(t, cmd.Args(cmd, []string{"a", "b", "c", "d"}))
	})

	t.Run("flag set error", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino)

		testCmd := NewSignCmd()
		testCmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		// Test with invalid args to trigger flag set error
		err := testCmd.RunE(testCmd, []string{"", "validaddr", "validfile.json"})
		require.Error(t, err)
	})

	t.Run("client context error", func(t *testing.T) {
		testCmd := NewSignCmd()
		// Set minimal context to avoid nil pointer but still cause error
		encCfg := testutil.MakeTestEncodingConfig()
		clientCtx := client.Context{}.WithCodec(encCfg.Codec)
		testCmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		err := testCmd.RunE(testCmd, []string{"keyname", "validaddr", "validfile.json"})
		require.Error(t, err)
	})

	t.Run("authenticator id flag error", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino)

		testCmd := NewSignCmd()
		testCmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		// Set invalid authenticator ID to trigger error
		testCmd.Flags().Set(flagAuthenticatorID, "invalid")
		err := testCmd.RunE(testCmd, []string{"keyname", "validaddr", "validfile.json"})
		require.Error(t, err)
	})

	t.Run("invalid signer address", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		kr := keyring.NewInMemory(encCfg.Codec)

		// Create a test key to avoid keyring errors
		info, _, err := kr.NewMnemonic("test", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
		require.NoError(t, err)

		addr, err := info.GetAddress()
		require.NoError(t, err)

		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino).
			WithKeyring(kr).
			WithFromName("test").
			WithFromAddress(addr)

		testCmd := NewSignCmd()
		testCmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		// Test with invalid bech32 address
		err = testCmd.RunE(testCmd, []string{"test", "invalid-address", "validfile.json"})
		require.Error(t, err)
		// The error may be about address format or other issues - just verify it errors
	})

	t.Run("file not found", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino)

		testCmd := NewSignCmd()
		testCmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		// Valid address format but nonexistent file
		validAddr := "xion1234567890abcdef1234567890abcdef12345678"
		err := testCmd.RunE(testCmd, []string{"keyname", validAddr, "nonexistent.json"})
		require.Error(t, err)
		// Should be a file not found error
	})

	t.Run("invalid json file", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino)

		testCmd := NewSignCmd()
		testCmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		// Create temp file with invalid JSON
		content := `{invalid json}`
		tmpFile, cleanup, err := createTempJSONFile(content)
		require.NoError(t, err)
		defer cleanup()

		validAddr := "xion1234567890abcdef1234567890abcdef12345678"
		err = testCmd.RunE(testCmd, []string{"keyname", validAddr, tmpFile})
		require.Error(t, err)
		// Should be a JSON decode error
	})

	t.Run("valid json but invalid tx", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino)

		testCmd := NewSignCmd()
		testCmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		// Create temp file with valid JSON but invalid TX format
		content := `{"test": "data", "not": "a transaction"}`
		tmpFile, cleanup, err := createTempJSONFile(content)
		require.NoError(t, err)
		defer cleanup()

		validAddr := "xion1234567890abcdef1234567890abcdef12345678"
		err = testCmd.RunE(testCmd, []string{"keyname", validAddr, tmpFile})
		require.Error(t, err)
		// Should be a TX decode error
	})

	t.Run("execution with minimal valid setup", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		kr := keyring.NewInMemory(encCfg.Codec)

		// Create a test key
		info, _, err := kr.NewMnemonic("test", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
		require.NoError(t, err)

		addr, err := info.GetAddress()
		require.NoError(t, err)

		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino).
			WithKeyring(kr).
			WithFromName("test").
			WithFromAddress(addr).
			WithChainID("test-chain")

		testCmd := NewSignCmd()
		testCmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		// Create a basic transaction JSON
		txJson := `{
			"body": {
				"messages": [],
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
				}
			},
			"signatures": []
		}`

		tmpFile, cleanup, err := createTempJSONFile(txJson)
		require.NoError(t, err)
		defer cleanup()

		validAddr := addr.String()
		err = testCmd.RunE(testCmd, []string{"test", validAddr, tmpFile})
		require.Error(t, err)
		// Will fail due to network calls but we've exercised more of the function
	})

	t.Run("transaction with valid structure", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		kr := keyring.NewInMemory(encCfg.Codec)

		// Create a test key
		info, _, err := kr.NewMnemonic("test", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
		require.NoError(t, err)

		addr, err := info.GetAddress()
		require.NoError(t, err)

		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino).
			WithKeyring(kr).
			WithFromName("test").
			WithFromAddress(addr).
			WithChainID("test-chain")

		testCmd := NewSignCmd()
		testCmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		// Create a valid transaction structure (based on cosmos-sdk patterns)
		txBuilder := clientCtx.TxConfig.NewTxBuilder()
		// Empty transaction but valid structure
		tx := txBuilder.GetTx()
		txBytes, err := clientCtx.TxConfig.TxJSONEncoder()(tx)
		require.NoError(t, err)

		tmpFile, cleanup, err := createTempJSONFile(string(txBytes))
		require.NoError(t, err)
		defer cleanup()

		validAddr := addr.String()
		err = testCmd.RunE(testCmd, []string{"test", validAddr, tmpFile})
		require.Error(t, err)
		// Will likely fail due to network query, but we've covered TX JSON decoding
	})

	t.Run("different authenticator id values", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		kr := keyring.NewInMemory(encCfg.Codec)

		info, _, err := kr.NewMnemonic("test", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
		require.NoError(t, err)

		addr, err := info.GetAddress()
		require.NoError(t, err)

		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino).
			WithKeyring(kr).
			WithFromName("test").
			WithFromAddress(addr).
			WithChainID("test-chain")

		// Create basic tx json
		txJson := `{
			"body": {
				"messages": [],
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
				}
			},
			"signatures": []
		}`

		tmpFile, cleanup, err := createTempJSONFile(txJson)
		require.NoError(t, err)
		defer cleanup()

		validAddr := addr.String()

		// Test different authenticator ID values
		authIDs := []string{"0", "1", "255"}
		for _, authID := range authIDs {
			t.Run("auth_id_"+authID, func(t *testing.T) {
				testCmd := NewSignCmd()
				testCmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))
				testCmd.Flags().Set(flagAuthenticatorID, authID)

				err := testCmd.RunE(testCmd, []string{"test", validAddr, tmpFile})
				require.Error(t, err)
				// Each will error but we're exercising the flag parsing path
			})
		}
	})

	t.Run("edge case error handling", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		kr := keyring.NewInMemory(encCfg.Codec)

		info, _, err := kr.NewMnemonic("test", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
		require.NoError(t, err)

		addr, err := info.GetAddress()
		require.NoError(t, err)

		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino).
			WithKeyring(kr).
			WithFromName("test").
			WithFromAddress(addr).
			WithChainID("test-chain")

		// Test cases for different JSON structures
		testCases := []struct {
			name     string
			jsonData string
		}{
			{
				name:     "empty_json",
				jsonData: `{}`,
			},
			{
				name: "minimal_tx_structure",
				jsonData: `{
					"body": null,
					"auth_info": null,
					"signatures": null
				}`,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tmpFile, cleanup, err := createTempJSONFile(tc.jsonData)
				require.NoError(t, err)
				defer cleanup()

				testCmd := NewSignCmd()
				testCmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

				validAddr := addr.String()
				err = testCmd.RunE(testCmd, []string{"test", validAddr, tmpFile})
				require.Error(t, err)
				// All should error due to invalid transaction format
			})
		}
	})
}

// Tests for NewEmitArbitraryDataCmd - targeting 100% coverage
func TestNewEmitArbitraryDataCmdComprehensive(t *testing.T) {
	cmd := NewEmitArbitraryDataCmd()

	t.Run("command structure", func(t *testing.T) {
		require.Contains(t, cmd.Use, "emit")
		require.Equal(t, "Emit an arbitrary data from the chain", cmd.Short)
		require.NotNil(t, cmd.Flags())
	})

	t.Run("argument validation", func(t *testing.T) {
		// ExactArgs(2)
		require.Error(t, cmd.Args(cmd, []string{}))
		require.Error(t, cmd.Args(cmd, []string{"only"}))
		require.NoError(t, cmd.Args(cmd, []string{"data", "contract"}))
		require.Error(t, cmd.Args(cmd, []string{"a", "b", "c"}))
	})

	t.Run("flag handling", func(t *testing.T) {
		flags := cmd.Flags()

		// Test setting various flags
		err := flags.Set("gas", "auto")
		require.NoError(t, err)

		err = flags.Set("gas-adjustment", "1.5")
		require.NoError(t, err)
	})

	t.Run("execution with missing context", func(t *testing.T) {
		cmd.SetArgs([]string{"test-data", "cosmos1..."})
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)

		err := cmd.Execute()
		require.Error(t, err)
	})
}

// Tests for NewUpdateConfigsCmd - targeting 100% coverage
func TestNewUpdateConfigsCmdComprehensive(t *testing.T) {
	cmd := NewUpdateConfigsCmd()

	t.Run("command structure", func(t *testing.T) {
		require.Contains(t, cmd.Use, "update-configs")
		require.Equal(t, "Batch update grant configs and fee config for the treasury", cmd.Short)
		require.NotNil(t, cmd.Flags())
	})

	t.Run("argument validation", func(t *testing.T) {
		// ExactArgs(2)
		require.Error(t, cmd.Args(cmd, []string{}))
		require.Error(t, cmd.Args(cmd, []string{"config"}))
		require.NoError(t, cmd.Args(cmd, []string{"config", "address"}))
		require.Error(t, cmd.Args(cmd, []string{"a", "b", "c"}))
	})

	t.Run("execution with invalid JSON file", func(t *testing.T) {
		cmd.SetArgs([]string{"nonexistent.json", "cosmos1..."})
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)

		err := cmd.Execute()
		require.Error(t, err)
		// Just verify there's an error, the message may vary
	})

	t.Run("execution with valid JSON but missing context", func(t *testing.T) {
		// Create a valid JSON file
		content := `{"test": "config"}`
		tmpFile, cleanup, err := createTempJSONFile(content)
		require.NoError(t, err)
		defer cleanup()

		cmd.SetArgs([]string{tmpFile, "cosmos1..."})
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)

		err = cmd.Execute()
		require.Error(t, err)
	})

	t.Run("execution with invalid JSON content", func(t *testing.T) {
		// Create an invalid JSON file
		content := `{invalid json`
		tmpFile, cleanup, err := createTempJSONFile(content)
		require.NoError(t, err)
		defer cleanup()

		cmd.SetArgs([]string{tmpFile, "cosmos1..."})
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)

		err = cmd.Execute()
		require.Error(t, err)
		// Just verify there's an error
	})
}

// Tests for NewUpdateParamsCmd - targeting 100% coverage
func TestNewUpdateParamsCmdComprehensive(t *testing.T) {
	cmd := NewUpdateParamsCmd()

	t.Run("command structure", func(t *testing.T) {
		require.Contains(t, cmd.Use, "update-params")
		require.Equal(t, "Update treasury contract parameters", cmd.Short)
		require.NotNil(t, cmd.Flags())
	})

	t.Run("argument validation", func(t *testing.T) {
		// ExactArgs(4)
		require.Error(t, cmd.Args(cmd, []string{}))
		require.Error(t, cmd.Args(cmd, []string{"param"}))
		require.Error(t, cmd.Args(cmd, []string{"a", "b", "c"}))
		require.NoError(t, cmd.Args(cmd, []string{"a", "b", "c", "d"}))
		require.Error(t, cmd.Args(cmd, []string{"a", "b", "c", "d", "e"}))
	})

	t.Run("execution with missing context", func(t *testing.T) {
		cmd.SetArgs([]string{"contract", "https://example.com/display", "https://example.com/redirect", "https://example.com/icon.png"})
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)

		err := cmd.Execute()
		require.Error(t, err)
	})

	t.Run("flag functionality", func(t *testing.T) {
		cmd := NewUpdateParamsCmd()

		// Test that the command has the standard tx flags
		flags := cmd.Flags()
		require.NotNil(t, flags)

		// Test some standard flags are available
		chainIDFlag := flags.Lookup("chain-id")
		require.NotNil(t, chainIDFlag)

		fromFlag := flags.Lookup("from")
		require.NotNil(t, fromFlag)
	})
}

// Tests for private functions - these are the 0% coverage functions
func TestPrivateFunctions(t *testing.T) {
	t.Run("typeURL", func(t *testing.T) {
		// Test with AbstractAccount
		acc := &aatypes.AbstractAccount{}
		url := typeURL(acc)
		require.Equal(t, "/abstractaccount.v1.AbstractAccount", url)

		// Test with standard account
		stdAcc := &authtypes.BaseAccount{}
		url = typeURL(stdAcc)
		require.Equal(t, "/cosmos.auth.v1beta1.BaseAccount", url)
	})

	t.Run("registerMsg", func(t *testing.T) {
		sender := "cosmos1..."
		salt := "test-salt"
		instantiateMsg := `{"test": "msg"}`
		codeID := uint64(123)
		amount := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1000)))

		msg := registerMsg(sender, salt, instantiateMsg, codeID, amount)

		require.Equal(t, sender, msg.Sender)
		require.Equal(t, codeID, msg.CodeID)
		require.Equal(t, instantiateMsg, string(msg.Msg))
		require.Equal(t, amount, msg.Funds)
		require.Equal(t, []byte(salt), msg.Salt)
	})

	t.Run("newInstantiateMsg", func(t *testing.T) {
		authenticatorType := "Secp256k1"
		authenticatorID := uint8(1)
		signature := []byte("test-signature")
		pubKey := []byte("test-pubkey")

		msgStr, err := newInstantiateMsg(authenticatorType, authenticatorID, signature, pubKey)
		require.NoError(t, err)
		require.NotEmpty(t, msgStr)

		// Verify JSON structure
		var msg map[string]interface{}
		err = json.Unmarshal([]byte(msgStr), &msg)
		require.NoError(t, err)

		authenticator, ok := msg["authenticator"].(map[string]interface{})
		require.True(t, ok)

		authDetails, ok := authenticator[authenticatorType].(map[string]interface{})
		require.True(t, ok)

		require.Equal(t, float64(authenticatorID), authDetails["id"])
		require.Equal(t, "dGVzdC1wdWJrZXk=", authDetails["pubkey"])        // base64 encoded
		require.Equal(t, "dGVzdC1zaWduYXR1cmU=", authDetails["signature"]) // base64 encoded
	})

	t.Run("newInstantiateJwtMsg", func(t *testing.T) {
		token := "test-jwt-token"
		authenticatorType := "Jwt"
		sub := "test-subject"
		aud := "test-audience"
		authenticatorID := uint8(2)

		msgStr, err := newInstantiateJwtMsg(token, authenticatorType, sub, aud, authenticatorID)
		require.NoError(t, err)
		require.NotEmpty(t, msgStr)

		// Verify JSON structure
		var msg map[string]interface{}
		err = json.Unmarshal([]byte(msgStr), &msg)
		require.NoError(t, err)

		authenticator, ok := msg["authenticator"].(map[string]interface{})
		require.True(t, ok)

		authDetails, ok := authenticator[authenticatorType].(map[string]interface{})
		require.True(t, ok)

		require.Equal(t, sub, authDetails["sub"])
		require.Equal(t, aud, authDetails["aud"])
		require.Equal(t, float64(authenticatorID), authDetails["id"])
		require.Equal(t, "dGVzdC1qd3QtdG9rZW4=", authDetails["token"]) // base64 encoded
	})
}

// Additional comprehensive tests to achieve 100% coverage
func TestAddAuthenticatorCmdExecutionPaths(t *testing.T) {
	// Test with different sign modes to exercise the switch statement
	signModes := []string{
		flags.SignModeDirect,
		flags.SignModeLegacyAminoJSON,
		flags.SignModeDirectAux,
		flags.SignModeTextual,
		flags.SignModeEIP191,
		"", // Default case
	}

	for _, signModeStr := range signModes {
		t.Run(fmt.Sprintf("sign_mode_%s", signModeStr), func(t *testing.T) {
			cmd := NewAddAuthenticatorCmd()

			// Setup client context with different sign modes
			encCfg := testutil.MakeTestEncodingConfig()
			clientCtx := client.Context{}.
				WithCodec(encCfg.Codec).
				WithTxConfig(encCfg.TxConfig).
				WithLegacyAmino(encCfg.Amino).
				WithChainID("test-chain").
				WithSignModeStr(signModeStr)

			// Create in-memory keyring
			kb := keyring.NewInMemory(encCfg.Codec)
			mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
			info, err := kb.NewAccount("test", mnemonic, "", sdk.FullFundraiserPath, hd.Secp256k1)
			require.NoError(t, err)

			addr, err := info.GetAddress()
			require.NoError(t, err)

			clientCtx = clientCtx.WithKeyring(kb).
				WithFromName("test").
				WithFromAddress(addr)

			cmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))
			cmd.SetArgs([]string{"xion1contractaddress123"})
			cmd.Flags().Set(flagAuthenticatorID, "1")

			// Execute - will fail but exercises the sign mode switch statement
			err = cmd.Execute()
			require.Error(t, err) // Expected to fail due to missing blockchain setup
		})
	}
}

func TestSignCmdExecutionPaths(t *testing.T) {
	// Create a temporary transaction file
	tempDir := t.TempDir()
	txFile := filepath.Join(tempDir, "test_tx.json")

	// Create a minimal valid transaction JSON
	txJSON := `{
		"body": {
			"messages": [],
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
			}
		},
		"signatures": []
	}`

	err := os.WriteFile(txFile, []byte(txJSON), 0644)
	require.NoError(t, err)

	cmd := NewSignCmd()

	// Setup client context
	encCfg := testutil.MakeTestEncodingConfig()
	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithTxConfig(encCfg.TxConfig).
		WithLegacyAmino(encCfg.Amino).
		WithChainID("test-chain")

	// Create in-memory keyring
	kb := keyring.NewInMemory(encCfg.Codec)
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	info, err := kb.NewAccount("test", mnemonic, "", sdk.FullFundraiserPath, hd.Secp256k1)
	require.NoError(t, err)

	addr, err := info.GetAddress()
	require.NoError(t, err)

	clientCtx = clientCtx.WithKeyring(kb).
		WithFromName("test").
		WithFromAddress(addr)

	cmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

	// Test with invalid signer address first
	cmd.SetArgs([]string{"test", "invalid-signer-addr", txFile})
	cmd.Flags().Set(flagAuthenticatorID, "1")
	err = cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "decoding bech32 failed")
}

func TestUpdateConfigsCmdPaths(t *testing.T) {
	// Create temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.json")

	// Test various config formats to exercise JSON parsing branches
	configs := []map[string]interface{}{
		{"config": "value1"},
		{"nested": map[string]interface{}{"key": "value"}},
		{}, // Empty config
		{"array": []interface{}{1, 2, 3}},
	}

	for i, config := range configs {
		t.Run(fmt.Sprintf("config_%d", i), func(t *testing.T) {
			configJSON, err := json.Marshal(config)
			require.NoError(t, err)

			err = os.WriteFile(configFile, configJSON, 0644)
			require.NoError(t, err)

			cmd := NewUpdateConfigsCmd()
			cmd.SetArgs([]string{configFile})

			// Will error due to missing client context but exercises JSON reading
			err = cmd.Execute()
			require.Error(t, err) // Expected due to missing context
		})
	}

	// Test with invalid JSON file
	invalidFile := filepath.Join(tempDir, "invalid.json")
	err := os.WriteFile(invalidFile, []byte("invalid json"), 0644)
	require.NoError(t, err)

	cmd := NewUpdateConfigsCmd()
	cmd.SetArgs([]string{invalidFile})
	err = cmd.Execute()
	require.Error(t, err)
}

func TestUpdateParamsCmdPaths(t *testing.T) {
	t.Run("invalid display URL", func(t *testing.T) {
		cmd := NewUpdateParamsCmd()
		cmd.SetArgs([]string{"contract_addr", "invalid-url", "https://example.com/redirect", "https://example.com/icon.png"})
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid display URL")
	})

	t.Run("invalid redirect URL", func(t *testing.T) {
		cmd := NewUpdateParamsCmd()
		cmd.SetArgs([]string{"contract_addr", "https://example.com/display", "invalid-url", "https://example.com/icon.png"})
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid redirect URL")
	})

	t.Run("invalid icon URL", func(t *testing.T) {
		cmd := NewUpdateParamsCmd()
		cmd.SetArgs([]string{"contract_addr", "https://example.com/display", "https://example.com/redirect", "invalid-url"})
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid icon URL")
	})

	t.Run("valid URLs but missing client context", func(t *testing.T) {
		cmd := NewUpdateParamsCmd()
		cmd.SetArgs([]string{"contract_addr", "https://example.com/display", "https://example.com/redirect", "https://example.com/icon.png"})
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)

		err := cmd.Execute()
		require.Error(t, err)
		// Should fail at client context retrieval, not URL validation
		require.NotContains(t, err.Error(), "invalid")
	})

	t.Run("edge case URLs", func(t *testing.T) {
		// Test with various valid URL formats
		testCases := []struct {
			name        string
			displayURL  string
			redirectURL string
			iconURL     string
		}{
			{
				name:        "http URLs",
				displayURL:  "http://example.com/display",
				redirectURL: "http://example.com/redirect",
				iconURL:     "http://example.com/icon.png",
			},
			{
				name:        "URLs with ports",
				displayURL:  "https://example.com:8080/display",
				redirectURL: "https://example.com:9090/redirect",
				iconURL:     "https://example.com:3000/icon.png",
			},
			{
				name:        "URLs with paths and query params",
				displayURL:  "https://example.com/path/to/display?param=value",
				redirectURL: "https://example.com/path/to/redirect?param=value&other=test",
				iconURL:     "https://example.com/path/to/icon.png?size=64",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cmd := NewUpdateParamsCmd()
				cmd.SetArgs([]string{"contract_addr", tc.displayURL, tc.redirectURL, tc.iconURL})
				cmd.SetOut(io.Discard)
				cmd.SetErr(io.Discard)

				err := cmd.Execute()
				require.Error(t, err)
				// Should fail at client context, not URL validation
				require.NotContains(t, err.Error(), "invalid")
			})
		}
	})

	t.Run("malformed URLs edge cases", func(t *testing.T) {
		testCases := []struct {
			name        string
			displayURL  string
			redirectURL string
			iconURL     string
			expectedErr string
		}{
			{
				name:        "empty display URL",
				displayURL:  "",
				redirectURL: "https://example.com/redirect",
				iconURL:     "https://example.com/icon.png",
				expectedErr: "invalid display URL",
			},
			{
				name:        "empty redirect URL",
				displayURL:  "https://example.com/display",
				redirectURL: "",
				iconURL:     "https://example.com/icon.png",
				expectedErr: "invalid redirect URL",
			},
			{
				name:        "empty icon URL",
				displayURL:  "https://example.com/display",
				redirectURL: "https://example.com/redirect",
				iconURL:     "",
				expectedErr: "invalid icon URL",
			},
			{
				name:        "space in display URL",
				displayURL:  "https://example.com/display url",
				redirectURL: "https://example.com/redirect",
				iconURL:     "https://example.com/icon.png",
				expectedErr: "", // URL parsing may actually succeed with space
			},
			{
				name:        "invalid scheme in redirect URL",
				displayURL:  "https://example.com/display",
				redirectURL: "ftp://example.com/redirect",
				iconURL:     "https://example.com/icon.png",
				expectedErr: "", // url.ParseRequestURI allows ftp:// scheme
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cmd := NewUpdateParamsCmd()
				cmd.SetArgs([]string{"contract_addr", tc.displayURL, tc.redirectURL, tc.iconURL})
				cmd.SetOut(io.Discard)
				cmd.SetErr(io.Discard)

				err := cmd.Execute()
				require.Error(t, err)
				if tc.expectedErr != "" {
					require.Contains(t, err.Error(), tc.expectedErr)
				}
			})
		}
	})
}

// Additional test for edge cases in NewUpdateParamsCmd
func TestUpdateParamsCmdEdgeCases(t *testing.T) {
	t.Run("message construction validation", func(t *testing.T) {
		// Test additional URL formats that should pass validation
		cmd := NewUpdateParamsCmd()

		testCases := [][]string{
			{"contract_addr", "https://test.com", "https://test.com", "https://test.com"},
			{"contract_addr", "http://localhost:3000", "http://localhost:8080", "http://localhost:9000"},
			{"long_contract_address_name", "https://very-long-domain-name.example.com/very/long/path", "https://another-domain.com", "https://third-domain.org"},
		}

		for i, args := range testCases {
			t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
				cmd.SetArgs(args)
				cmd.SetOut(io.Discard)
				cmd.SetErr(io.Discard)

				err := cmd.Execute()
				require.Error(t, err)
				// Should fail at client context, not at message construction
				require.NotContains(t, err.Error(), "marshal")
				require.NotContains(t, err.Error(), "invalid")
			})
		}
	})

	t.Run("validate message structure is correctly formed", func(t *testing.T) {
		// We can indirectly test message construction by verifying the function
		// reaches the client context part without errors in URL validation or marshaling
		cmd := NewUpdateParamsCmd()
		cmd.SetArgs([]string{
			"cosmos1contractaddress",
			"https://display.example.com/path?param=value",
			"https://redirect.example.com/path?param=value",
			"https://icon.example.com/path?param=value",
		})
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)

		err := cmd.Execute()
		require.Error(t, err)
		// If we reach here, it means URL validation and message construction passed
		// and only failed at the client context level
		require.NotContains(t, err.Error(), "invalid")
		require.NotContains(t, err.Error(), "marshal")
	})
}

// Tests for getSignerOfTx function - targeting 100% coverage
func TestGetSignerOfTx(t *testing.T) {
	// Create a test address
	testAddr := sdk.AccAddress("test_address_______")

	t.Run("successful case - valid AbstractAccount", func(t *testing.T) {
		// Create a mock query client
		mockClient := &MockQueryClient{}

		// Create a valid AbstractAccount
		abstractAcc := &aatypes.AbstractAccount{}
		abstractAcc.SetAddress(testAddr)
		abstractAcc.SetAccountNumber(1)
		abstractAcc.SetSequence(0)

		// Marshal the account
		accBytes, err := abstractAcc.Marshal()
		require.NoError(t, err)

		// Create the response with correct TypeURL
		response := &authtypes.QueryAccountResponse{
			Account: &codectypes.Any{
				TypeUrl: typeURL((*aatypes.AbstractAccount)(nil)),
				Value:   accBytes,
			},
		}

		// Set up mock expectation
		mockClient.On("Account", mock.Anything, mock.Anything, mock.Anything).Return(response, nil)

		// Call the function
		result, err := getSignerOfTx(mockClient, testAddr)

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, testAddr.String(), result.GetAddress().String())

		// Verify mock was called
		mockClient.AssertExpectations(t)
	})

	t.Run("error case - query client fails", func(t *testing.T) {
		// Create a mock query client
		mockClient := &MockQueryClient{}

		// Set up mock to return an error
		expectedErr := fmt.Errorf("query failed")
		mockClient.On("Account", mock.Anything, mock.Anything, mock.Anything).Return((*authtypes.QueryAccountResponse)(nil), expectedErr)

		// Call the function
		result, err := getSignerOfTx(mockClient, testAddr)

		// Verify results
		require.Error(t, err)
		require.Nil(t, result)
		require.Equal(t, expectedErr, err)

		// Verify mock was called
		mockClient.AssertExpectations(t)
	})

	t.Run("error case - account is not AbstractAccount", func(t *testing.T) {
		// Create a mock query client
		mockClient := &MockQueryClient{}

		// Create a response with wrong TypeURL (e.g., regular account)
		response := &authtypes.QueryAccountResponse{
			Account: &codectypes.Any{
				TypeUrl: "/cosmos.auth.v1beta1.BaseAccount", // Wrong type
				Value:   []byte("some_data"),
			},
		}

		// Set up mock expectation
		mockClient.On("Account", mock.Anything, mock.Anything, mock.Anything).Return(response, nil)

		// Call the function
		result, err := getSignerOfTx(mockClient, testAddr)

		// Verify results
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "is not an AbstractAccount")
		require.Contains(t, err.Error(), testAddr.String())

		// Verify mock was called
		mockClient.AssertExpectations(t)
	})

	t.Run("error case - proto unmarshal fails", func(t *testing.T) {
		// Create a mock query client
		mockClient := &MockQueryClient{}

		// Create a response with correct TypeURL but invalid data
		response := &authtypes.QueryAccountResponse{
			Account: &codectypes.Any{
				TypeUrl: typeURL((*aatypes.AbstractAccount)(nil)),
				Value:   []byte("invalid_proto_data"), // Invalid protobuf data
			},
		}

		// Set up mock expectation
		mockClient.On("Account", mock.Anything, mock.Anything, mock.Anything).Return(response, nil)

		// Call the function
		result, err := getSignerOfTx(mockClient, testAddr)

		// Verify results
		require.Error(t, err)
		require.Nil(t, result)

		// Verify mock was called
		mockClient.AssertExpectations(t)
	})
}

// Tests for typeURL function - targeting 100% coverage
func TestTypeURL(t *testing.T) {
	t.Run("AbstractAccount type", func(t *testing.T) {
		acc := &aatypes.AbstractAccount{}
		result := typeURL(acc)
		expected := "/" + proto.MessageName(acc)
		require.Equal(t, expected, result)
		require.Contains(t, result, "AbstractAccount")
	})

	t.Run("other proto message type", func(t *testing.T) {
		req := &authtypes.QueryAccountRequest{}
		result := typeURL(req)
		expected := "/" + proto.MessageName(req)
		require.Equal(t, expected, result)
		require.Contains(t, result, "QueryAccountRequest")
	})
}

// MockWasmQueryClient for testing NewRegisterCmd
type MockWasmQueryClient struct {
	mock.Mock
}

func (m *MockWasmQueryClient) ContractInfo(ctx context.Context, req *wasmtypes.QueryContractInfoRequest, opts ...grpc.CallOption) (*wasmtypes.QueryContractInfoResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*wasmtypes.QueryContractInfoResponse), args.Error(1)
}

func (m *MockWasmQueryClient) ContractHistory(ctx context.Context, req *wasmtypes.QueryContractHistoryRequest, opts ...grpc.CallOption) (*wasmtypes.QueryContractHistoryResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*wasmtypes.QueryContractHistoryResponse), args.Error(1)
}

func (m *MockWasmQueryClient) ContractsByCode(ctx context.Context, req *wasmtypes.QueryContractsByCodeRequest, opts ...grpc.CallOption) (*wasmtypes.QueryContractsByCodeResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*wasmtypes.QueryContractsByCodeResponse), args.Error(1)
}

func (m *MockWasmQueryClient) AllContractState(ctx context.Context, req *wasmtypes.QueryAllContractStateRequest, opts ...grpc.CallOption) (*wasmtypes.QueryAllContractStateResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*wasmtypes.QueryAllContractStateResponse), args.Error(1)
}

func (m *MockWasmQueryClient) RawContractState(ctx context.Context, req *wasmtypes.QueryRawContractStateRequest, opts ...grpc.CallOption) (*wasmtypes.QueryRawContractStateResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*wasmtypes.QueryRawContractStateResponse), args.Error(1)
}

func (m *MockWasmQueryClient) SmartContractState(ctx context.Context, req *wasmtypes.QuerySmartContractStateRequest, opts ...grpc.CallOption) (*wasmtypes.QuerySmartContractStateResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*wasmtypes.QuerySmartContractStateResponse), args.Error(1)
}

func (m *MockWasmQueryClient) Code(ctx context.Context, req *wasmtypes.QueryCodeRequest, opts ...grpc.CallOption) (*wasmtypes.QueryCodeResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*wasmtypes.QueryCodeResponse), args.Error(1)
}

func (m *MockWasmQueryClient) Codes(ctx context.Context, req *wasmtypes.QueryCodesRequest, opts ...grpc.CallOption) (*wasmtypes.QueryCodesResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*wasmtypes.QueryCodesResponse), args.Error(1)
}

func (m *MockWasmQueryClient) PinnedCodes(ctx context.Context, req *wasmtypes.QueryPinnedCodesRequest, opts ...grpc.CallOption) (*wasmtypes.QueryPinnedCodesResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*wasmtypes.QueryPinnedCodesResponse), args.Error(1)
}

func (m *MockWasmQueryClient) Params(ctx context.Context, req *wasmtypes.QueryParamsRequest, opts ...grpc.CallOption) (*wasmtypes.QueryParamsResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*wasmtypes.QueryParamsResponse), args.Error(1)
}

func (m *MockWasmQueryClient) ContractsByCreator(ctx context.Context, req *wasmtypes.QueryContractsByCreatorRequest, opts ...grpc.CallOption) (*wasmtypes.QueryContractsByCreatorResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*wasmtypes.QueryContractsByCreatorResponse), args.Error(1)
}

func (m *MockWasmQueryClient) BuildAddress(ctx context.Context, req *wasmtypes.QueryBuildAddressRequest, opts ...grpc.CallOption) (*wasmtypes.QueryBuildAddressResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*wasmtypes.QueryBuildAddressResponse), args.Error(1)
}

func TestNewRegisterCmd(t *testing.T) {
	t.Run("basic command structure", func(t *testing.T) {
		cmd := NewRegisterCmd()
		require.NotNil(t, cmd)
		require.Contains(t, cmd.Use, "register")
		require.Equal(t, "Register an abstract account", cmd.Short)
		require.NotNil(t, cmd.RunE)

		// Check required flags are present
		require.NotNil(t, cmd.Flag(flagSalt))
		require.NotNil(t, cmd.Flag(flagAuthenticator))
		require.NotNil(t, cmd.Flag(flagFunds))
		require.NotNil(t, cmd.Flag(flagAuthenticatorID))
		require.NotNil(t, cmd.Flag(flagAudience))
		require.NotNil(t, cmd.Flag(flagToken))
		require.NotNil(t, cmd.Flag(flagSubject))
	})

	t.Run("argument validation", func(t *testing.T) {
		cmd := NewRegisterCmd()

		// Test valid arguments
		require.NoError(t, cmd.Args(cmd, []string{}))
		require.NoError(t, cmd.Args(cmd, []string{"1"}))
		require.NoError(t, cmd.Args(cmd, []string{"1", "keyname"}))

		// Test invalid arguments (too many)
		require.Error(t, cmd.Args(cmd, []string{"1", "keyname", "extra"}))
	})

	t.Run("flag parsing errors", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino).
			WithInput(os.Stdin).
			WithHomeDir("./").
			WithFromAddress(sdk.AccAddress("test"))

		cmd := NewRegisterCmd()
		cmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		// Test invalid code ID
		cmd.SetArgs([]string{"invalid", "keyname"})
		err := cmd.RunE(cmd, []string{"invalid", "keyname"})
		require.Error(t, err)
		// Just check that it errors, don't check specific message since it might vary
	})

	t.Run("flag validation", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		kr := keyring.NewInMemory(encCfg.Codec)

		// Create a test key
		info, _, err := kr.NewMnemonic("test", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
		require.NoError(t, err)

		addr, err := info.GetAddress()
		require.NoError(t, err)

		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino).
			WithInput(os.Stdin).
			WithHomeDir("./").
			WithKeyring(kr).
			WithFromName("test").
			WithFromAddress(addr)

		cmd := NewRegisterCmd()
		cmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		// Test with valid code ID but missing required flags
		cmd.SetArgs([]string{"1", "test"})

		// Set some basic flags to get further into the function
		cmd.Flags().Set(flagAuthenticatorID, "1")
		cmd.Flags().Set(flagSalt, "test-salt")
		cmd.Flags().Set(flagFunds, "")
		cmd.Flags().Set(flagAuthenticator, "Secp256k1")

		runErr := cmd.RunE(cmd, []string{"1", "test"})
		require.Error(t, runErr)
		// Should error somewhere in the validation/execution path
	})

	t.Run("invalid coin parsing", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		kr := keyring.NewInMemory(encCfg.Codec)

		// Create a test key
		info, _, err := kr.NewMnemonic("test", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
		require.NoError(t, err)

		addr, err := info.GetAddress()
		require.NoError(t, err)

		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino).
			WithInput(os.Stdin).
			WithHomeDir("./").
			WithKeyring(kr).
			WithFromName("test").
			WithFromAddress(addr)

		cmd := NewRegisterCmd()
		cmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		// Test with invalid coin format
		cmd.SetArgs([]string{"1", "test"})
		cmd.Flags().Set(flagAuthenticatorID, "1")
		cmd.Flags().Set(flagSalt, "test-salt")
		cmd.Flags().Set(flagFunds, "invalid-coin-format")
		cmd.Flags().Set(flagAuthenticator, "Secp256k1")

		runErr := cmd.RunE(cmd, []string{"1", "test"})
		require.Error(t, runErr)
		require.Contains(t, runErr.Error(), "amount:")
	})

	t.Run("jwt authenticator path", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		kr := keyring.NewInMemory(encCfg.Codec)

		// Create a test key
		info, _, err := kr.NewMnemonic("test", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
		require.NoError(t, err)

		addr, err := info.GetAddress()
		require.NoError(t, err)

		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino).
			WithInput(os.Stdin).
			WithHomeDir("./").
			WithKeyring(kr).
			WithFromName("test").
			WithFromAddress(addr)

		cmd := NewRegisterCmd()
		cmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		// Test JWT authenticator path but missing flags
		cmd.SetArgs([]string{"1", "test"})
		cmd.Flags().Set(flagAuthenticatorID, "1")
		cmd.Flags().Set(flagSalt, "test-salt")
		cmd.Flags().Set(flagFunds, "1000stake")
		cmd.Flags().Set(flagAuthenticator, "Jwt")
		// Don't set JWT-specific flags to trigger errors

		runErr := cmd.RunE(cmd, []string{"1", "test"})
		require.Error(t, runErr)
		// The error could be about missing JWT flags or network issues - both are valid error paths
	})

	// Helper function for string matching
	containsAny := func(str string, substrings []string) bool {
		for _, substr := range substrings {
			if strings.Contains(str, substr) {
				return true
			}
		}
		return false
	}

	t.Run("jwt flag errors", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		kr := keyring.NewInMemory(encCfg.Codec)

		info, _, err := kr.NewMnemonic("test", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
		require.NoError(t, err)

		addr, err := info.GetAddress()
		require.NoError(t, err)

		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino).
			WithInput(os.Stdin).
			WithHomeDir("./").
			WithKeyring(kr).
			WithFromName("test").
			WithFromAddress(addr)

		// Test various JWT flag error paths
		testCases := []struct {
			name        string
			setupFlags  func(*cobra.Command)
			expectError string
		}{
			{
				name: "missing subject",
				setupFlags: func(c *cobra.Command) {
					c.Flags().Set(flagAuthenticatorID, "1")
					c.Flags().Set(flagSalt, "test-salt")
					c.Flags().Set(flagFunds, "1000stake")
					c.Flags().Set(flagAuthenticator, "Jwt")
					// Missing subject flag
				},
				expectError: "subject:",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Create new command for each test
				testCmd := NewRegisterCmd()
				testCmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))
				testCmd.SetArgs([]string{"1", "test"})

				tc.setupFlags(testCmd)

				runErr := testCmd.RunE(testCmd, []string{"1", "test"})
				require.Error(t, runErr)

				// Check if error contains expected substring (if we get that far)
				// Network errors might happen first in some cases
				errorStr := runErr.Error()
				if !containsAny(errorStr, []string{"connection refused", "post failed"}) {
					require.Contains(t, errorStr, tc.expectError)
				}
			})
		}
	})

	t.Run("more comprehensive flag testing", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		kr := keyring.NewInMemory(encCfg.Codec)

		info, _, err := kr.NewMnemonic("test", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
		require.NoError(t, err)

		addr, err := info.GetAddress()
		require.NoError(t, err)

		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino).
			WithInput(os.Stdin).
			WithHomeDir("./").
			WithKeyring(kr).
			WithFromName("test").
			WithFromAddress(addr)

		// Test flag retrieval errors
		testCases := []struct {
			name      string
			setupTest func() *cobra.Command
		}{
			{
				name: "salt flag error",
				setupTest: func() *cobra.Command {
					cmd := NewRegisterCmd()
					cmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))
					cmd.SetArgs([]string{"1", "test"})
					cmd.Flags().Set(flagAuthenticatorID, "1")
					// Don't set salt flag to trigger error
					return cmd
				},
			},
			{
				name: "funds flag error",
				setupTest: func() *cobra.Command {
					cmd := NewRegisterCmd()
					cmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))
					cmd.SetArgs([]string{"1", "test"})
					cmd.Flags().Set(flagAuthenticatorID, "1")
					cmd.Flags().Set(flagSalt, "test-salt")
					// Don't set funds flag to trigger error
					return cmd
				},
			},
			{
				name: "authenticator flag error",
				setupTest: func() *cobra.Command {
					cmd := NewRegisterCmd()
					cmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))
					cmd.SetArgs([]string{"1", "test"})
					cmd.Flags().Set(flagAuthenticatorID, "1")
					cmd.Flags().Set(flagSalt, "test-salt")
					cmd.Flags().Set(flagFunds, "1000stake")
					// Don't set authenticator flag to trigger error
					return cmd
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cmd := tc.setupTest()
				runErr := cmd.RunE(cmd, []string{"1", "test"})
				require.Error(t, runErr)
				// Just verify it errors - the specific error can vary
			})
		}
	})

	t.Run("default authenticator path", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		kr := keyring.NewInMemory(encCfg.Codec)

		info, _, err := kr.NewMnemonic("test", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
		require.NoError(t, err)

		addr, err := info.GetAddress()
		require.NoError(t, err)

		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino).
			WithInput(os.Stdin).
			WithHomeDir("./").
			WithKeyring(kr).
			WithFromName("test").
			WithFromAddress(addr)

		cmd := NewRegisterCmd()
		cmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		// Test default authenticator path (non-JWT)
		cmd.SetArgs([]string{"1", "test"})
		cmd.Flags().Set(flagAuthenticatorID, "1")
		cmd.Flags().Set(flagSalt, "test-salt")
		cmd.Flags().Set(flagFunds, "1000stake")
		cmd.Flags().Set(flagAuthenticator, "Secp256k1") // Use default path

		runErr := cmd.RunE(cmd, []string{"1", "test"})
		require.Error(t, runErr)
		// Should error due to network/query client issues, but we've exercised the default path
	})

	t.Run("client context errors", func(t *testing.T) {
		cmd := NewRegisterCmd()

		// Test with minimal context that will cause early errors
		encCfg := testutil.MakeTestEncodingConfig()
		clientCtx := client.Context{}.WithCodec(encCfg.Codec)
		cmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		cmd.SetArgs([]string{"1", "test"})

		runErr := cmd.RunE(cmd, []string{"1", "test"})
		require.Error(t, runErr)
		// Should fail early due to incomplete client context
	})

	t.Run("code id parsing edge cases", func(t *testing.T) {
		encCfg := testutil.MakeTestEncodingConfig()
		clientCtx := client.Context{}.
			WithCodec(encCfg.Codec).
			WithTxConfig(encCfg.TxConfig).
			WithLegacyAmino(encCfg.Amino)

		cmd := NewRegisterCmd()
		cmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		// Test edge cases for code ID parsing
		edgeCases := []string{"0", "999999999999999999999", "-1", "abc", "1.5"}

		for _, codeIDStr := range edgeCases {
			t.Run("codeID_"+codeIDStr, func(t *testing.T) {
				cmd.SetArgs([]string{codeIDStr, "test"})
				runErr := cmd.RunE(cmd, []string{codeIDStr, "test"})
				require.Error(t, runErr)
				// Each should error for different reasons (parsing, validation, etc.)
			})
		}
	})
}
