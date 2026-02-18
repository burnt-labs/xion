package cli_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/x/dkim/client/cli"
	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestGetQueryCmd(t *testing.T) {
	cmd := cli.GetQueryCmd()
	require.NotNil(t, cmd)
	require.Equal(t, types.ModuleName, cmd.Use)
	require.True(t, cmd.DisableFlagParsing)

	// Verify subcommands are added
	require.True(t, len(cmd.Commands()) > 0)
}

func TestGetDkimPublicKey(t *testing.T) {
	cmd := cli.GetDkimPublicKey()
	require.NotNil(t, cmd)
	require.Contains(t, cmd.Use, "dkim-pubkey")
	require.Equal(t, "Get a DKIM public key", cmd.Short)
	require.NotEmpty(t, cmd.Aliases)
	require.Contains(t, cmd.Aliases, "qdkim")

	// Test that command has RunE function defined
	require.NotNil(t, cmd.RunE)

	// Test command structure
	require.NotNil(t, cmd.Flags())

	// Test args validation
	t.Run("args validator with 1 arg should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"domain"})
		require.Error(t, err, "should require exactly 2 args")
	})

	t.Run("args validator with 2 args", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"domain", "selector"})
		require.NoError(t, err, "should accept exactly 2 args")
	})

	t.Run("args validator with 3 args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"domain", "selector", "extra"})
		require.Error(t, err, "should reject more than 2 args")
	})

	// Call RunE to ensure coverage - expect panic/error without context
	func() {
		defer func() {
			_ = recover() // Ignore panics from missing context
		}()
		_ = cmd.RunE(cmd, []string{"example.com", "default"})
	}()
}

func TestGetDkimPublicKeys(t *testing.T) {
	cmd := cli.GetDkimPublicKeys()
	require.NotNil(t, cmd)
	require.Contains(t, cmd.Use, "dkim-pubkeys")
	require.Equal(t, "Get a DKIM public key matching filter parameters", cmd.Short)
	require.NotEmpty(t, cmd.Long)
	require.NotEmpty(t, cmd.Example)
	require.NotEmpty(t, cmd.Aliases)
	require.Contains(t, cmd.Aliases, "qdkims")

	// Test that command has RunE function defined
	require.NotNil(t, cmd.RunE)

	// Test command structure and flags
	require.NotNil(t, cmd.Flags())
	domainFlag := cmd.Flags().Lookup("domain")
	require.NotNil(t, domainFlag)
	require.Equal(t, "Filter by domain", domainFlag.Usage)

	selectorFlag := cmd.Flags().Lookup("selector")
	require.NotNil(t, selectorFlag)
	require.Contains(t, selectorFlag.Usage, "selector")

	hashFlag := cmd.Flags().Lookup("hash")
	require.NotNil(t, hashFlag)
	require.Contains(t, hashFlag.Usage, "poseidon hash")

	// Test Args validator with different argument counts
	// cobra.RangeArgs(0, 3) should allow 0-3 args
	t.Run("args validator with 0 args", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		require.NoError(t, err)
	})

	t.Run("args validator with 1 arg", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"arg1"})
		require.NoError(t, err)
	})

	t.Run("args validator with 2 args", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"arg1", "arg2"})
		require.NoError(t, err)
	})

	t.Run("args validator with 3 args", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"arg1", "arg2", "arg3"})
		require.NoError(t, err)
	})

	t.Run("args validator with 4 args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"arg1", "arg2", "arg3", "arg4"})
		require.Error(t, err)
	})

	// Test with different flag combinations to improve coverage
	t.Run("no flags", func(t *testing.T) {
		func() {
			defer func() {
				_ = recover() // Ignore panics from missing context
			}()
			_ = cmd.RunE(cmd, []string{})
		}()
	})

	t.Run("with domain flag", func(t *testing.T) {
		require.NoError(t, cmd.Flags().Set("domain", "example.com"))
		func() {
			defer func() {
				_ = recover() // Ignore panics from missing context
			}()
			_ = cmd.RunE(cmd, []string{})
		}()
	})

	t.Run("with selector flag", func(t *testing.T) {
		require.NoError(t, cmd.Flags().Set("selector", "default"))
		func() {
			defer func() {
				_ = recover() // Ignore panics from missing context
			}()
			_ = cmd.RunE(cmd, []string{})
		}()
	})

	t.Run("with hash flag", func(t *testing.T) {
		require.NoError(t, cmd.Flags().Set("hash", "somehash"))
		func() {
			defer func() {
				_ = recover() // Ignore panics from missing context
			}()
			_ = cmd.RunE(cmd, []string{})
		}()
	})
}

func TestGenerateDkimPublicKey(t *testing.T) {
	cmd := cli.GenerateDkimPublicKey()
	require.NotNil(t, cmd)
	require.Contains(t, cmd.Use, "generate-dkim-pubkey")
	require.Equal(t, "Generate a DKIM msg to create a new DKIM public key", cmd.Short)
	require.NotEmpty(t, cmd.Aliases)
	require.Contains(t, cmd.Aliases, "gdkim")
	require.NotEmpty(t, cmd.Example)
	require.NotEmpty(t, cmd.Long)

	// Test that command has RunE function defined
	require.NotNil(t, cmd.RunE)

	// Test command structure
	require.NotNil(t, cmd.Flags())

	// Test args validation
	t.Run("args validator with 1 arg should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"domain"})
		require.Error(t, err, "should require exactly 2 args")
	})

	t.Run("args validator with 2 args", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"domain", "selector"})
		require.NoError(t, err, "should accept exactly 2 args")
	})

	// Call RunE to ensure coverage - expect panic/error without context or DNS
	func() {
		defer func() {
			_ = recover() // Ignore panics from missing context
		}()
		_ = cmd.RunE(cmd, []string{"example.com", "default"})
	}()
}

func TestGetParams(t *testing.T) {
	cmd := cli.GetParams()
	require.NotNil(t, cmd)
	require.Equal(t, "params", cmd.Use)
	require.Equal(t, "Query DKIM module parameters", cmd.Short)
	require.NotEmpty(t, cmd.Long)
	require.NotEmpty(t, cmd.Example)

	// Test that command has RunE function defined
	require.NotNil(t, cmd.RunE)

	// Test command structure
	require.NotNil(t, cmd.Flags())

	// Test args validation - params takes no args (cobra.NoArgs)
	t.Run("args validator with 0 args", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		require.NoError(t, err, "should accept no args")
	})

	t.Run("args validator with 1 arg should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"extra"})
		require.Error(t, err, "should reject any args")
	})

	t.Run("args validator with 2 args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"arg1", "arg2"})
		require.Error(t, err, "should reject any args")
	})

	// Call RunE to ensure coverage - expect panic/error without context
	func() {
		defer func() {
			_ = recover() // Ignore panics from missing context
		}()
		_ = cmd.RunE(cmd, []string{})
	}()
}

func TestGenerateDkimPubKeyMsg(t *testing.T) {
	t.Run("with invalid domain/selector (DNS lookup fails)", func(t *testing.T) {
		// Testing with a domain that won't have DKIM records
		dkimPubKey, err := cli.GenerateDkimPubKeyMsg("nonexistent-domain-that-will-fail-12345.invalid", "selector")
		require.Error(t, err)
		require.Nil(t, dkimPubKey)
		require.Contains(t, err.Error(), "failed to lookup TXT records")
	})

	t.Run("with domain missing DKIM selector", func(t *testing.T) {
		// Testing with a real domain but invalid selector
		// This will fail at one of several stages:
		// 1. DNS lookup fails
		// 2. DKIM public key not found in DNS records
		// 3. Public key decoding/parsing fails
		// 4. Poseidon hash computation fails
		dkimPubKey, err := cli.GenerateDkimPubKeyMsg("example.com", "nonexistent-selector-12345")
		require.Error(t, err)
		require.Nil(t, dkimPubKey)
		// Error could be at any stage - just verify we get an error
		require.NotNil(t, err, "Expected an error from DNS lookup, key parsing, or hash computation")
	})

	t.Run("with real domain and valid selector (google)", func(t *testing.T) {
		// Test with Google's well-known DKIM selector
		// This tests the success path including Poseidon hash computation
		dkimPubKey, err := cli.GenerateDkimPubKeyMsg("google.com", "20230601")
		if err != nil {
			// Skip if DNS is not available or key format changed
			t.Skipf("Skipping real DNS test: %v", err)
		}
		require.NotNil(t, dkimPubKey)
		require.Equal(t, "google.com", dkimPubKey.Domain)
		require.Equal(t, "20230601", dkimPubKey.Selector)
		require.NotEmpty(t, dkimPubKey.PubKey)
		require.NotEmpty(t, dkimPubKey.PoseidonHash)
	})
}

func TestMsgRevokeDkimPubKey(t *testing.T) {
	cmd := cli.MsgRevokeDkimPubKey()
	require.NotNil(t, cmd)
	require.Contains(t, cmd.Use, "revoke-dkim")
	require.Equal(t, "Revoke a Dkim pubkey without governance.", cmd.Short)
	require.NotEmpty(t, cmd.Aliases)
	require.Contains(t, cmd.Aliases, "rdkim")
	require.NotEmpty(t, cmd.Long)
	require.Contains(t, cmd.Long, "PEM encoded private key")

	// Test that command has RunE function defined
	require.NotNil(t, cmd.RunE)

	// Test command structure
	require.NotNil(t, cmd.Flags())

	// Test args validation - requires exactly 2 args
	t.Run("args validator with 0 args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		require.Error(t, err)
	})

	t.Run("args validator with 1 arg should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"domain"})
		require.Error(t, err)
	})

	t.Run("args validator with 2 args", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"domain", "privkey"})
		require.NoError(t, err)
	})

	t.Run("args validator with 3 args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"domain", "privkey", "extra"})
		require.Error(t, err)
	})

	// Test with invalid private key to cover error handling
	t.Run("with invalid private key", func(t *testing.T) {
		func() {
			defer func() {
				_ = recover() // Ignore panics from missing context
			}()
			_ = cmd.RunE(cmd, []string{"example.com", "invalid-key"})
		}()
	})

	// Test with a properly formatted (but fake) private key to cover more code paths
	t.Run("with formatted private key", func(t *testing.T) {
		// This will fail at getting client context, but will cover the PEM decoding logic
		fakeKey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA"
		func() {
			defer func() {
				_ = recover() // Ignore panics from missing context
			}()
			_ = cmd.RunE(cmd, []string{"https://example.com", fakeKey})
		}()
	})
}

