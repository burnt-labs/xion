package cli_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/burnt-labs/xion/x/dkim/client/cli"
	"github.com/burnt-labs/xion/x/dkim/types"
)

// Mock QueryClient for testing
type mockQueryClient struct {
	types.QueryClient
	paramsFunc      func(ctx context.Context, req *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error)
	dkimPubKeyFunc  func(ctx context.Context, req *types.QueryDkimPubKeyRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeyResponse, error)
	dkimPubKeysFunc func(ctx context.Context, req *types.QueryDkimPubKeysRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeysResponse, error)
}

func (m *mockQueryClient) Params(ctx context.Context, req *types.QueryParamsRequest, opts ...grpc.CallOption) (*types.QueryParamsResponse, error) {
	if m.paramsFunc != nil {
		return m.paramsFunc(ctx, req, opts...)
	}
	params := types.DefaultParams()
	return &types.QueryParamsResponse{Params: &params}, nil
}

func (m *mockQueryClient) DkimPubKey(ctx context.Context, req *types.QueryDkimPubKeyRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeyResponse, error) {
	if m.dkimPubKeyFunc != nil {
		return m.dkimPubKeyFunc(ctx, req, opts...)
	}
	return &types.QueryDkimPubKeyResponse{
		DkimPubKey: &types.DkimPubKey{
			Domain:   req.Domain,
			Selector: req.Selector,
		},
	}, nil
}

func (m *mockQueryClient) DkimPubKeys(ctx context.Context, req *types.QueryDkimPubKeysRequest, opts ...grpc.CallOption) (*types.QueryDkimPubKeysResponse, error) {
	if m.dkimPubKeysFunc != nil {
		return m.dkimPubKeysFunc(ctx, req, opts...)
	}
	return &types.QueryDkimPubKeysResponse{
		DkimPubKeys: []*types.DkimPubKey{
			{
				Domain:       req.Domain,
				Selector:     req.Selector,
				PoseidonHash: req.PoseidonHash,
			},
		},
	}, nil
}

func TestQueryParams(t *testing.T) {
	mockClient := &mockQueryClient{}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	res, err := cli.QueryParams(mockClient, cmd)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Params)
}

func TestQueryDkimPubKey(t *testing.T) {
	mockClient := &mockQueryClient{}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	t.Run("successful query", func(t *testing.T) {
		res, err := cli.QueryDkimPubKey(mockClient, cmd, "example.com", "default")
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, "example.com", res.DkimPubKey.Domain)
		require.Equal(t, "default", res.DkimPubKey.Selector)
	})

	t.Run("with different domain and selector", func(t *testing.T) {
		res, err := cli.QueryDkimPubKey(mockClient, cmd, "test.org", "selector123")
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, "test.org", res.DkimPubKey.Domain)
		require.Equal(t, "selector123", res.DkimPubKey.Selector)
	})
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
}

func TestQueryDkimPubKeys(t *testing.T) {
	mockClient := &mockQueryClient{}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	t.Run("with all filters", func(t *testing.T) {
		res, err := cli.QueryDkimPubKeys(mockClient, cmd, "example.com", "default", "hash123")
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.DkimPubKeys, 1)
		require.Equal(t, "example.com", res.DkimPubKeys[0].Domain)
		require.Equal(t, "default", res.DkimPubKeys[0].Selector)
	})

	t.Run("with no filters", func(t *testing.T) {
		res, err := cli.QueryDkimPubKeys(mockClient, cmd, "", "", "")
		require.NoError(t, err)
		require.NotNil(t, res)
	})
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
