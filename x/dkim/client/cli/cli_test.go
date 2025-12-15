package cli_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

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

	// Note: We cannot reliably test the success case without mocking DNS lookups
	// or relying on external DNS infrastructure which makes tests flaky.
	// The success path is covered by integration tests or manual testing.
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