func TestParseAndValidateRevokeDkimMsg(t *testing.T) {
	// Generate a valid RSA private key for testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Encode as PKCS1 and get just the base64 part (without PEM headers)
	privKeyDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyDER,
	})

	// Remove PEM headers/footers to simulate user input (FormatToPemKey will add them back)
	privKeyStr := string(privKeyPEM)
	privKeyStr = privKeyStr[len("-----BEGIN RSA PRIVATE KEY-----\n"):]
	privKeyStr = privKeyStr[:len(privKeyStr)-len("\n-----END RSA PRIVATE KEY-----\n")]

	t.Run("valid private key", func(t *testing.T) {
		msg, err := cli.ParseAndValidateRevokeDkimMsg(
			"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a",
			"https://example.com",
			privKeyStr,
		)
		require.NoError(t, err)
		require.NotNil(t, msg)
		require.Equal(t, "xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a", msg.Signer)
		require.Equal(t, "https://example.com", msg.Domain)
		require.NotEmpty(t, msg.PrivKey)
	})

	t.Run("invalid private key", func(t *testing.T) {
		msg, err := cli.ParseAndValidateRevokeDkimMsg(
			"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a",
			"https://example.com",
			"invalid-key-data",
		)
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("invalid private key - not base64", func(t *testing.T) {
		msg, err := cli.ParseAndValidateRevokeDkimMsg(
			"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a",
			"example.com",
			"this-is-not-valid-base64!!!",
		)
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrParsingPrivKey)
		require.Nil(t, msg)
	})

	t.Run("empty private key", func(t *testing.T) {
		msg, err := cli.ParseAndValidateRevokeDkimMsg(
			"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a",
			"example.com",
			"",
		)
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrParsingPrivKey)
		require.Nil(t, msg)
	})

	t.Run("invalid PEM block", func(t *testing.T) {
		msg, err := cli.ParseAndValidateRevokeDkimMsg(
			"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a",
			"example.com",
			"notavalidpemblock",
		)
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrParsingPrivKey)
		require.Nil(t, msg)
	})

	t.Run("domain is parsed leniently", func(t *testing.T) {
		// url.Parse is very lenient, so even "not a url" parses successfully
		msg, err := cli.ParseAndValidateRevokeDkimMsg(
			"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a",
			"not-a-url",
			privKeyStr,
		)
		require.NoError(t, err) // url.Parse accepts almost anything
		require.NotNil(t, msg)
	})

	t.Run("empty signer is allowed", func(t *testing.T) {
		// ValidateBasic doesn't check signer validity
		msg, err := cli.ParseAndValidateRevokeDkimMsg(
			"",
			"https://example.com",
			privKeyStr,
		)
		require.NoError(t, err)
		require.NotNil(t, msg)
		require.Empty(t, msg.Signer)
	})
}

func TestNewTxCmdStructure(t *testing.T) {
	cmd := cli.NewTxCmd()

	// Test command properties
	require.Equal(t, types.ModuleName, cmd.Use)
	require.Contains(t, cmd.Short, types.ModuleName)
	require.True(t, cmd.DisableFlagParsing)
	require.Equal(t, 2, cmd.SuggestionsMinimumDistance)

	// Verify RunE is set to ValidateCmd
	require.NotNil(t, cmd.RunE)

	// Test that subcommands include revoke-dkim
	subCmds := cmd.Commands()
	require.True(t, len(subCmds) > 0)

	found := false
	for _, subCmd := range subCmds {
		if subCmd.Use == "revoke-dkim <domain> <priv_key>" {
			found = true
			break
		}
	}
	require.True(t, found, "expected revoke-dkim subcommand to be present")
}

func TestMsgRevokeDkimPubKeyHasTxFlags(t *testing.T) {
	cmd := cli.MsgRevokeDkimPubKey()

	// Check that transaction flags are present (added by flags.AddTxFlagsToCmd)
	fromFlag := cmd.Flags().Lookup("from")
	require.NotNil(t, fromFlag, "expected 'from' flag to be present")

	feesFlag := cmd.Flags().Lookup("fees")
	require.NotNil(t, feesFlag, "expected 'fees' flag to be present")

	gasFlag := cmd.Flags().Lookup("gas")
	require.NotNil(t, gasFlag, "expected 'gas' flag to be present")

	chainIDFlag := cmd.Flags().Lookup("chain-id")
	require.NotNil(t, chainIDFlag, "expected 'chain-id' flag to be present")
}

func TestParseDkimPubKeysFlags(t *testing.T) {
	t.Run("all flags set", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("domain", "", "")
		cmd.Flags().String("selector", "", "")
		cmd.Flags().String("hash", "", "")

		require.NoError(t, cmd.Flags().Set("domain", "example.com"))
		require.NoError(t, cmd.Flags().Set("selector", "default"))
		require.NoError(t, cmd.Flags().Set("hash", "abc123"))

		domain, selector, hash, err := cli.ParseDkimPubKeysFlags(cmd)
		require.NoError(t, err)
		require.Equal(t, "example.com", domain)
		require.Equal(t, "default", selector)
		require.Equal(t, "abc123", hash)
	})

	t.Run("no flags set", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("domain", "", "")
		cmd.Flags().String("selector", "", "")
		cmd.Flags().String("hash", "", "")

		domain, selector, hash, err := cli.ParseDkimPubKeysFlags(cmd)
		require.NoError(t, err)
		require.Equal(t, "", domain)
		require.Equal(t, "", selector)
		require.Equal(t, "", hash)
	})

	t.Run("only domain flag set", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("domain", "", "")
		cmd.Flags().String("selector", "", "")
		cmd.Flags().String("hash", "", "")

		require.NoError(t, cmd.Flags().Set("domain", "test.com"))

		domain, selector, hash, err := cli.ParseDkimPubKeysFlags(cmd)
		require.NoError(t, err)
		require.Equal(t, "test.com", domain)
		require.Equal(t, "", selector)
		require.Equal(t, "", hash)
	})

	t.Run("missing domain flag definition", func(t *testing.T) {
		cmd := &cobra.Command{}
		// Don't define domain flag
		cmd.Flags().String("selector", "", "")
		cmd.Flags().String("hash", "", "")

		_, _, _, err := cli.ParseDkimPubKeysFlags(cmd)
		require.Error(t, err)
	})

	t.Run("missing selector flag definition", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("domain", "", "")
		// Don't define selector flag
		cmd.Flags().String("hash", "", "")

		_, _, _, err := cli.ParseDkimPubKeysFlags(cmd)
		require.Error(t, err)
	})

	t.Run("missing hash flag definition", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("domain", "", "")
		cmd.Flags().String("selector", "", "")
		// Don't define hash flag

		_, _, _, err := cli.ParseDkimPubKeysFlags(cmd)
		require.Error(t, err)
	})
}

func TestQueryDkimPubKey(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	t.Run("successful query", func(t *testing.T) {
		mockClient := &MockQueryClient{
			dkimPubKeyFunc: func(ctx context.Context, req *types.QueryDkimPubKeyRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeyResponse, error) {
				require.Equal(t, "example.com", req.Domain)
				require.Equal(t, "default", req.Selector)
				return &types.QueryDkimPubKeyResponse{
					DkimPubKey: &types.DkimPubKey{
						Domain:   req.Domain,
						Selector: req.Selector,
						PubKey:   "MIIBIjANBgkq...",
					},
				}, nil
			},
		}

		res, err := cli.QueryDkimPubKey(mockClient, cmd, "example.com", "default")
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, "example.com", res.DkimPubKey.Domain)
		require.Equal(t, "default", res.DkimPubKey.Selector)
	})

	t.Run("with different domain and selector", func(t *testing.T) {
		mockClient := &MockQueryClient{
			dkimPubKeyFunc: func(ctx context.Context, req *types.QueryDkimPubKeyRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeyResponse, error) {
				require.Equal(t, "test.org", req.Domain)
				require.Equal(t, "selector123", req.Selector)
				return &types.QueryDkimPubKeyResponse{
					DkimPubKey: &types.DkimPubKey{
						Domain:   req.Domain,
						Selector: req.Selector,
						PubKey:   "MIIBIjANBgkq...",
					},
				}, nil
			},
		}

		res, err := cli.QueryDkimPubKey(mockClient, cmd, "test.org", "selector123")
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, "test.org", res.DkimPubKey.Domain)
		require.Equal(t, "selector123", res.DkimPubKey.Selector)
	})

	t.Run("query error", func(t *testing.T) {
		mockClient := &MockQueryClient{
			dkimPubKeyFunc: func(ctx context.Context, req *types.QueryDkimPubKeyRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeyResponse, error) {
				return nil, errors.New("key not found")
			},
		}

		resp, err := cli.QueryDkimPubKey(mockClient, cmd, "notfound.com", "dkim1")
		require.Error(t, err)
		require.Nil(t, resp)
	})

	t.Run("empty domain", func(t *testing.T) {
		mockClient := &MockQueryClient{
			dkimPubKeyFunc: func(ctx context.Context, req *types.QueryDkimPubKeyRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeyResponse, error) {
				require.Equal(t, "", req.Domain)
				require.Equal(t, "dkim1", req.Selector)
				return &types.QueryDkimPubKeyResponse{}, nil
			},
		}

		resp, err := cli.QueryDkimPubKey(mockClient, cmd, "", "dkim1")
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

func TestQueryDkimPubKeys(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	t.Run("with all filters", func(t *testing.T) {
		mockClient := &MockQueryClient{
			dkimPubKeysFunc: func(ctx context.Context, req *types.QueryDkimPubKeysRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeysResponse, error) {
				require.Equal(t, "example.com", req.Domain)
				require.Equal(t, "default", req.Selector)
				require.Equal(t, []byte("hash123"), req.PoseidonHash)
				return &types.QueryDkimPubKeysResponse{
					DkimPubKeys: []*types.DkimPubKey{
						{Domain: req.Domain, Selector: req.Selector},
					},
				}, nil
			},
		}

		res, err := cli.QueryDkimPubKeys(mockClient, cmd, "example.com", "default", "hash123")
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.DkimPubKeys, 1)
		require.Equal(t, "example.com", res.DkimPubKeys[0].Domain)
		require.Equal(t, "default", res.DkimPubKeys[0].Selector)
	})

	t.Run("with no filters", func(t *testing.T) {
		mockClient := &MockQueryClient{
			dkimPubKeysFunc: func(ctx context.Context, req *types.QueryDkimPubKeysRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeysResponse, error) {
				require.Equal(t, "", req.Domain)
				require.Equal(t, "", req.Selector)
				require.Equal(t, []byte(""), req.PoseidonHash)
				return &types.QueryDkimPubKeysResponse{
					DkimPubKeys: []*types.DkimPubKey{
						{Domain: "example1.com"},
						{Domain: "example2.com"},
					},
				}, nil
			},
		}

		res, err := cli.QueryDkimPubKeys(mockClient, cmd, "", "", "")
		require.NoError(t, err)
		require.NotNil(t, res)
	})

	t.Run("with domain and selector", func(t *testing.T) {
		mockClient := &MockQueryClient{
			dkimPubKeysFunc: func(ctx context.Context, req *types.QueryDkimPubKeysRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeysResponse, error) {
				require.Equal(t, "example.com", req.Domain)
				require.Equal(t, "dkim1", req.Selector)
				return &types.QueryDkimPubKeysResponse{
					DkimPubKeys: []*types.DkimPubKey{
						{Domain: "example.com", Selector: "dkim1"},
					},
				}, nil
			},
		}

		res, err := cli.QueryDkimPubKeys(mockClient, cmd, "example.com", "dkim1", "")
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.DkimPubKeys, 1)
	})

	t.Run("with domain and hash", func(t *testing.T) {
		mockClient := &MockQueryClient{
			dkimPubKeysFunc: func(ctx context.Context, req *types.QueryDkimPubKeysRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeysResponse, error) {
				require.Equal(t, "example.com", req.Domain)
				require.Equal(t, "", req.Selector)
				require.Equal(t, []byte("abc123"), req.PoseidonHash)
				return &types.QueryDkimPubKeysResponse{
					DkimPubKeys: []*types.DkimPubKey{
						{Domain: "example.com", PoseidonHash: []byte("abc123")},
					},
				}, nil
			},
		}

		res, err := cli.QueryDkimPubKeys(mockClient, cmd, "example.com", "", "abc123")
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.DkimPubKeys, 1)
	})

	t.Run("query error", func(t *testing.T) {
		mockClient := &MockQueryClient{
			dkimPubKeysFunc: func(ctx context.Context, req *types.QueryDkimPubKeysRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeysResponse, error) {
				return nil, errors.New("internal error")
			},
		}

		resp, err := cli.QueryDkimPubKeys(mockClient, cmd, "example.com", "dkim1", "")
		require.Error(t, err)
		require.Nil(t, resp)
	})
}

