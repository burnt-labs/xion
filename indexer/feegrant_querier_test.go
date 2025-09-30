package indexer

import (
	"testing"

	"github.com/stretchr/testify/require"

	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"

	indexerfeegrant "github.com/burnt-labs/xion/indexer/feegrant"
)

// TestParseAllowanceRequestParams tests the allowance request parsing logic
// This is 100% testable without any pagination
func TestParseAllowanceRequestParams(t *testing.T) {
	addrCodec := addresscodec.NewBech32Codec("xion")

	// Create test addresses
	granter := sdk.AccAddress([]byte("granter_addr_test__"))
	grantee := sdk.AccAddress([]byte("grantee_addr_test__"))
	granterStr, err := addrCodec.BytesToString(granter)
	require.NoError(t, err)
	granteeStr, err := addrCodec.BytesToString(grantee)
	require.NoError(t, err)

	tests := []struct {
		name          string
		req           *indexerfeegrant.QueryAllowanceRequest
		expectGranter sdk.AccAddress
		expectGrantee sdk.AccAddress
		expectError   bool
	}{
		{
			name: "valid granter and grantee",
			req: &indexerfeegrant.QueryAllowanceRequest{
				Granter: granterStr,
				Grantee: granteeStr,
			},
			expectGranter: granter,
			expectGrantee: grantee,
			expectError:   false,
		},
		{
			name: "invalid granter address",
			req: &indexerfeegrant.QueryAllowanceRequest{
				Granter: "invalid_address",
				Grantee: granteeStr,
			},
			expectError: true,
		},
		{
			name: "invalid grantee address",
			req: &indexerfeegrant.QueryAllowanceRequest{
				Granter: granterStr,
				Grantee: "invalid_address",
			},
			expectError: true,
		},
		{
			name: "empty granter",
			req: &indexerfeegrant.QueryAllowanceRequest{
				Granter: "",
				Grantee: granteeStr,
			},
			expectError: true,
		},
		{
			name: "empty grantee",
			req: &indexerfeegrant.QueryAllowanceRequest{
				Granter: granterStr,
				Grantee: "",
			},
			expectError: true,
		},
		{
			name: "both empty",
			req: &indexerfeegrant.QueryAllowanceRequest{
				Granter: "",
				Grantee: "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultGranter, resultGrantee, err := ParseAllowanceRequestParams(tt.req, addrCodec)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectGranter, resultGranter)
			require.Equal(t, tt.expectGrantee, resultGrantee)
		})
	}
}

// TestParseAllowancesRequestParams tests the allowances request parsing logic
func TestParseAllowancesRequestParams(t *testing.T) {
	addrCodec := addresscodec.NewBech32Codec("xion")

	grantee := sdk.AccAddress([]byte("grantee_addr_test__"))
	granteeStr, err := addrCodec.BytesToString(grantee)
	require.NoError(t, err)

	tests := []struct {
		name          string
		req           *indexerfeegrant.QueryAllowancesRequest
		expectGrantee sdk.AccAddress
		expectError   bool
	}{
		{
			name: "valid grantee",
			req: &indexerfeegrant.QueryAllowancesRequest{
				Grantee: granteeStr,
			},
			expectGrantee: grantee,
			expectError:   false,
		},
		{
			name: "invalid grantee address",
			req: &indexerfeegrant.QueryAllowancesRequest{
				Grantee: "invalid_address",
			},
			expectError: true,
		},
		{
			name: "empty grantee",
			req: &indexerfeegrant.QueryAllowancesRequest{
				Grantee: "",
			},
			expectError: true,
		},
		{
			name: "malformed bech32",
			req: &indexerfeegrant.QueryAllowancesRequest{
				Grantee: "xion1invalidchecksum",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultGrantee, err := ParseAllowancesRequestParams(tt.req, addrCodec)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectGrantee, resultGrantee)
		})
	}
}

// TestParseAllowancesByGranterRequestParams tests the allowances by granter request parsing
func TestParseAllowancesByGranterRequestParams(t *testing.T) {
	addrCodec := addresscodec.NewBech32Codec("xion")

	granter := sdk.AccAddress([]byte("granter_addr_test__"))
	granterStr, err := addrCodec.BytesToString(granter)
	require.NoError(t, err)

	tests := []struct {
		name          string
		req           *indexerfeegrant.QueryAllowancesByGranterRequest
		expectGranter sdk.AccAddress
		expectError   bool
	}{
		{
			name: "valid granter",
			req: &indexerfeegrant.QueryAllowancesByGranterRequest{
				Granter: granterStr,
			},
			expectGranter: granter,
			expectError:   false,
		},
		{
			name: "invalid granter address",
			req: &indexerfeegrant.QueryAllowancesByGranterRequest{
				Granter: "invalid_address",
			},
			expectError: true,
		},
		{
			name: "empty granter",
			req: &indexerfeegrant.QueryAllowancesByGranterRequest{
				Granter: "",
			},
			expectError: true,
		},
		{
			name: "wrong prefix",
			req: &indexerfeegrant.QueryAllowancesByGranterRequest{
				Granter: "cosmos1test",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultGranter, err := ParseAllowancesByGranterRequestParams(tt.req, addrCodec)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectGranter, resultGranter)
		})
	}
}

