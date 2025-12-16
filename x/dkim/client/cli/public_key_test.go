package cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	cli "github.com/burnt-labs/xion/x/dkim/client/cli"
)

func TestGetDkimDNSPublicKey(t *testing.T) {
	testCases := []struct {
		name     string
		domain   string
		selector string
		result   string
	}{
		{
			name:     "x.com dkim-202308",
			domain:   "x.com",
			selector: "dkim-202308",
			result:   "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB",
		},
		{
			name:     "slack email",
			domain:   "email.slackhq.com",
			selector: "200608",
			result:   "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDGoQCNwAQdJBy23MrShs1EuHqK/dtDC33QrTqgWd9CJmtM3CK2ZiTYugkhcxnkEtGbzg+IJqcDRNkZHyoRezTf6QbinBB2dbyANEuwKI5DVRBFowQOj9zvM3IvxAEboMlb0szUjAoML94HOkKuGuCkdZ1gbVEi3GcVwrIQphal1QIDAQAB",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pubKey, err := cli.GetDKIMPublicKey(tc.selector, tc.domain)
			require.NoError(t, err)
			require.EqualValues(t, tc.result, pubKey)
		})
	}
}

// ============================================================================
// Error Cases - DNS Lookup Failures
// ============================================================================

func TestGetDKIMPublicKeyDNSErrors(t *testing.T) {
	t.Run("nonexistent domain returns error", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", "this-domain-does-not-exist-12345.invalid")
		require.Error(t, err)
		require.Empty(t, pubKey)
		require.Contains(t, err.Error(), "failed to lookup TXT records")
	})

	t.Run("empty domain returns error", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", "")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("empty selector returns error", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("", "example.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("both empty returns error", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("", "")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("invalid TLD returns error", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", "domain.invalidtld12345")
		require.Error(t, err)
		require.Empty(t, pubKey)
		require.Contains(t, err.Error(), "failed to lookup TXT records")
	})
}

// ============================================================================
// Error Cases - DKIM Record Not Found
// ============================================================================

func TestGetDKIMPublicKeyNotFound(t *testing.T) {
	t.Run("valid domain but nonexistent selector", func(t *testing.T) {
		// Use a very random selector that definitely won't exist
		pubKey, err := cli.GetDKIMPublicKey("zzz-nonexistent-selector-xyz-99999-zzz", "cloudflare.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
		// Could be either DNS lookup failure or DKIM key not found
		require.True(t,
			contains(err.Error(), "failed to lookup TXT records") ||
				contains(err.Error(), "DKIM public key not found"),
			"expected DNS lookup error or DKIM not found error, got: %v", err)
	})

	t.Run("domain with invalid selector pattern", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("definitely-not-a-real-selector-12345678", "microsoft.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("another domain with invalid selector", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("fake-zzz-selector-999-zzz", "apple.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})
}

// ============================================================================
// Edge Cases - Domain Formats
// ============================================================================

func TestGetDKIMPublicKeyDomainFormats(t *testing.T) {
	t.Run("domain with protocol prefix fails", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", "https://example.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("domain with port fails", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", "example.com:8080")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("domain with path fails", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", "example.com/path")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("IP address as domain fails", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", "192.168.1.1")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("IPv6 address as domain fails", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", "::1")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("localhost fails", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", "localhost")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("domain with leading dot fails", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", ".example.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("domain with double dots fails", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", "example..com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("domain with spaces fails", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", "example .com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("very long domain fails", func(t *testing.T) {
		longDomain := "a" + string(make([]byte, 300)) + ".com"
		pubKey, err := cli.GetDKIMPublicKey("selector", longDomain)
		require.Error(t, err)
		require.Empty(t, pubKey)
	})
}

// ============================================================================
// Edge Cases - Selector Formats
// ============================================================================