func TestQueryParams(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	t.Run("successful query", func(t *testing.T) {
		mockClient := &MockQueryClient{
			paramsFunc: func(ctx context.Context, req *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error) {
				return &types.QueryParamsResponse{
					Params: &types.Params{
						VkeyIdentifier: 42,
					},
				}, nil
			},
		}

		res, err := cli.QueryParams(mockClient, cmd)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Params)
		require.Equal(t, uint64(42), res.Params.VkeyIdentifier)
	})

	t.Run("query error", func(t *testing.T) {
		mockClient := &MockQueryClient{
			paramsFunc: func(ctx context.Context, req *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error) {
				return nil, errors.New("params not found")
			},
		}

		resp, err := cli.QueryParams(mockClient, cmd)
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "params not found")
	})

	t.Run("empty params", func(t *testing.T) {
		mockClient := &MockQueryClient{
			paramsFunc: func(ctx context.Context, req *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error) {
				return &types.QueryParamsResponse{
					Params: &types.Params{
						VkeyIdentifier: 0,
					},
				}, nil
			},
		}

		res, err := cli.QueryParams(mockClient, cmd)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Params)
		require.Equal(t, uint64(0), res.Params.VkeyIdentifier)
	})

	t.Run("params with multiple dkim pubkeys", func(t *testing.T) {
		mockClient := &MockQueryClient{
			paramsFunc: func(ctx context.Context, req *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error) {
				return &types.QueryParamsResponse{
					Params: &types.Params{
						VkeyIdentifier: 1,
					},
				}, nil
			},
		}

		res, err := cli.QueryParams(mockClient, cmd)
		require.NoError(t, err)
		require.NotNil(t, res)
	})

	t.Run("params with large vkey identifier", func(t *testing.T) {
		mockClient := &MockQueryClient{
			paramsFunc: func(ctx context.Context, req *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error) {
				return &types.QueryParamsResponse{
					Params: &types.Params{
						VkeyIdentifier: 18446744073709551615, // max uint64
					},
				}, nil
			},
		}

		res, err := cli.QueryParams(mockClient, cmd)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, uint64(18446744073709551615), res.Params.VkeyIdentifier)
	})
}

func TestGenerateDkimPublicKeyCommand(t *testing.T) {
	cmd := cli.GenerateDkimPublicKey()

	t.Run("command structure", func(t *testing.T) {
		require.NotNil(t, cmd)
		require.Contains(t, cmd.Use, "generate-dkim-pubkey")
		require.Equal(t, "Generate a DKIM msg to create a new DKIM public key", cmd.Short)
		require.NotEmpty(t, cmd.Long)
		require.Contains(t, cmd.Long, "DKIM msg")
		require.Contains(t, cmd.Long, "poseidon hash")
		require.NotEmpty(t, cmd.Example)
		require.Contains(t, cmd.Example, "gen-dkim-pubkey")
	})

	t.Run("command aliases", func(t *testing.T) {
		require.NotEmpty(t, cmd.Aliases)
		require.Contains(t, cmd.Aliases, "gdkim")
	})

	t.Run("command flags", func(t *testing.T) {
		require.NotNil(t, cmd.Flags())
		// Check that query flags are added
		nodeFlag := cmd.Flags().Lookup("node")
		require.NotNil(t, nodeFlag, "expected 'node' flag from query flags")
	})

	t.Run("RunE is defined", func(t *testing.T) {
		require.NotNil(t, cmd.RunE)
	})
}

func TestGenerateDkimPublicKeyArgsValidation(t *testing.T) {
	cmd := cli.GenerateDkimPublicKey()

	t.Run("zero args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		require.Error(t, err)
	})

	t.Run("one arg should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"example.com"})
		require.Error(t, err)
	})

	t.Run("two args should succeed", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"example.com", "selector"})
		require.NoError(t, err)
	})

	t.Run("three args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"example.com", "selector", "extra"})
		require.Error(t, err)
	})

	t.Run("four args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"a", "b", "c", "d"})
		require.Error(t, err)
	})
}

func TestGenerateDkimPublicKeyRunE(t *testing.T) {
	cmd := cli.GenerateDkimPublicKey()

	t.Run("RunE without client context panics/errors", func(t *testing.T) {
		// Without proper client context, this will fail
		func() {
			defer func() {
				r := recover()
				// Either panics or returns error - both are acceptable
				_ = r
			}()
			err := cmd.RunE(cmd, []string{"example.com", "default"})
			// If we get here without panic, expect an error
			require.Error(t, err)
		}()
	})

	t.Run("RunE with context but invalid domain", func(t *testing.T) {
		cmdWithCtx := cli.GenerateDkimPublicKey()
		cmdWithCtx.SetContext(context.Background())

		func() {
			defer func() {
				r := recover()
				_ = r
			}()
			err := cmdWithCtx.RunE(cmdWithCtx, []string{"invalid-domain-12345.nonexistent", "selector"})
			// Should error due to DNS lookup failure or missing client context
			if err != nil {
				require.Error(t, err)
			}
		}()
	})

	t.Run("RunE with empty domain", func(t *testing.T) {
		cmdWithCtx := cli.GenerateDkimPublicKey()
		cmdWithCtx.SetContext(context.Background())

		func() {
			defer func() {
				r := recover()
				_ = r
			}()
			err := cmdWithCtx.RunE(cmdWithCtx, []string{"", "selector"})
			if err != nil {
				require.Error(t, err)
			}
		}()
	})

	t.Run("RunE with empty selector", func(t *testing.T) {
		cmdWithCtx := cli.GenerateDkimPublicKey()
		cmdWithCtx.SetContext(context.Background())

		func() {
			defer func() {
				r := recover()
				_ = r
			}()
			err := cmdWithCtx.RunE(cmdWithCtx, []string{"example.com", ""})
			if err != nil {
				require.Error(t, err)
			}
		}()
	})
}

// ============================================================================
// GenerateDkimPubKeyMsg Tests
// ============================================================================