// TestParseAllowanceRequestParamsAddressTypes tests different address byte lengths
func TestParseAllowanceRequestParamsAddressTypes(t *testing.T) {
	addrCodec := addresscodec.NewBech32Codec("xion")

	tests := []struct {
		name        string
		granterAddr sdk.AccAddress
		granteeAddr sdk.AccAddress
	}{
		{
			name:        "standard 20 byte addresses",
			granterAddr: sdk.AccAddress([]byte("granter_addr_test__")), // 19 bytes
			granteeAddr: sdk.AccAddress([]byte("grantee_addr_test__")), // 19 bytes
		},
		{
			name:        "minimal addresses",
			granterAddr: sdk.AccAddress([]byte("g")),
			granteeAddr: sdk.AccAddress([]byte("g")),
		},
		{
			name:        "different length addresses",
			granterAddr: sdk.AccAddress([]byte("short")),
			granteeAddr: sdk.AccAddress([]byte("much_longer_address_here")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			granterStr, err := addrCodec.BytesToString(tt.granterAddr)
			require.NoError(t, err)
			granteeStr, err := addrCodec.BytesToString(tt.granteeAddr)
			require.NoError(t, err)

			req := &indexerfeegrant.QueryAllowanceRequest{
				Granter: granterStr,
				Grantee: granteeStr,
			}

			resultGranter, resultGrantee, err := ParseAllowanceRequestParams(req, addrCodec)
			require.NoError(t, err)
			require.Equal(t, tt.granterAddr, resultGranter)
			require.Equal(t, tt.granteeAddr, resultGrantee)
		})
	}
}

// TestParseAllowanceRequestParamsConsistency tests that parsing is deterministic
func TestParseAllowanceRequestParamsConsistency(t *testing.T) {
	addrCodec := addresscodec.NewBech32Codec("xion")

	granter := sdk.AccAddress([]byte("granter_addr_test__"))
	grantee := sdk.AccAddress([]byte("grantee_addr_test__"))
	granterStr, _ := addrCodec.BytesToString(granter)
	granteeStr, _ := addrCodec.BytesToString(grantee)

	req := &indexerfeegrant.QueryAllowanceRequest{
		Granter: granterStr,
		Grantee: granteeStr,
	}

	// Parse multiple times
	results := make([][2]sdk.AccAddress, 10)
	for i := 0; i < 10; i++ {
		g1, g2, err := ParseAllowanceRequestParams(req, addrCodec)
		require.NoError(t, err)
		results[i] = [2]sdk.AccAddress{g1, g2}
	}

	// All results should be identical
	for i := 1; i < len(results); i++ {
		require.Equal(t, results[0], results[i], "Parsing should be deterministic")
	}
}

// Benchmark tests for performance
func BenchmarkParseAllowanceRequestParams(b *testing.B) {
	addrCodec := addresscodec.NewBech32Codec("xion")
	granter := sdk.AccAddress([]byte("granter_addr_test__"))
	grantee := sdk.AccAddress([]byte("grantee_addr_test__"))
	granterStr, _ := addrCodec.BytesToString(granter)
	granteeStr, _ := addrCodec.BytesToString(grantee)

	req := &indexerfeegrant.QueryAllowanceRequest{
		Granter: granterStr,
		Grantee: granteeStr,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = ParseAllowanceRequestParams(req, addrCodec)
	}
}

func BenchmarkParseAllowancesRequestParams(b *testing.B) {
	addrCodec := addresscodec.NewBech32Codec("xion")
	grantee := sdk.AccAddress([]byte("grantee_addr_test__"))
	granteeStr, _ := addrCodec.BytesToString(grantee)

	req := &indexerfeegrant.QueryAllowancesRequest{
		Grantee: granteeStr,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseAllowancesRequestParams(req, addrCodec)
	}
}

func BenchmarkParseAllowancesByGranterRequestParams(b *testing.B) {
	addrCodec := addresscodec.NewBech32Codec("xion")
	granter := sdk.AccAddress([]byte("granter_addr_test__"))
	granterStr, _ := addrCodec.BytesToString(granter)

	req := &indexerfeegrant.QueryAllowancesByGranterRequest{
		Granter: granterStr,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseAllowancesByGranterRequestParams(req, addrCodec)
	}
}