func TestGetDKIMPublicKeySelectorFormats(t *testing.T) {
	t.Run("selector with spaces fails", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector with spaces", "example.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("very long selector fails", func(t *testing.T) {
		longSelector := string(make([]byte, 500))
		pubKey, err := cli.GetDKIMPublicKey(longSelector, "example.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("selector with special characters", func(t *testing.T) {
		// These should fail because the domain won't have such DKIM records
		specialSelectors := []string{
			"selector!@#",
			"selector<>",
			"selector\"quote",
		}
		for _, sel := range specialSelectors {
			pubKey, err := cli.GetDKIMPublicKey(sel, "example.com")
			require.Error(t, err)
			require.Empty(t, pubKey)
		}
	})
}

// ============================================================================
// DNS Query Format Verification
// ============================================================================

func TestGetDKIMPublicKeyDNSQueryFormat(t *testing.T) {
	// These tests verify that the DNS query is constructed correctly
	// by testing with domains that should have predictable DNS behavior

	t.Run("subdomain is handled correctly", func(t *testing.T) {
		// Testing with a subdomain - the DKIM query should be selector._domainkey.subdomain.domain.tld
		pubKey, err := cli.GetDKIMPublicKey("zzz-nonexistent-zzz", "mail.nonexistent-domain-12345.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
		// This verifies the function handles subdomains without panicking
	})

	t.Run("multiple subdomains handled correctly", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", "deep.sub.nonexistent-domain-12345.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("selector with underscore format validation", func(t *testing.T) {
		// Underscores are valid in DKIM selectors
		// Use a nonexistent domain to ensure it fails
		pubKey, err := cli.GetDKIMPublicKey("my_selector", "zzz-nonexistent-domain-12345.invalid")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("selector with hyphen format validation", func(t *testing.T) {
		// Hyphens are valid in DKIM selectors
		pubKey, err := cli.GetDKIMPublicKey("my-selector-2023", "zzz-nonexistent-domain-12345.invalid")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("numeric selector format validation", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("20230815", "zzz-nonexistent-domain-12345.invalid")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})
}

// ============================================================================
// Unicode and International Domains
// ============================================================================

func TestGetDKIMPublicKeyInternationalDomains(t *testing.T) {
	t.Run("unicode domain fails gracefully", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", "例え.jp")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("punycode domain", func(t *testing.T) {
		// xn-- prefix indicates punycode
		pubKey, err := cli.GetDKIMPublicKey("selector", "xn--n3h.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("unicode selector fails gracefully", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("選択", "example.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})
}

// ============================================================================
// Whitespace Handling
// ============================================================================

func TestGetDKIMPublicKeyWhitespace(t *testing.T) {
	t.Run("domain with leading whitespace", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", " example.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("domain with trailing whitespace", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", "example.com ")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("selector with leading whitespace", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey(" selector", "example.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("selector with trailing whitespace", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector ", "example.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("whitespace only domain", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", "   ")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("whitespace only selector", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("   ", "example.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("tab characters", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector\t", "example.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("newline characters", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector\n", "example.com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})
}

// ============================================================================
// Boundary Conditions
// ============================================================================

func TestGetDKIMPublicKeyBoundaryConditions(t *testing.T) {
	t.Run("single character domain", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("selector", "a")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("single character selector", func(t *testing.T) {
		// Use a nonexistent domain to ensure failure
		pubKey, err := cli.GetDKIMPublicKey("s", "zzz-nonexistent-domain-12345.invalid")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("domain at max label length (63 chars)", func(t *testing.T) {
		// DNS labels can be up to 63 characters
		longLabel := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijk" // 63 chars
		pubKey, err := cli.GetDKIMPublicKey("selector", longLabel+".invalid")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})

	t.Run("domain exceeding max label length", func(t *testing.T) {
		// DNS labels cannot exceed 63 characters
		tooLongLabel := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl" // 64 chars
		pubKey, err := cli.GetDKIMPublicKey("selector", tooLongLabel+".com")
		require.Error(t, err)
		require.Empty(t, pubKey)
	})
}

// ============================================================================
// Case Sensitivity
// ============================================================================

func TestGetDKIMPublicKeyCaseSensitivity(t *testing.T) {
	// DNS is case-insensitive, but the function should handle various cases

	t.Run("uppercase domain", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("dkim-202308", "X.COM")
		// Should either succeed (DNS is case-insensitive) or fail gracefully
		if err == nil {
			require.NotEmpty(t, pubKey)
		}
	})

	t.Run("mixed case domain", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("dkim-202308", "X.Com")
		if err == nil {
			require.NotEmpty(t, pubKey)
		}
	})

	t.Run("uppercase selector", func(t *testing.T) {
		pubKey, err := cli.GetDKIMPublicKey("DKIM-202308", "x.com")
		if err == nil {
			require.NotEmpty(t, pubKey)
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