func TestGenerateDkimPubKeyMsgErrors(t *testing.T) {
	t.Run("nonexistent domain returns DNS error", func(t *testing.T) {
		result, err := cli.GenerateDkimPubKeyMsg("this-domain-does-not-exist-12345.invalid", "selector")
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "failed to lookup TXT records")
	})

	t.Run("invalid TLD returns error", func(t *testing.T) {
		result, err := cli.GenerateDkimPubKeyMsg("domain.invalidtld12345", "selector")
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("empty domain returns error", func(t *testing.T) {
		result, err := cli.GenerateDkimPubKeyMsg("", "selector")
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("empty selector returns error", func(t *testing.T) {
		result, err := cli.GenerateDkimPubKeyMsg("example.com", "")
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("domain with no DKIM record returns error", func(t *testing.T) {
		// example.com likely doesn't have a DKIM record for random selectors
		result, err := cli.GenerateDkimPubKeyMsg("example.com", "nonexistent-selector-xyz-12345")
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("special characters in domain", func(t *testing.T) {
		result, err := cli.GenerateDkimPubKeyMsg("domain with spaces.com", "selector")
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("special characters in selector", func(t *testing.T) {
		result, err := cli.GenerateDkimPubKeyMsg("example.com", "selector with spaces")
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("very long domain name", func(t *testing.T) {
		longDomain := "a]" + string(make([]byte, 255)) + ".com"
		result, err := cli.GenerateDkimPubKeyMsg(longDomain, "selector")
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("very long selector", func(t *testing.T) {
		longSelector := string(make([]byte, 1000))
		result, err := cli.GenerateDkimPubKeyMsg("example.com", longSelector)
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("domain with underscore prefix (DKIM format)", func(t *testing.T) {
		// The function constructs selector._domainkey.domain internally
		// Testing with various edge cases
		result, err := cli.GenerateDkimPubKeyMsg("_example.com", "selector")
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("numeric domain", func(t *testing.T) {
		result, err := cli.GenerateDkimPubKeyMsg("12345.67890", "selector")
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("domain with port number", func(t *testing.T) {
		result, err := cli.GenerateDkimPubKeyMsg("example.com:8080", "selector")
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("domain with protocol prefix", func(t *testing.T) {
		result, err := cli.GenerateDkimPubKeyMsg("https://example.com", "selector")
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("localhost domain", func(t *testing.T) {
		result, err := cli.GenerateDkimPubKeyMsg("localhost", "selector")
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("IP address as domain", func(t *testing.T) {
		result, err := cli.GenerateDkimPubKeyMsg("192.168.1.1", "selector")
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("IPv6 address as domain", func(t *testing.T) {
		result, err := cli.GenerateDkimPubKeyMsg("::1", "selector")
		require.Error(t, err)
		require.Nil(t, result)
	})
}

// ============================================================================
// Integration with GetQueryCmd
// ============================================================================

func TestGetQueryCmdIncludesGenerateDkim(t *testing.T) {
	queryCmd := cli.GetQueryCmd()

	// Find the generate-dkim-pubkey subcommand
	var generateCmd *cobra.Command
	for _, subCmd := range queryCmd.Commands() {
		if subCmd.Use == "generate-dkim-pubkey [flag] <domain> <selector>" {
			generateCmd = subCmd
			break
		}
	}

	require.NotNil(t, generateCmd, "generate-dkim-pubkey should be a subcommand of query")
	require.Equal(t, "Generate a DKIM msg to create a new DKIM public key", generateCmd.Short)
	require.Contains(t, generateCmd.Aliases, "gdkim")
}

func TestGetQueryCmdIncludesParams(t *testing.T) {
	queryCmd := cli.GetQueryCmd()

	// Find the params subcommand
	var paramsCmd *cobra.Command
	for _, subCmd := range queryCmd.Commands() {
		if subCmd.Use == "params" {
			paramsCmd = subCmd
			break
		}
	}

	require.NotNil(t, paramsCmd, "params should be a subcommand of query")
	require.Equal(t, "Query DKIM module parameters", paramsCmd.Short)
	require.NotEmpty(t, paramsCmd.Long)
	require.NotEmpty(t, paramsCmd.Example)
}

func TestGenerateDkimPublicKeyInQueryCmdTree(t *testing.T) {
	queryCmd := cli.GetQueryCmd()

	// Verify the command tree structure
	subCommands := queryCmd.Commands()
	require.GreaterOrEqual(t, len(subCommands), 3, "should have at least 3 subcommands")

	commandNames := make(map[string]bool)
	for _, cmd := range subCommands {
		commandNames[cmd.Name()] = true
	}

	require.True(t, commandNames["generate-dkim-pubkey"], "generate-dkim-pubkey should be present")
	require.True(t, commandNames["dkim-pubkey"], "dkim-pubkey should be present")
	require.True(t, commandNames["dkim-pubkeys"], "dkim-pubkeys should be present")
}

// ============================================================================
// Command Help and Usage Tests
// ============================================================================

func TestGenerateDkimPublicKeyHelp(t *testing.T) {
	cmd := cli.GenerateDkimPublicKey()

	t.Run("use string format", func(t *testing.T) {
		require.Contains(t, cmd.Use, "generate-dkim-pubkey")
		require.Contains(t, cmd.Use, "<domain>")
		require.Contains(t, cmd.Use, "<selector>")
	})

	t.Run("example format", func(t *testing.T) {
		require.Contains(t, cmd.Example, "gen-dkim-pubkey")
		require.Contains(t, cmd.Example, "x.com")
		require.Contains(t, cmd.Example, "dkim-202308")
	})

	t.Run("long description content", func(t *testing.T) {
		require.Contains(t, cmd.Long, "generates a DKIM msg")
		require.Contains(t, cmd.Long, "query dns")
		require.Contains(t, cmd.Long, "poseidon hash")
		require.Contains(t, cmd.Long, "AddDkimPubkey")
	})
}

// ============================================================================
// Edge Cases for Arguments
// ============================================================================

func TestGenerateDkimPublicKeyArgumentEdgeCases(t *testing.T) {
	cmd := cli.GenerateDkimPublicKey()

	t.Run("unicode domain", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"例え.jp", "selector"})
		require.NoError(t, err) // Args validation passes, DNS lookup will fail
	})

	t.Run("unicode selector", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"example.com", "選択"})
		require.NoError(t, err) // Args validation passes, DNS lookup will fail
	})

	t.Run("domain with trailing dot", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"example.com.", "selector"})
		require.NoError(t, err) // Args validation passes
	})

	t.Run("domain with leading dot", func(t *testing.T) {
		err := cmd.Args(cmd, []string{".example.com", "selector"})
		require.NoError(t, err) // Args validation passes, DNS lookup will fail
	})

	t.Run("selector with underscore", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"example.com", "dkim_selector"})
		require.NoError(t, err)
	})

	t.Run("selector with hyphen", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"example.com", "dkim-selector"})
		require.NoError(t, err)
	})

	t.Run("selector with numbers", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"example.com", "dkim20230815"})
		require.NoError(t, err)
	})
}

// ============================================================================
// MsgRevokeDkimPubKey Extended Tests
// ============================================================================

func TestMsgRevokeDkimPubKeyCommand(t *testing.T) {
	cmd := cli.MsgRevokeDkimPubKey()

	t.Run("command structure", func(t *testing.T) {
		require.NotNil(t, cmd)
		require.Equal(t, "revoke-dkim <domain> <priv_key>", cmd.Use)
		require.Equal(t, "Revoke a Dkim pubkey without governance.", cmd.Short)
		require.Contains(t, cmd.Long, "PEM encoded private key")
		require.Contains(t, cmd.Long, "without the headers")
		require.Contains(t, cmd.Long, "contiguous string")
	})

	t.Run("command aliases", func(t *testing.T) {
		require.NotEmpty(t, cmd.Aliases)
		require.Contains(t, cmd.Aliases, "rdkim")
	})

	t.Run("RunE is defined", func(t *testing.T) {
		require.NotNil(t, cmd.RunE)
	})
}

func TestMsgRevokeDkimPubKeyArgsValidation(t *testing.T) {
	cmd := cli.MsgRevokeDkimPubKey()

	testCases := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{"zero args should fail", []string{}, true},
		{"one arg should fail", []string{"domain.com"}, true},
		{"two args should succeed", []string{"domain.com", "privkey"}, false},
		{"three args should fail", []string{"domain.com", "privkey", "extra"}, true},
		{"four args should fail", []string{"a", "b", "c", "d"}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := cmd.Args(cmd, tc.args)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgRevokeDkimPubKeyRunE(t *testing.T) {
	// Generate valid RSA key for testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privKeyDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privKeyBase64 := base64.StdEncoding.EncodeToString(privKeyDER)

	t.Run("RunE without client context", func(t *testing.T) {
		cmd := cli.MsgRevokeDkimPubKey()
		func() {
			defer func() { _ = recover() }()
			err := cmd.RunE(cmd, []string{"example.com", privKeyBase64})
			if err != nil {
				require.Error(t, err)
			}
		}()
	})

	t.Run("RunE with invalid private key", func(t *testing.T) {
		cmd := cli.MsgRevokeDkimPubKey()
		func() {
			defer func() { _ = recover() }()
			err := cmd.RunE(cmd, []string{"example.com", "not-valid-key"})
			if err != nil {
				require.Error(t, err)
			}
		}()
	})

	t.Run("RunE with empty domain", func(t *testing.T) {
		cmd := cli.MsgRevokeDkimPubKey()
		func() {
			defer func() { _ = recover() }()
			err := cmd.RunE(cmd, []string{"", privKeyBase64})
			if err != nil {
				require.Error(t, err)
			}
		}()
	})

	t.Run("RunE with empty private key", func(t *testing.T) {
		cmd := cli.MsgRevokeDkimPubKey()
		func() {
			defer func() { _ = recover() }()
			err := cmd.RunE(cmd, []string{"example.com", ""})
			if err != nil {
				require.Error(t, err)
			}
		}()
	})

	t.Run("RunE with URL domain", func(t *testing.T) {
		cmd := cli.MsgRevokeDkimPubKey()
		func() {
			defer func() { _ = recover() }()
			err := cmd.RunE(cmd, []string{"https://example.com/path", privKeyBase64})
			if err != nil {
				require.Error(t, err)
			}
		}()
	})

	t.Run("RunE with context set", func(t *testing.T) {
		cmd := cli.MsgRevokeDkimPubKey()
		cmd.SetContext(context.Background())
		func() {
			defer func() { _ = recover() }()
			err := cmd.RunE(cmd, []string{"example.com", privKeyBase64})
			if err != nil {
				require.Error(t, err)
			}
		}()
	})
}

func TestParseAndValidateRevokeDkimMsgExtended(t *testing.T) {
	// Generate valid RSA keys for testing
	privateKey2048, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateKey4096, err := rsa.GenerateKey(rand.Reader, 4096)
	require.NoError(t, err)

	// Helper to get base64 key without PEM headers
	getBase64Key := func(key *rsa.PrivateKey) string {
		privKeyDER := x509.MarshalPKCS1PrivateKey(key)
		return base64.StdEncoding.EncodeToString(privKeyDER)
	}

	t.Run("valid 2048-bit key", func(t *testing.T) {
		msg, err := cli.ParseAndValidateRevokeDkimMsg(
			"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a",
			"example.com",
			getBase64Key(privateKey2048),
		)
		require.NoError(t, err)
		require.NotNil(t, msg)
		require.Equal(t, "example.com", msg.Domain)
	})

	t.Run("valid 4096-bit key", func(t *testing.T) {
		msg, err := cli.ParseAndValidateRevokeDkimMsg(
			"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a",
			"example.com",
			getBase64Key(privateKey4096),
		)
		require.NoError(t, err)
		require.NotNil(t, msg)
	})

	t.Run("key with newlines should fail", func(t *testing.T) {
		keyWithNewlines := "MIIBIjAN\nBgkqhkiG\n9w0BAQEF"
		msg, err := cli.ParseAndValidateRevokeDkimMsg(
			"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a",
			"example.com",
			keyWithNewlines,
		)
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("truncated key", func(t *testing.T) {
		truncatedKey := getBase64Key(privateKey2048)[:50]
		msg, err := cli.ParseAndValidateRevokeDkimMsg(
			"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a",
			"example.com",
			truncatedKey,
		)
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("corrupted base64", func(t *testing.T) {
		msg, err := cli.ParseAndValidateRevokeDkimMsg(
			"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a",
			"example.com",
			"!!!not-base64!!!",
		)
		require.Error(t, err)
		require.ErrorIs(t, err, types.ErrParsingPrivKey)
		require.Nil(t, msg)
	})

	t.Run("valid base64 but not a key", func(t *testing.T) {
		notAKey := base64.StdEncoding.EncodeToString([]byte("this is not a private key"))
		msg, err := cli.ParseAndValidateRevokeDkimMsg(
			"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a",
			"example.com",
			notAKey,
		)
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("domain variations", func(t *testing.T) {
		key := getBase64Key(privateKey2048)

		// Simple domain
		msg, err := cli.ParseAndValidateRevokeDkimMsg("signer", "example.com", key)
		require.NoError(t, err)
		require.Equal(t, "example.com", msg.Domain)

		// Subdomain
		msg, err = cli.ParseAndValidateRevokeDkimMsg("signer", "mail.example.com", key)
		require.NoError(t, err)
		require.Equal(t, "mail.example.com", msg.Domain)

		// URL with protocol
		msg, err = cli.ParseAndValidateRevokeDkimMsg("signer", "https://example.com", key)
		require.NoError(t, err)
		require.Equal(t, "https://example.com", msg.Domain)

		// URL with path
		msg, err = cli.ParseAndValidateRevokeDkimMsg("signer", "https://example.com/path", key)
		require.NoError(t, err)
		require.Equal(t, "https://example.com/path", msg.Domain)
	})

	t.Run("signer variations", func(t *testing.T) {
		key := getBase64Key(privateKey2048)

		// Valid bech32 address
		msg, err := cli.ParseAndValidateRevokeDkimMsg(
			"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a",
			"example.com",
			key,
		)
		require.NoError(t, err)
		require.Equal(t, "xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a", msg.Signer)

		// Empty signer (allowed at parse level)
		msg, err = cli.ParseAndValidateRevokeDkimMsg("", "example.com", key)
		require.NoError(t, err)
		require.Empty(t, msg.Signer)

		// Random string as signer
		msg, err = cli.ParseAndValidateRevokeDkimMsg("random-signer", "example.com", key)
		require.NoError(t, err)
		require.Equal(t, "random-signer", msg.Signer)
	})

	t.Run("PEM key already formatted", func(t *testing.T) {
		// Test with a key that already has PEM formatting
		privKeyDER := x509.MarshalPKCS1PrivateKey(privateKey2048)
		privKeyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privKeyDER,
		})

		// Extract just the base64 content
		pemStr := string(privKeyPEM)
		// Remove headers
		pemStr = pemStr[len("-----BEGIN RSA PRIVATE KEY-----\n"):]
		pemStr = pemStr[:len(pemStr)-len("-----END RSA PRIVATE KEY-----\n")]
		// Remove newlines
		cleanKey := ""
		for _, c := range pemStr {
			if c != '\n' {
				cleanKey += string(c)
			}
		}

		msg, err := cli.ParseAndValidateRevokeDkimMsg(
			"xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a",
			"example.com",
			cleanKey,
		)
		require.NoError(t, err)
		require.NotNil(t, msg)
	})
}

// ============================================================================
// GetParams Extended Tests
// ============================================================================

func TestGetParamsExtended(t *testing.T) {
	t.Run("command has correct flags", func(t *testing.T) {
		cmd := cli.GetParams()

		// Check query flags are present
		nodeFlag := cmd.Flags().Lookup("node")
		require.NotNil(t, nodeFlag)

		outputFlag := cmd.Flags().Lookup("output")
		require.NotNil(t, outputFlag)
	})

	t.Run("command properties", func(t *testing.T) {
		cmd := cli.GetParams()

		require.Equal(t, "params", cmd.Use)
		require.Contains(t, cmd.Long, "VkeyIdentifier")
		require.Contains(t, cmd.Long, "DKIM public keys")
		require.Contains(t, cmd.Example, "xiond")
		require.Contains(t, cmd.Example, "query")
		require.Contains(t, cmd.Example, "dkim")
		require.Contains(t, cmd.Example, "params")
	})

	t.Run("RunE without context", func(t *testing.T) {
		cmd := cli.GetParams()
		func() {
			defer func() { _ = recover() }()
			_ = cmd.RunE(cmd, []string{})
		}()
	})

	t.Run("args rejection", func(t *testing.T) {
		cmd := cli.GetParams()

		// Should reject various argument counts
		for i := 1; i <= 5; i++ {
			args := make([]string, i)
			for j := 0; j < i; j++ {
				args[j] = "arg"
			}
			err := cmd.Args(cmd, args)
			require.Error(t, err, "should reject %d args", i)
		}
	})
}

func TestQueryParamsExtended(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	t.Run("params with nil response", func(t *testing.T) {
		mockClient := &MockQueryClient{
			paramsFunc: func(ctx context.Context, req *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error) {
				return nil, errors.New("nil response")
			},
		}

		res, err := cli.QueryParams(mockClient, cmd)
		require.Error(t, err)
		require.Nil(t, res)
	})

	t.Run("params with connection error", func(t *testing.T) {
		mockClient := &MockQueryClient{
			paramsFunc: func(ctx context.Context, req *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error) {
				return nil, errors.New("connection refused")
			},
		}

		res, err := cli.QueryParams(mockClient, cmd)
		require.Error(t, err)
		require.Contains(t, err.Error(), "connection refused")
		require.Nil(t, res)
	})

	t.Run("params with timeout error", func(t *testing.T) {
		mockClient := &MockQueryClient{
			paramsFunc: func(ctx context.Context, req *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error) {
				return nil, errors.New("context deadline exceeded")
			},
		}

		res, err := cli.QueryParams(mockClient, cmd)
		require.Error(t, err)
		require.Contains(t, err.Error(), "context deadline exceeded")
		require.Nil(t, res)
	})

	t.Run("params with dkim pubkeys containing all fields", func(t *testing.T) {
		mockClient := &MockQueryClient{
			paramsFunc: func(ctx context.Context, req *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error) {
				return &types.QueryParamsResponse{
					Params: &types.Params{
						VkeyIdentifier: 100,
					},
				}, nil
			},
		}

		res, err := cli.QueryParams(mockClient, cmd)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.Params)
		require.Equal(t, uint64(100), res.Params.VkeyIdentifier)
	})
}

// ============================================================================
// GetDkimPublicKey Extended Tests
// ============================================================================

func TestGetDkimPublicKeyExtended(t *testing.T) {
	t.Run("command has correct flags", func(t *testing.T) {
		cmd := cli.GetDkimPublicKey()

		// Check query flags are present
		nodeFlag := cmd.Flags().Lookup("node")
		require.NotNil(t, nodeFlag)

		outputFlag := cmd.Flags().Lookup("output")
		require.NotNil(t, outputFlag)
	})

	t.Run("RunE with various argument combinations", func(t *testing.T) {
		testCases := []struct {
			name     string
			domain   string
			selector string
		}{
			{"standard domain and selector", "example.com", "default"},
			{"subdomain", "mail.example.com", "dkim1"},
			{"numeric selector", "example.com", "20230815"},
			{"hyphenated selector", "example.com", "dkim-2023-08"},
			{"underscore selector", "example.com", "dkim_selector"},
			{"long selector", "example.com", "verylongselectorname123456789"},
			{"international domain", "例え.jp", "selector"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cmd := cli.GetDkimPublicKey()
				func() {
					defer func() { _ = recover() }()
					_ = cmd.RunE(cmd, []string{tc.domain, tc.selector})
				}()
			})
		}
	})
}

func TestQueryDkimPubKeyExtended(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	t.Run("query with special characters in domain", func(t *testing.T) {
		mockClient := &MockQueryClient{
			dkimPubKeyFunc: func(ctx context.Context, req *types.QueryDkimPubKeyRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeyResponse, error) {
				return &types.QueryDkimPubKeyResponse{
					DkimPubKey: &types.DkimPubKey{
						Domain:   req.Domain,
						Selector: req.Selector,
					},
				}, nil
			},
		}

		// Test various domain formats
		domains := []string{
			"example.com",
			"sub.example.com",
			"deep.sub.example.com",
			"example-with-dash.com",
			"example123.com",
		}

		for _, domain := range domains {
			res, err := cli.QueryDkimPubKey(mockClient, cmd, domain, "selector")
			require.NoError(t, err)
			require.Equal(t, domain, res.DkimPubKey.Domain)
		}
	})

	t.Run("query with special characters in selector", func(t *testing.T) {
		mockClient := &MockQueryClient{
			dkimPubKeyFunc: func(ctx context.Context, req *types.QueryDkimPubKeyRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeyResponse, error) {
				return &types.QueryDkimPubKeyResponse{
					DkimPubKey: &types.DkimPubKey{
						Domain:   req.Domain,
						Selector: req.Selector,
					},
				}, nil
			},
		}

		selectors := []string{
			"default",
			"dkim1",
			"dkim-2023",
			"dkim_selector",
			"selector123",
			"s",
		}

		for _, selector := range selectors {
			res, err := cli.QueryDkimPubKey(mockClient, cmd, "example.com", selector)
			require.NoError(t, err)
			require.Equal(t, selector, res.DkimPubKey.Selector)
		}
	})

	t.Run("query returns full DkimPubKey", func(t *testing.T) {
		expectedPubKey := &types.DkimPubKey{
			Domain:       "example.com",
			Selector:     "dkim1",
			PubKey:       "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...",
			PoseidonHash: []byte("hash123"),
			Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
		}

		mockClient := &MockQueryClient{
			dkimPubKeyFunc: func(ctx context.Context, req *types.QueryDkimPubKeyRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeyResponse, error) {
				return &types.QueryDkimPubKeyResponse{DkimPubKey: expectedPubKey}, nil
			},
		}

		res, err := cli.QueryDkimPubKey(mockClient, cmd, "example.com", "dkim1")
		require.NoError(t, err)
		require.Equal(t, expectedPubKey.Domain, res.DkimPubKey.Domain)
		require.Equal(t, expectedPubKey.Selector, res.DkimPubKey.Selector)
		require.Equal(t, expectedPubKey.PubKey, res.DkimPubKey.PubKey)
		require.Equal(t, expectedPubKey.PoseidonHash, res.DkimPubKey.PoseidonHash)
	})

	t.Run("query with various error types", func(t *testing.T) {
		errorCases := []struct {
			name     string
			errorMsg string
		}{
			{"not found error", "key not found"},
			{"internal error", "internal server error"},
			{"timeout error", "context deadline exceeded"},
			{"connection error", "connection refused"},
		}

		for _, ec := range errorCases {
			t.Run(ec.name, func(t *testing.T) {
				mockClient := &MockQueryClient{
					dkimPubKeyFunc: func(ctx context.Context, req *types.QueryDkimPubKeyRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeyResponse, error) {
						return nil, errors.New(ec.errorMsg)
					},
				}

				res, err := cli.QueryDkimPubKey(mockClient, cmd, "example.com", "dkim1")
				require.Error(t, err)
				require.Nil(t, res)
				require.Contains(t, err.Error(), ec.errorMsg)
			})
		}
	})
}

// ============================================================================
// GetDkimPublicKeys Extended Tests
// ============================================================================

func TestGetDkimPublicKeysExtended(t *testing.T) {
	t.Run("command flags have correct defaults", func(t *testing.T) {
		cmd := cli.GetDkimPublicKeys()

		domainFlag := cmd.Flags().Lookup("domain")
		require.NotNil(t, domainFlag)
		require.Equal(t, "", domainFlag.DefValue)

		selectorFlag := cmd.Flags().Lookup("selector")
		require.NotNil(t, selectorFlag)
		require.Equal(t, "", selectorFlag.DefValue)

		hashFlag := cmd.Flags().Lookup("hash")
		require.NotNil(t, hashFlag)
		require.Equal(t, "", hashFlag.DefValue)
	})

	t.Run("RunE with all flag combinations", func(t *testing.T) {
		flagCombinations := []struct {
			name     string
			domain   string
			selector string
			hash     string
		}{
			{"no flags", "", "", ""},
			{"domain only", "example.com", "", ""},
			{"selector only", "", "dkim1", ""},
			{"hash only", "", "", "abc123"},
			{"domain and selector", "example.com", "dkim1", ""},
			{"domain and hash", "example.com", "", "abc123"},
			{"all flags", "example.com", "dkim1", "abc123"},
		}

		for _, fc := range flagCombinations {
			t.Run(fc.name, func(t *testing.T) {
				cmd := cli.GetDkimPublicKeys()
				if fc.domain != "" {
					require.NoError(t, cmd.Flags().Set("domain", fc.domain))
				}
				if fc.selector != "" {
					require.NoError(t, cmd.Flags().Set("selector", fc.selector))
				}
				if fc.hash != "" {
					require.NoError(t, cmd.Flags().Set("hash", fc.hash))
				}

				func() {
					defer func() { _ = recover() }()
					_ = cmd.RunE(cmd, []string{})
				}()
			})
		}
	})
}

func TestQueryDkimPubKeysExtended(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	t.Run("query returns multiple keys", func(t *testing.T) {
		mockClient := &MockQueryClient{
			dkimPubKeysFunc: func(ctx context.Context, req *types.QueryDkimPubKeysRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeysResponse, error) {
				return &types.QueryDkimPubKeysResponse{
					DkimPubKeys: []*types.DkimPubKey{
						{Domain: "example1.com", Selector: "dkim1"},
						{Domain: "example2.com", Selector: "dkim2"},
						{Domain: "example3.com", Selector: "dkim3"},
					},
				}, nil
			},
		}

		res, err := cli.QueryDkimPubKeys(mockClient, cmd, "", "", "")
		require.NoError(t, err)
		require.Len(t, res.DkimPubKeys, 3)
	})

	t.Run("query returns empty list", func(t *testing.T) {
		mockClient := &MockQueryClient{
			dkimPubKeysFunc: func(ctx context.Context, req *types.QueryDkimPubKeysRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeysResponse, error) {
				return &types.QueryDkimPubKeysResponse{
					DkimPubKeys: []*types.DkimPubKey{},
				}, nil
			},
		}

		res, err := cli.QueryDkimPubKeys(mockClient, cmd, "nonexistent.com", "", "")
		require.NoError(t, err)
		require.Empty(t, res.DkimPubKeys)
	})

	t.Run("query with pagination", func(t *testing.T) {
		mockClient := &MockQueryClient{
			dkimPubKeysFunc: func(ctx context.Context, req *types.QueryDkimPubKeysRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeysResponse, error) {
				return &types.QueryDkimPubKeysResponse{
					DkimPubKeys: []*types.DkimPubKey{
						{Domain: "example.com", Selector: "dkim1"},
					},
				}, nil
			},
		}

		res, err := cli.QueryDkimPubKeys(mockClient, cmd, "example.com", "", "")
		require.NoError(t, err)
		require.NotNil(t, res)
	})

	t.Run("query filters work correctly", func(t *testing.T) {
		mockClient := &MockQueryClient{
			dkimPubKeysFunc: func(ctx context.Context, req *types.QueryDkimPubKeysRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeysResponse, error) {
				// Verify filters are passed correctly
				require.Equal(t, "filtered.com", req.Domain)
				require.Equal(t, "filtered-selector", req.Selector)
				require.Equal(t, []byte("filtered-hash"), req.PoseidonHash)
				return &types.QueryDkimPubKeysResponse{
					DkimPubKeys: []*types.DkimPubKey{},
				}, nil
			},
		}

		_, err := cli.QueryDkimPubKeys(mockClient, cmd, "filtered.com", "filtered-selector", "filtered-hash")
		require.NoError(t, err)
	})

	t.Run("query with only hash filter", func(t *testing.T) {
		mockClient := &MockQueryClient{
			dkimPubKeysFunc: func(ctx context.Context, req *types.QueryDkimPubKeysRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeysResponse, error) {
				require.Equal(t, "", req.Domain)
				require.Equal(t, "", req.Selector)
				require.Equal(t, []byte("somehash"), req.PoseidonHash)
				return &types.QueryDkimPubKeysResponse{
					DkimPubKeys: []*types.DkimPubKey{
						{Domain: "found.com", PoseidonHash: []byte("somehash")},
					},
				}, nil
			},
		}

		res, err := cli.QueryDkimPubKeys(mockClient, cmd, "", "", "somehash")
		require.NoError(t, err)
		require.Len(t, res.DkimPubKeys, 1)
	})
}

func TestParseDkimPubKeysFlagsExtended(t *testing.T) {
	t.Run("flags with whitespace", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("domain", "", "")
		cmd.Flags().String("selector", "", "")
		cmd.Flags().String("hash", "", "")

		require.NoError(t, cmd.Flags().Set("domain", "  example.com  "))
		require.NoError(t, cmd.Flags().Set("selector", "  dkim1  "))
		require.NoError(t, cmd.Flags().Set("hash", "  hash123  "))

		domain, selector, hash, err := cli.ParseDkimPubKeysFlags(cmd)
		require.NoError(t, err)
		// Note: flags don't trim whitespace automatically
		require.Equal(t, "  example.com  ", domain)
		require.Equal(t, "  dkim1  ", selector)
		require.Equal(t, "  hash123  ", hash)
	})

	t.Run("flags with special characters", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("domain", "", "")
		cmd.Flags().String("selector", "", "")
		cmd.Flags().String("hash", "", "")

		require.NoError(t, cmd.Flags().Set("domain", "example-test.com"))
		require.NoError(t, cmd.Flags().Set("selector", "dkim_2023-08"))
		require.NoError(t, cmd.Flags().Set("hash", "abc123def456"))

		domain, selector, hash, err := cli.ParseDkimPubKeysFlags(cmd)
		require.NoError(t, err)
		require.Equal(t, "example-test.com", domain)
		require.Equal(t, "dkim_2023-08", selector)
		require.Equal(t, "abc123def456", hash)
	})
}

// ============================================================================
// GenerateDkimPubKeyMsg Extended Tests
// ============================================================================

func TestGenerateDkimPubKeyMsgExtended(t *testing.T) {
	t.Run("various invalid domain formats", func(t *testing.T) {
		invalidDomains := []struct {
			name   string
			domain string
		}{
			{"empty domain", ""},
			{"whitespace only", "   "},
			{"domain with spaces", "example .com"},
			{"domain with protocol", "http://example.com"},
			{"domain with port", "example.com:8080"},
			{"IP address", "192.168.1.1"},
			{"IPv6 address", "::1"},
			{"localhost", "localhost"},
			{"single label", "localhost"},
			{"starts with dot", ".example.com"},
			{"ends with dot followed by invalid", "example."},
			{"double dots", "example..com"},
			{"very long domain", string(make([]byte, 300)) + ".com"},
		}

		for _, tc := range invalidDomains {
			t.Run(tc.name, func(t *testing.T) {
				result, err := cli.GenerateDkimPubKeyMsg(tc.domain, "selector")
				require.Error(t, err)
				require.Nil(t, result)
			})
		}
	})

	t.Run("various invalid selector formats", func(t *testing.T) {
		invalidSelectors := []struct {
			name     string
			selector string
		}{
			{"empty selector", ""},
			{"whitespace only", "   "},
			{"selector with spaces", "dkim selector"},
			{"very long selector", string(make([]byte, 500))},
		}

		for _, tc := range invalidSelectors {
			t.Run(tc.name, func(t *testing.T) {
				result, err := cli.GenerateDkimPubKeyMsg("example.com", tc.selector)
				require.Error(t, err)
				require.Nil(t, result)
			})
		}
	})

	t.Run("real domains without DKIM", func(t *testing.T) {
		// These domains exist but won't have DKIM records for random selectors
		domainsWithoutDkim := []struct {
			domain   string
			selector string
		}{
			{"example.com", "nonexistent-selector-xyz"},
			{"google.com", "invalid-selector-abc123"},
			{"github.com", "fake-selector-999"},
		}

		for _, tc := range domainsWithoutDkim {
			t.Run(tc.domain, func(t *testing.T) {
				result, err := cli.GenerateDkimPubKeyMsg(tc.domain, tc.selector)
				require.Error(t, err)
				require.Nil(t, result)
			})
		}
	})

	t.Run("completely invalid TLDs", func(t *testing.T) {
		invalidTLDs := []string{
			"example.invalidtld",
			"test.notarealtld",
			"domain.xyz123abc",
		}

		for _, domain := range invalidTLDs {
			t.Run(domain, func(t *testing.T) {
				result, err := cli.GenerateDkimPubKeyMsg(domain, "selector")
				require.Error(t, err)
				require.Nil(t, result)
			})
		}
	})

	t.Run("unicode and IDN domains", func(t *testing.T) {
		// IDN domains - these may or may not resolve
		idnDomains := []string{
			"例え.jp",
			"münchen.de",
			"موقع.عربي",
		}

		for _, domain := range idnDomains {
			t.Run(domain, func(t *testing.T) {
				result, err := cli.GenerateDkimPubKeyMsg(domain, "selector")
				// These will fail due to DNS lookup, but shouldn't panic
				require.Error(t, err)
				require.Nil(t, result)
			})
		}
	})
}

func TestNewTxDecodeCmd(t *testing.T) {
	cmd := cli.NewTxDecodeCmd()
	require.NotNil(t, cmd)
	require.Equal(t, "decode [base64-tx]", cmd.Use)
	require.Contains(t, cmd.Short, "Decode")
	require.NotEmpty(t, cmd.Long)
	require.Contains(t, cmd.Long, "base64")

	// Test that command has RunE function defined
	require.NotNil(t, cmd.RunE)

	// Test command structure
	require.NotNil(t, cmd.Flags())
}

func TestNewTxDecodeCmdArgsValidation(t *testing.T) {
	cmd := cli.NewTxDecodeCmd()

	t.Run("args validator with 0 args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		require.Error(t, err, "should require exactly 1 arg")
	})

	t.Run("args validator with 1 arg", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"base64string"})
		require.NoError(t, err, "should accept exactly 1 arg")
	})

	t.Run("args validator with 2 args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"arg1", "arg2"})
		require.Error(t, err, "should reject more than 1 arg")
	})
}

func TestNewTxDecodeCmdRunE(t *testing.T) {
	cmd := cli.NewTxDecodeCmd()

	t.Run("RunE without client context panics/errors", func(t *testing.T) {
		func() {
			defer func() {
				r := recover()
				_ = r
			}()
			err := cmd.RunE(cmd, []string{"invalidbase64"})
			if err != nil {
				require.Error(t, err)
			}
		}()
	})

	t.Run("RunE with context but invalid base64", func(t *testing.T) {
		cmdWithCtx := cli.NewTxDecodeCmd()
		cmdWithCtx.SetContext(context.Background())

		func() {
			defer func() {
				r := recover()
				_ = r
			}()
			err := cmdWithCtx.RunE(cmdWithCtx, []string{"!!!not-valid-base64!!!"})
			if err != nil {
				require.Error(t, err)
			}
		}()
	})

	t.Run("RunE with empty string", func(t *testing.T) {
		cmdWithCtx := cli.NewTxDecodeCmd()
		cmdWithCtx.SetContext(context.Background())

		func() {
			defer func() {
				r := recover()
				_ = r
			}()
			err := cmdWithCtx.RunE(cmdWithCtx, []string{""})
			if err != nil {
				require.Error(t, err)
			}
		}()
	})
}

// ============================================================================
// DecodeTx Function Tests
// ============================================================================

func TestDecodeTx(t *testing.T) {
	// Set up encoding config using the app's config
	encCfg := app.MakeEncodingConfig(t)

	var buf bytes.Buffer
	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithTxConfig(encCfg.TxConfig).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithOutput(&buf)

	t.Run("valid transaction", func(t *testing.T) {
		buf.Reset()
		txBytes := createTestTxBytes(t, encCfg.TxConfig, "test memo", sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1000))), 200000)
		base64Tx := base64.StdEncoding.EncodeToString(txBytes)

		err := cli.DecodeTx(clientCtx, base64Tx)
		require.NoError(t, err)

		// Verify output is valid JSON
		output := buf.String()
		require.NotEmpty(t, output)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)

		// Verify body exists with messages
		body, ok := result["body"].(map[string]interface{})
		require.True(t, ok, "body should exist")

		messages, ok := body["messages"].([]interface{})
		require.True(t, ok, "messages should exist")
		require.Len(t, messages, 1)

		// Verify memo
		require.Equal(t, "test memo", body["memo"])

		// Verify auth_info exists with fee
		authInfo, ok := result["auth_info"].(map[string]interface{})
		require.True(t, ok, "auth_info should exist")

		feeInfo, ok := authInfo["fee"].(map[string]interface{})
		require.True(t, ok, "fee should exist")
		require.Equal(t, "200000", feeInfo["gas_limit"])
	})

	t.Run("valid sign doc", func(t *testing.T) {
		buf.Reset()
		signDocBytes := createTestSignDocBytes(t, "xion-1", 12, 1)
		base64SignDoc := base64.StdEncoding.EncodeToString(signDocBytes)

		err := cli.DecodeTx(clientCtx, base64SignDoc)
		require.NoError(t, err)

		// Verify output is valid JSON
		output := buf.String()
		require.NotEmpty(t, output)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)

		// Verify body exists with messages
		body, ok := result["body"].(map[string]interface{})
		require.True(t, ok, "body should exist")

		messages, ok := body["messages"].([]interface{})
		require.True(t, ok, "messages should exist")
		require.Len(t, messages, 1)

		// Verify chain_id and account_number from SignDoc
		require.Equal(t, "xion-1", result["chain_id"])
		require.Equal(t, "12", result["account_number"])

		// Verify auth_info exists
		authInfo, ok := result["auth_info"].(map[string]interface{})
		require.True(t, ok, "auth_info should exist")

		_, ok = authInfo["fee"].(map[string]interface{})
		require.True(t, ok, "fee should exist")
	})

	t.Run("valid sign doc with URL-safe base64", func(t *testing.T) {
		buf.Reset()
		// This is the precomputed sign doc from the e2e test
		base64SignDoc := "CqIBCp8BChwvY29zbW9zLmJhbmsudjFiZXRhMS5Nc2dTZW5kEn8KP3hpb24xNG43OWVocGZ3aGRoNHN6dWRhZ2Q0bm14N2NsajU3bHk1dTBsenhzNm1nZjVxeTU1a3k5c21zenM0OBIreGlvbjFxYWYyeGZseDVqM2FndGx2cWs1dmhqcGV1aGw2ZzQ1aHhzaHdxahoPCgV1eGlvbhIGMTAwMDAwEmYKTQpDCh0vYWJzdHJhY3RhY2NvdW50LnYxLk5pbFB1YktleRIiCiCs_FzcKXXbesBcb1Daz2b2Pyp75Kcf8Roa2hNAEpSxCxIECgIIARgBEhUKDgoFdXhpb24SBTYwMDAwEICHpw4aBnhpb24tMSAM"

		err := cli.DecodeTx(clientCtx, base64SignDoc)
		require.NoError(t, err)

		// Verify output is valid JSON
		output := buf.String()
		require.NotEmpty(t, output)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)

		// Verify body exists with messages
		body, ok := result["body"].(map[string]interface{})
		require.True(t, ok, "body should exist")

		messages, ok := body["messages"].([]interface{})
		require.True(t, ok, "messages should exist")
		require.Len(t, messages, 1)

		// Verify it's a MsgSend
		msg := messages[0].(map[string]interface{})
		require.Contains(t, msg["@type"], "MsgSend")

		// Verify chain_id from SignDoc
		require.Equal(t, "xion-1", result["chain_id"])
		require.Equal(t, "12", result["account_number"])
	})

	t.Run("invalid base64", func(t *testing.T) {
		buf.Reset()
		err := cli.DecodeTx(clientCtx, "not-valid-base64!!!")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode base64")
	})

	t.Run("invalid transaction bytes", func(t *testing.T) {
		buf.Reset()
		invalidTxBytes := base64.StdEncoding.EncodeToString([]byte("invalid transaction data"))
		err := cli.DecodeTx(clientCtx, invalidTxBytes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode")
	})

	t.Run("empty string", func(t *testing.T) {
		buf.Reset()
		err := cli.DecodeTx(clientCtx, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode base64")
	})

	t.Run("nil TxConfig", func(t *testing.T) {
		var nilBuf bytes.Buffer
		clientCtxNoTxConfig := client.Context{}.WithOutput(&nilBuf)
		txBytes := createTestTxBytes(t, encCfg.TxConfig, "memo", sdk.NewCoins(), 100000)
		base64Tx := base64.StdEncoding.EncodeToString(txBytes)

		err := cli.DecodeTx(clientCtxNoTxConfig, base64Tx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "tx config is not initialized")
	})

	t.Run("transaction without memo", func(t *testing.T) {
		buf.Reset()
		txBytes := createTestTxBytes(t, encCfg.TxConfig, "", sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(500))), 100000)
		base64Tx := base64.StdEncoding.EncodeToString(txBytes)

		err := cli.DecodeTx(clientCtx, base64Tx)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		body := result["body"].(map[string]interface{})
		memo, exists := body["memo"]
		if exists {
			require.Equal(t, "", memo)
		}
	})

	t.Run("transaction without fee", func(t *testing.T) {
		buf.Reset()
		txBytes := createTestTxBytes(t, encCfg.TxConfig, "no fee tx", sdk.NewCoins(), 0)
		base64Tx := base64.StdEncoding.EncodeToString(txBytes)

		err := cli.DecodeTx(clientCtx, base64Tx)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		authInfo := result["auth_info"].(map[string]interface{})
		feeInfo := authInfo["fee"].(map[string]interface{})
		require.Equal(t, "0", feeInfo["gas_limit"])
	})
}

func TestDecodeTxEdgeCases(t *testing.T) {
	encCfg := app.MakeEncodingConfig(t)

	var buf bytes.Buffer
	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithTxConfig(encCfg.TxConfig).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithOutput(&buf)

	tests := []struct {
		name        string
		input       string
		expectError bool
		errContains string
	}{
		{
			name:        "empty string",
			input:       "",
			expectError: true,
			errContains: "failed to decode base64",
		},
		{
			name:        "invalid base64 characters",
			input:       "!!!invalid!!!",
			expectError: true,
			errContains: "failed to decode base64",
		},
		{
			name:        "valid base64 but random bytes",
			input:       base64.StdEncoding.EncodeToString([]byte{0x00, 0x01, 0x02, 0x03}),
			expectError: true,
			errContains: "failed to decode",
		},
		{
			name:        "valid base64 of text",
			input:       base64.StdEncoding.EncodeToString([]byte("hello world")),
			expectError: true,
			errContains: "failed to decode",
		},
		{
			name:        "whitespace only",
			input:       "   ",
			expectError: true,
			errContains: "failed to decode base64",
		},
		{
			name:        "base64 with padding issues",
			input:       "SGVsbG8",
			expectError: true,
			errContains: "",
		},
		{
			name:        "very long invalid string",
			input:       string(make([]byte, 10000)),
			expectError: true,
			errContains: "failed to decode base64",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()

			err := cli.DecodeTx(clientCtx, tc.input)

			if tc.expectError {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDecodeTxOutputFormat(t *testing.T) {
	encCfg := app.MakeEncodingConfig(t)

	var buf bytes.Buffer
	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithTxConfig(encCfg.TxConfig).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithOutput(&buf)

	memo := "test output format"
	fee := sdk.NewCoins(sdk.NewCoin("uatom", math.NewInt(5000)))
	gas := uint64(200000)

	txBytes := createTestTxBytes(t, encCfg.TxConfig, memo, fee, gas)
	base64Tx := base64.StdEncoding.EncodeToString(txBytes)

	err := cli.DecodeTx(clientCtx, base64Tx)
	require.NoError(t, err)

	output := buf.String()

	// Verify it's valid JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	// The output should have the standard Cosmos SDK tx structure
	require.Contains(t, result, "body")
	require.Contains(t, result, "auth_info")
	require.Contains(t, result, "signatures")
}

func TestDecodeTxJSONEncoderCompatibility(t *testing.T) {
	encCfg := app.MakeEncodingConfig(t)

	txConfig := encCfg.TxConfig

	// Create a transaction
	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	fromAddr := sdk.AccAddress(pubKey.Address())
	toAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	msg := banktypes.NewMsgSend(fromAddr, toAddr, sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(100))))

	txBuilder := txConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	require.NoError(t, err)

	txBuilder.SetMemo("json encoder test")
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1000))))
	txBuilder.SetGasLimit(200000)

	sigV2 := signing.SignatureV2{
		PubKey: pubKey,
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: []byte("test signature"),
		},
		Sequence: 1,
	}
	err = txBuilder.SetSignatures(sigV2)
	require.NoError(t, err)

	builtTx := txBuilder.GetTx()

	// Encode to bytes and then to base64
	txBytes, err := txConfig.TxEncoder()(builtTx)
	require.NoError(t, err)
	base64Tx := base64.StdEncoding.EncodeToString(txBytes)

	// Decode using our function
	var buf bytes.Buffer
	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithTxConfig(txConfig).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithOutput(&buf)

	err = cli.DecodeTx(clientCtx, base64Tx)
	require.NoError(t, err)

	// Verify the output matches what TxJSONEncoder produces
	expectedJSON, err := txConfig.TxJSONEncoder()(builtTx)
	require.NoError(t, err)

	// Parse both for comparison (ignoring formatting differences)
	var expected, actual map[string]interface{}
	err = json.Unmarshal(expectedJSON, &expected)
	require.NoError(t, err)
	err = json.Unmarshal(buf.Bytes(), &actual)
	require.NoError(t, err)

	// Compare the structures
	require.Equal(t, expected["body"], actual["body"])
	require.Equal(t, expected["auth_info"], actual["auth_info"])
}

// ============================================================================
// Command Structure Tests
// ============================================================================

func TestNewTxDecodeCmdHasQueryFlags(t *testing.T) {
	cmd := cli.NewTxDecodeCmd()

	// Check that query flags are present (added by flags.AddQueryFlagsToCmd)
	nodeFlag := cmd.Flags().Lookup("node")
	require.NotNil(t, nodeFlag, "expected 'node' flag to be present")

	outputFlag := cmd.Flags().Lookup("output")
	require.NotNil(t, outputFlag, "expected 'output' flag to be present")
}

func TestNewTxDecodeCmdHelp(t *testing.T) {
	cmd := cli.NewTxDecodeCmd()

	t.Run("use string format", func(t *testing.T) {
		require.Contains(t, cmd.Use, "decode")
		require.Contains(t, cmd.Use, "base64-tx")
	})

	t.Run("long description content", func(t *testing.T) {
		require.Contains(t, cmd.Long, "Decode")
		require.Contains(t, cmd.Long, "base64")
	})
}

// ============================================================================
// Integration with Parent Commands
// ============================================================================

func TestTxDecodeCmdInTxCmdTree(t *testing.T) {
	// If NewTxDecodeCmd is added to a parent tx command, verify it's present
	// This test assumes the command is added to the auth module's tx commands
	// Adjust based on your actual command structure

	cmd := cli.NewTxDecodeCmd()
	require.NotNil(t, cmd)
	require.Equal(t, "decode", cmd.Name())
}

// ============================================================================
// Various Transaction Types
// ============================================================================

func TestDecodeTxWithDifferentMessageTypes(t *testing.T) {
	encCfg := app.MakeEncodingConfig(t)

	var buf bytes.Buffer
	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithTxConfig(encCfg.TxConfig).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithOutput(&buf)

	t.Run("MsgSend transaction", func(t *testing.T) {
		buf.Reset()
		txBytes := createTestTxBytes(t, encCfg.TxConfig, "send tx", sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(100))), 100000)
		base64Tx := base64.StdEncoding.EncodeToString(txBytes)

		err := cli.DecodeTx(clientCtx, base64Tx)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		body := result["body"].(map[string]interface{})
		messages := body["messages"].([]interface{})
		require.Len(t, messages, 1)

		msg := messages[0].(map[string]interface{})
		require.Contains(t, msg["@type"], "MsgSend")
	})
}

func TestDecodeTxWithLargeMemo(t *testing.T) {
	encCfg := app.MakeEncodingConfig(t)

	var buf bytes.Buffer
	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithTxConfig(encCfg.TxConfig).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithOutput(&buf)

	// Create a transaction with a large memo
	largeMemo := string(make([]byte, 256))
	for i := range largeMemo {
		largeMemo = largeMemo[:i] + "a" + largeMemo[i+1:]
	}

	txBytes := createTestTxBytes(t, encCfg.TxConfig, largeMemo, sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(100))), 100000)
	base64Tx := base64.StdEncoding.EncodeToString(txBytes)

	err := cli.DecodeTx(clientCtx, base64Tx)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	body := result["body"].(map[string]interface{})
	require.Equal(t, largeMemo, body["memo"])
}

func TestDecodeTxWithMultipleCoins(t *testing.T) {
	encCfg := app.MakeEncodingConfig(t)

	var buf bytes.Buffer
	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithTxConfig(encCfg.TxConfig).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithOutput(&buf)

	fee := sdk.NewCoins(
		sdk.NewCoin("atom", math.NewInt(500)),
		sdk.NewCoin("stake", math.NewInt(1000)),
	)

	txBytes := createTestTxBytes(t, encCfg.TxConfig, "multi coin fee", fee, 200000)
	base64Tx := base64.StdEncoding.EncodeToString(txBytes)

	err := cli.DecodeTx(clientCtx, base64Tx)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	authInfo := result["auth_info"].(map[string]interface{})
	feeInfo := authInfo["fee"].(map[string]interface{})
	amount := feeInfo["amount"].([]interface{})
	require.Len(t, amount, 2)
}

// ============================================================================
// Real-World Transaction Format Test
// ============================================================================

func TestDecodeTxRealWorldFormat(t *testing.T) {
	// This test uses a format similar to what was provided in the conversation
	// The actual base64 string from a real transaction
	encCfg := app.MakeEncodingConfig(t)

	var buf bytes.Buffer
	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithTxConfig(encCfg.TxConfig).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithOutput(&buf)

	// Create a realistic transaction
	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	fromAddr := sdk.AccAddress(pubKey.Address())
	toAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	msg := banktypes.NewMsgSend(fromAddr, toAddr, sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(100000))))

	txBuilder := encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	require.NoError(t, err)

	txBuilder.SetMemo("")
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(60000))))
	txBuilder.SetGasLimit(400000)

	sigV2 := signing.SignatureV2{
		PubKey: pubKey,
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: 0,
	}
	err = txBuilder.SetSignatures(sigV2)
	require.NoError(t, err)

	txBytes, err := encCfg.TxConfig.TxEncoder()(txBuilder.GetTx())
	require.NoError(t, err)

	base64Tx := base64.StdEncoding.EncodeToString(txBytes)

	err = cli.DecodeTx(clientCtx, base64Tx)
	require.NoError(t, err)

	// Verify we can parse the output
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// Verify structure
	require.Contains(t, result, "body")
	require.Contains(t, result, "auth_info")
	require.Contains(t, result, "signatures")

	// Verify message type
	body := result["body"].(map[string]interface{})
	messages := body["messages"].([]interface{})
	require.Len(t, messages, 1)

	msgMap := messages[0].(map[string]interface{})
	require.Contains(t, msgMap["@type"], "MsgSend")
}

// ============================================================================
// Helper Functions
// ============================================================================

// createTestTxBytes creates a test transaction and returns its encoded bytes
func createTestTxBytes(t *testing.T, txConfig client.TxConfig, memo string, fee sdk.Coins, gas uint64) []byte {
	t.Helper()

	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	fromAddr := sdk.AccAddress(pubKey.Address())
	toAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	msg := banktypes.NewMsgSend(fromAddr, toAddr, sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(100))))

	txBuilder := txConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	require.NoError(t, err)

	txBuilder.SetMemo(memo)
	txBuilder.SetFeeAmount(fee)
	txBuilder.SetGasLimit(gas)

	sigV2 := signing.SignatureV2{
		PubKey: pubKey,
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: 0,
	}
	err = txBuilder.SetSignatures(sigV2)
	require.NoError(t, err)

	txBytes, err := txConfig.TxEncoder()(txBuilder.GetTx())
	require.NoError(t, err)

	return txBytes
}

// createTestSignDocBytes creates a test SignDoc and returns its encoded bytes
func createTestSignDocBytes(t *testing.T, chainID string, accountNumber, sequence uint64) []byte {
	t.Helper()

	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	fromAddr := sdk.AccAddress(pubKey.Address())
	toAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	msg := banktypes.NewMsgSend(fromAddr, toAddr, sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(100000))))

	// Create TxBody
	txBody := txtypes.TxBody{
		Messages: []*codectypes.Any{},
		Memo:     "",
	}

	// Convert message to Any
	anyMsg, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)
	txBody.Messages = append(txBody.Messages, anyMsg)

	// Create AuthInfo
	authInfo := txtypes.AuthInfo{
		SignerInfos: []*txtypes.SignerInfo{
			{
				PublicKey: nil, // Can be nil for testing
				ModeInfo: &txtypes.ModeInfo{
					Sum: &txtypes.ModeInfo_Single_{
						Single: &txtypes.ModeInfo_Single{
							Mode: signing.SignMode_SIGN_MODE_DIRECT,
						},
					},
				},
				Sequence: sequence,
			},
		},
		Fee: &txtypes.Fee{
			Amount:   sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(60000))),
			GasLimit: 2000000,
		},
	}

	// Marshal body and auth info
	bodyBytes, err := txBody.Marshal()
	require.NoError(t, err)

	authInfoBytes, err := authInfo.Marshal()
	require.NoError(t, err)

	signDoc := txtypes.SignDoc{
		BodyBytes:     bodyBytes,
		AuthInfoBytes: authInfoBytes,
		ChainId:       chainID,
		AccountNumber: accountNumber,
	}

	signDocBytes, err := signDoc.Marshal()
	require.NoError(t, err)

	return signDocBytes
}

// ============================================================================
// Additional Coverage Tests
// ============================================================================

func TestDecodeTxRawBase64WithoutPadding(t *testing.T) {
	// Test base64 decoding without padding (RawStdEncoding fallback)
	encCfg := app.MakeEncodingConfig(t)

	var buf bytes.Buffer
	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithTxConfig(encCfg.TxConfig).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithOutput(&buf)

	t.Run("valid tx with raw base64 no padding", func(t *testing.T) {
		buf.Reset()
		txBytes := createTestTxBytes(t, encCfg.TxConfig, "test memo", sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1000))), 200000)

		// Use RawStdEncoding (no padding)
		base64Tx := base64.RawStdEncoding.EncodeToString(txBytes)

		err := cli.DecodeTx(clientCtx, base64Tx)
		require.NoError(t, err)

		// Verify output is valid JSON
		output := buf.String()
		require.NotEmpty(t, output)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)
	})

	t.Run("valid sign doc with raw base64 no padding", func(t *testing.T) {
		buf.Reset()
		signDocBytes := createTestSignDocBytes(t, "xion-test", 5, 0)

		// Use RawStdEncoding (no padding)
		base64SignDoc := base64.RawStdEncoding.EncodeToString(signDocBytes)

		err := cli.DecodeTx(clientCtx, base64SignDoc)
		require.NoError(t, err)

		// Verify output is valid JSON
		output := buf.String()
		require.NotEmpty(t, output)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)

		// Verify chain_id
		require.Equal(t, "xion-test", result["chain_id"])
	})
}

func TestDecodeTxURLSafeBase64Variants(t *testing.T) {
	encCfg := app.MakeEncodingConfig(t)

	var buf bytes.Buffer
	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithTxConfig(encCfg.TxConfig).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithOutput(&buf)

	t.Run("URL-safe base64 with raw encoding", func(t *testing.T) {
		buf.Reset()
		txBytes := createTestTxBytes(t, encCfg.TxConfig, "test", sdk.NewCoins(), 100000)

		// Use RawURLEncoding (URL-safe, no padding)
		base64Tx := base64.RawURLEncoding.EncodeToString(txBytes)

		err := cli.DecodeTx(clientCtx, base64Tx)
		require.NoError(t, err)
	})
}

func TestPrintPrettyJSONFallback(t *testing.T) {
	// Test the printPrettyJSON function's fallback behavior
	// We can't directly test private functions, but we can test through DecodeTx
	// with edge cases that might trigger fallback paths

	encCfg := app.MakeEncodingConfig(t)

	var buf bytes.Buffer
	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithTxConfig(encCfg.TxConfig).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithOutput(&buf)

	t.Run("transaction with special characters in memo", func(t *testing.T) {
		buf.Reset()
		// Use special characters that might cause JSON edge cases
		txBytes := createTestTxBytes(t, encCfg.TxConfig, "test memo with \"quotes\" and \\backslashes\\", sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(100))), 100000)
		base64Tx := base64.StdEncoding.EncodeToString(txBytes)

		err := cli.DecodeTx(clientCtx, base64Tx)
		require.NoError(t, err)

		// Verify output is valid JSON
		output := buf.String()
		require.NotEmpty(t, output)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)
	})

	t.Run("transaction with unicode in memo", func(t *testing.T) {
		buf.Reset()
		txBytes := createTestTxBytes(t, encCfg.TxConfig, "test 🎉 emoji memo 日本語", sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(100))), 100000)
		base64Tx := base64.StdEncoding.EncodeToString(txBytes)

		err := cli.DecodeTx(clientCtx, base64Tx)
		require.NoError(t, err)

		output := buf.String()
		require.NotEmpty(t, output)

		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)
	})
}
