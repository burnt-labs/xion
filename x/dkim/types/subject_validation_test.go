package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestValidateForcedSubject(t *testing.T) {
	testCases := []struct {
		name      string
		subject   string
		expected  bool
		errorDesc string
	}{
		// Valid cases - basic tag
		{
			name:      "valid - just tag",
			subject:   "[Reply Needed]",
			expected:  true,
			errorDesc: "subject with just the required tag should be valid",
		},
		{
			name:      "valid - tag with trailing content",
			subject:   "[Reply Needed] Some additional text",
			expected:  true,
			errorDesc: "subject with tag and trailing content should be valid",
		},
		{
			name:      "valid - tag with whitespace",
			subject:   "  [Reply Needed]  ",
			expected:  true,
			errorDesc: "subject with whitespace should be valid after trimming",
		},

		// Valid cases - Re: prefix variations
		{
			name:      "valid - Re: prefix",
			subject:   "Re: [Reply Needed]",
			expected:  true,
			errorDesc: "subject with Re: prefix should be valid",
		},
		{
			name:      "valid - RE: prefix",
			subject:   "RE: [Reply Needed]",
			expected:  true,
			errorDesc: "subject with RE: prefix should be valid",
		},
		{
			name:      "valid - re: prefix",
			subject:   "re: [Reply Needed]",
			expected:  true,
			errorDesc: "subject with re: prefix should be valid",
		},
		{
			name:      "valid - Re: prefix with trailing content",
			subject:   "Re: [Reply Needed] Please respond",
			expected:  true,
			errorDesc: "subject with Re: prefix and trailing content should be valid",
		},
		{
			name:      "valid - Re: prefix with multiple spaces",
			subject:   "Re:   [Reply Needed]",
			expected:  true,
			errorDesc: "subject with Re: prefix and multiple spaces should be valid",
		},

		// Valid cases - Fwd: prefix variations
		{
			name:      "valid - Fwd: prefix",
			subject:   "Fwd: [Reply Needed]",
			expected:  true,
			errorDesc: "subject with Fwd: prefix should be valid",
		},
		{
			name:      "valid - FWD: prefix",
			subject:   "FWD: [Reply Needed]",
			expected:  true,
			errorDesc: "subject with FWD: prefix should be valid",
		},
		{
			name:      "valid - fwd: prefix",
			subject:   "fwd: [Reply Needed]",
			expected:  true,
			errorDesc: "subject with fwd: prefix should be valid",
		},
		{
			name:      "valid - Fwd: prefix with trailing content",
			subject:   "Fwd: [Reply Needed] Forwarded message",
			expected:  true,
			errorDesc: "subject with Fwd: prefix and trailing content should be valid",
		},

		// Valid cases - multiple prefixes
		{
			name:      "valid - multiple Re: prefixes",
			subject:   "Re: RE: [Reply Needed]",
			expected:  true,
			errorDesc: "subject with multiple Re: prefixes should be valid",
		},
		{
			name:      "valid - Re: and Fwd: prefixes",
			subject:   "Re: Fwd: [Reply Needed]",
			expected:  true,
			errorDesc: "subject with Re: and Fwd: prefixes should be valid",
		},
		{
			name:      "valid - multiple mixed prefixes",
			subject:   "Re: RE: Fwd: [Reply Needed]",
			expected:  true,
			errorDesc: "subject with multiple mixed prefixes should be valid",
		},
		{
			name:      "valid - multiple prefixes with trailing content",
			subject:   "Re: Fwd: [Reply Needed] Important message",
			expected:  true,
			errorDesc: "subject with multiple prefixes and trailing content should be valid",
		},

		// Invalid cases - empty and missing tag
		{
			name:      "invalid - empty string",
			subject:   "",
			expected:  false,
			errorDesc: "empty subject should be invalid",
		},
		{
			name:      "invalid - whitespace only",
			subject:   "   ",
			expected:  false,
			errorDesc: "whitespace-only subject should be invalid",
		},
		{
			name:      "invalid - missing tag",
			subject:   "Some random subject",
			expected:  false,
			errorDesc: "subject without tag should be invalid",
		},
		{
			name:      "invalid - tag misspelled",
			subject:   "[Reply Need]",
			expected:  false,
			errorDesc: "subject with misspelled tag should be invalid",
		},
		{
			name:      "invalid - tag without brackets",
			subject:   "Reply Needed",
			expected:  false,
			errorDesc: "subject with tag without brackets should be invalid",
		},

		// Invalid cases - malicious patterns (header injection)
		{
			name:      "invalid - contains from:",
			subject:   "[Reply Needed] from: attacker@evil.com",
			expected:  false,
			errorDesc: "subject containing from: should be invalid",
		},
		{
			name:      "invalid - contains FROM:",
			subject:   "[Reply Needed] FROM: attacker@evil.com",
			expected:  false,
			errorDesc: "subject containing FROM: (case-insensitive) should be invalid",
		},
		{
			name:      "invalid - contains to:",
			subject:   "[Reply Needed] to: victim@example.com",
			expected:  false,
			errorDesc: "subject containing to: should be invalid",
		},
		{
			name:      "invalid - contains cc:",
			subject:   "[Reply Needed] cc: someone@example.com",
			expected:  false,
			errorDesc: "subject containing cc: should be invalid",
		},
		{
			name:      "invalid - contains bcc:",
			subject:   "[Reply Needed] bcc: hidden@example.com",
			expected:  false,
			errorDesc: "subject containing bcc: should be invalid",
		},
		{
			name:      "invalid - contains subject:",
			subject:   "[Reply Needed] subject: injected",
			expected:  false,
			errorDesc: "subject containing subject: should be invalid",
		},
		{
			name:      "invalid - contains reply-to:",
			subject:   "[Reply Needed] reply-to: attacker@evil.com",
			expected:  false,
			errorDesc: "subject containing reply-to: should be invalid",
		},
		{
			name:      "invalid - contains return-path:",
			subject:   "[Reply Needed] return-path: attacker@evil.com",
			expected:  false,
			errorDesc: "subject containing return-path: should be invalid",
		},
		{
			name:      "invalid - contains message-id:",
			subject:   "[Reply Needed] message-id: <fake@example.com>",
			expected:  false,
			errorDesc: "subject containing message-id: should be invalid",
		},
		{
			name:      "invalid - contains date:",
			subject:   "[Reply Needed] date: Mon, 1 Jan 2024",
			expected:  false,
			errorDesc: "subject containing date: should be invalid",
		},
		{
			name:      "invalid - contains content-type:",
			subject:   "[Reply Needed] content-type: text/html",
			expected:  false,
			errorDesc: "subject containing content-type: should be invalid",
		},
		{
			name:      "invalid - contains content-transfer-encoding:",
			subject:   "[Reply Needed] content-transfer-encoding: base64",
			expected:  false,
			errorDesc: "subject containing content-transfer-encoding: should be invalid",
		},
		{
			name:      "invalid - contains mime-version:",
			subject:   "[Reply Needed] mime-version: 1.0",
			expected:  false,
			errorDesc: "subject containing mime-version: should be invalid",
		},
		{
			name:      "invalid - contains received:",
			subject:   "[Reply Needed] received: from server",
			expected:  false,
			errorDesc: "subject containing received: should be invalid",
		},
		{
			name:      "invalid - contains x- prefix",
			subject:   "[Reply Needed] x-custom-header: value",
			expected:  false,
			errorDesc: "subject containing x- prefix should be invalid",
		},
		{
			name:      "invalid - contains X- prefix",
			subject:   "[Reply Needed] X-Custom-Header: value",
			expected:  false,
			errorDesc: "subject containing X- prefix (case-insensitive) should be invalid",
		},

		// Invalid cases - newline characters
		{
			name:      "invalid - contains newline",
			subject:   "[Reply Needed]\nfrom: attacker@evil.com",
			expected:  false,
			errorDesc: "subject containing newline should be invalid",
		},
		{
			name:      "invalid - contains carriage return",
			subject:   "[Reply Needed]\rfrom: attacker@evil.com",
			expected:  false,
			errorDesc: "subject containing carriage return should be invalid",
		},
		{
			name:      "invalid - contains CRLF",
			subject:   "[Reply Needed]\r\nfrom: attacker@evil.com",
			expected:  false,
			errorDesc: "subject containing CRLF should be invalid",
		},
		{
			name:      "valid - newline before tag (trimmed)",
			subject:   "\n[Reply Needed]",
			expected:  true,
			errorDesc: "subject with leading newline is trimmed and becomes valid",
		},

		// Invalid cases - invalid prefixes
		{
			name:      "invalid - invalid prefix before tag",
			subject:   "Invalid: [Reply Needed]",
			expected:  false,
			errorDesc: "subject with invalid prefix should be invalid",
		},
		{
			name:      "invalid - from: prefix",
			subject:   "from: [Reply Needed]",
			expected:  false,
			errorDesc: "subject with from: prefix should be invalid",
		},
		{
			name:      "invalid - to: prefix",
			subject:   "to: [Reply Needed]",
			expected:  false,
			errorDesc: "subject with to: prefix should be invalid",
		},
		{
			name:      "invalid - mixed valid and invalid prefix",
			subject:   "Re: Invalid: [Reply Needed]",
			expected:  false,
			errorDesc: "subject with valid and invalid prefix should be invalid",
		},
		{
			name:      "invalid - prefix without colon",
			subject:   "Re [Reply Needed]",
			expected:  false,
			errorDesc: "subject with prefix without colon should be invalid",
		},
		{
			name:      "invalid - prefix with extra text",
			subject:   "Re: extra [Reply Needed]",
			expected:  false,
			errorDesc: "subject with prefix and extra text before tag should be invalid",
		},

		// Invalid cases - tag position issues
		{
			name:      "invalid - tag at end with invalid prefix",
			subject:   "Invalid: text [Reply Needed]",
			expected:  false,
			errorDesc: "subject with invalid prefix and tag at end should be invalid",
		},
		{
			name:      "invalid - multiple tags but invalid content",
			subject:   "[Reply Needed] invalid: header",
			expected:  false,
			errorDesc: "subject with tag but invalid content after should be invalid",
		},

		// Edge cases - boundary conditions
		{
			name:      "invalid - only prefix without tag",
			subject:   "Re:",
			expected:  false,
			errorDesc: "subject with only prefix should be invalid",
		},
		{
			name:      "invalid - prefix with whitespace but no tag",
			subject:   "Re:   ",
			expected:  false,
			errorDesc: "subject with prefix and whitespace but no tag should be invalid",
		},
		{
			name:      "valid - prefix with minimal spacing",
			subject:   "Re:[Reply Needed]",
			expected:  true,
			errorDesc: "subject with prefix and minimal spacing should be valid (flexible whitespace)",
		},
		{
			name:      "valid - tag with special characters in trailing content",
			subject:   "[Reply Needed] @mention #hashtag $price",
			expected:  true,
			errorDesc: "subject with tag and special characters in trailing content should be valid",
		},
		{
			name:      "invalid - malicious pattern in prefix area",
			subject:   "from: [Reply Needed]",
			expected:  false,
			errorDesc: "subject with malicious pattern in prefix area should be invalid",
		},
		{
			name:      "invalid - malicious pattern embedded in text",
			subject:   "[Reply Needed] Please reply to: admin@example.com",
			expected:  false,
			errorDesc: "subject with malicious pattern embedded in text should be invalid",
		},
		{
			name:      "invalid - header-like pattern",
			subject:   "[Reply Needed] custom-header: value",
			expected:  false,
			errorDesc: "subject with header-like pattern should be invalid",
		},
		{
			name:      "invalid - header-like pattern uppercase",
			subject:   "[Reply Needed] Custom-Header: value",
			expected:  false,
			errorDesc: "subject with header-like pattern uppercase should be invalid",
		},
		{
			name:      "valid - text that looks like header but is not",
			subject:   "[Reply Needed] Please contact admin",
			expected:  true,
			errorDesc: "subject with normal text after tag should be valid",
		},
		{
			name:      "invalid - colon in middle of word",
			subject:   "[Reply Needed] ratio: 1:2",
			expected:  false,
			errorDesc: "subject with colon pattern that looks like header should be invalid",
		},
		{
			name:      "valid - email address in trailing content",
			subject:   "[Reply Needed] Contact admin@example.com",
			expected:  true,
			errorDesc: "subject with email address in trailing content should be valid",
		},
		{
			name:      "invalid - email with to: pattern",
			subject:   "[Reply Needed] Send to: admin@example.com",
			expected:  false,
			errorDesc: "subject with to: pattern in trailing content should be invalid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := types.ValidateForcedSubject(tc.subject)
			require.Equal(t, tc.expected, result, tc.errorDesc)
		})
	}
}

func TestValidateForcedSubject_EdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		subject  string
		expected bool
	}{
		{
			name:     "valid - very long trailing content",
			subject:  "[Reply Needed] " + string(make([]byte, 1000)),
			expected: true,
		},
		{
			name:     "valid - unicode characters",
			subject:  "[Reply Needed] 你好世界",
			expected: true,
		},
		{
			name:     "valid - emoji in trailing content",
			subject:  "[Reply Needed] 🚀 Important!",
			expected: true,
		},
		{
			name:     "invalid - tab character",
			subject:  "[Reply Needed]\tfrom: attacker",
			expected: false,
		},
		{
			name:     "valid - tab in trailing content (not newline)",
			subject:  "[Reply Needed]\tSome text",
			expected: true,
		},
		{
			name:     "invalid - null byte",
			subject:  "[Reply Needed]\x00from: attacker",
			expected: false,
		},
		{
			name:     "valid - multiple spaces between prefix and tag",
			subject:  "Re:     [Reply Needed]",
			expected: true,
		},
		{
			name:     "valid - prefix with no space before tag",
			subject:  "Re:[Reply Needed]",
			expected: true,
		},
		{
			name:     "invalid - prefix with text between prefix and tag",
			subject:  "Re: some text [Reply Needed]",
			expected: false,
		},
		{
			name:     "valid - tag in middle of long text",
			subject:  "Re: [Reply Needed] This is a very long message that continues for many words",
			expected: true,
		},
		{
			name:     "invalid - malicious pattern case variations",
			subject:  "[Reply Needed] FrOm: attacker",
			expected: false,
		},
		{
			name:     "invalid - malicious pattern with mixed case",
			subject:  "[Reply Needed] To: victim",
			expected: false,
		},
		{
			name:     "invalid - x- prefix with different casing",
			subject:  "[Reply Needed] X-Custom: value",
			expected: false,
		},
		{
			name:     "valid - re: and fwd: in different cases",
			subject:  "RE: fwd: [Reply Needed]",
			expected: true,
		},
		{
			name:     "valid - all uppercase prefixes",
			subject:  "RE: FWD: [Reply Needed]",
			expected: true,
		},
		{
			name:     "valid - all lowercase prefixes",
			subject:  "re: fwd: [Reply Needed]",
			expected: true,
		},
		{
			name:     "invalid - newline in middle",
			subject:  "[Reply Needed]\nSome text",
			expected: false,
		},
		{
			name:     "invalid - carriage return in middle",
			subject:  "[Reply Needed]\rSome text",
			expected: false,
		},
		{
			name:     "invalid - CRLF in middle",
			subject:  "[Reply Needed]\r\nSome text",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := types.ValidateForcedSubject(tc.subject)
			require.Equal(t, tc.expected, result, "subject: %q", tc.subject)
		})
	}
}

func TestValidateForcedSubject_MaliciousPatterns(t *testing.T) {
	maliciousPatterns := []string{
		"from:",
		"to:",
		"cc:",
		"bcc:",
		"subject:",
		"reply-to:",
		"return-path:",
		"message-id:",
		"date:",
		"content-type:",
		"content-transfer-encoding:",
		"mime-version:",
		"received:",
		"x-",
	}

	for _, pattern := range maliciousPatterns {
		t.Run("malicious pattern: "+pattern, func(t *testing.T) {
			// Test with pattern before tag
			subject1 := pattern + " [Reply Needed]"
			require.False(t, types.ValidateForcedSubject(subject1),
				"subject with malicious pattern before tag should be invalid: %q", subject1)

			// Test with pattern after tag
			subject2 := "[Reply Needed] " + pattern + " value"
			require.False(t, types.ValidateForcedSubject(subject2),
				"subject with malicious pattern after tag should be invalid: %q", subject2)

			// Test with pattern in middle
			subject3 := "[Reply Needed] text " + pattern + " value"
			require.False(t, types.ValidateForcedSubject(subject3),
				"subject with malicious pattern in middle should be invalid: %q", subject3)

			// Test case-insensitive
			patternUpper := pattern
			if len(pattern) > 0 {
				patternUpper = string([]byte{pattern[0] - 32}) + pattern[1:]
			}
			subject4 := "[Reply Needed] " + patternUpper + " value"
			require.False(t, types.ValidateForcedSubject(subject4),
				"subject with malicious pattern (uppercase) should be invalid: %q", subject4)
		})
	}
}

func TestValidateForcedSubject_PrefixCombinations(t *testing.T) {
	validPrefixes := []string{"Re:", "RE:", "re:", "Fwd:", "FWD:", "fwd:"}

	// Test all combinations of two prefixes
	for _, prefix1 := range validPrefixes {
		for _, prefix2 := range validPrefixes {
			t.Run("prefixes: "+prefix1+" "+prefix2, func(t *testing.T) {
				subject := prefix1 + " " + prefix2 + " [Reply Needed]"
				require.True(t, types.ValidateForcedSubject(subject),
					"subject with valid prefix combination should be valid: %q", subject)
			})
		}
	}

	// Test three prefixes
	t.Run("three prefixes", func(t *testing.T) {
		subject := "Re: Fwd: RE: [Reply Needed]"
		require.True(t, types.ValidateForcedSubject(subject),
			"subject with three valid prefixes should be valid: %q", subject)
	})

	// Test many prefixes
	t.Run("many prefixes", func(t *testing.T) {
		subject := "Re: RE: Fwd: FWD: re: fwd: [Reply Needed]"
		require.True(t, types.ValidateForcedSubject(subject),
			"subject with many valid prefixes should be valid: %q", subject)
	})
}

func TestValidateForcedSubject_MIMEEncoded(t *testing.T) {
	testCases := []struct {
		name      string
		subject   string
		expected  bool
		errorDesc string
	}{
		// Valid MIME-encoded subjects
		{
			name:      "valid - MIME encoded with utf-8 base64",
			subject:   "=?utf-8?B?W1JlcGx5IE5lZWRlZF0=?=",
			expected:  true,
			errorDesc: "MIME encoded subject with utf-8 base64 should decode and validate",
		},
		{
			name:      "valid - MIME encoded with trailing content",
			subject:   "=?utf-8?B?W1JlcGx5IE5lZWRlZF0gQ29tbWFuZCBDb25maXJtYXRpb24gUmVxdWlyZWQg?=",
			expected:  true,
			errorDesc: "MIME encoded subject with utf-8 base64 and trailing content should decode and validate",
		},
		{
			name:      "valid - MIME encoded with Re: prefix",
			subject:   "=?utf-8?B?UmU6IFtSZXBseSBOZWVkZWRd?=",
			expected:  true,
			errorDesc: "MIME encoded subject with Re: prefix should decode and validate",
		},
		{
			name:      "valid - multiple MIME encoded parts",
			subject:   "=?utf-8?B?W1JlcGx5IE5lZWRlZF0=?= =?utf-8?B?Q29tbWFuZA==?=",
			expected:  true,
			errorDesc: "Multiple MIME encoded parts should decode and validate",
		},
		{
			name:      "valid - MIME encoded with case variations",
			subject:   "=?UTF-8?B?W1JlcGx5IE5lZWRlZF0=?=",
			expected:  true,
			errorDesc: "MIME encoded subject with uppercase charset should work",
		},
		{
			name:      "valid - MIME encoded with lowercase encoding",
			subject:   "=?utf-8?b?W1JlcGx5IE5lZWRlZF0=?=",
			expected:  true,
			errorDesc: "MIME encoded subject with lowercase encoding should work",
		},
		{
			name:      "valid - MIME encoded with newline separator",
			subject:   "=?utf-8?B?W1JlcGx5IE5lZWRlZF0gQ29tbWFuZCBDb25maXJtYXRpb24gUmVxdWlyZWQg?=\n =?utf-8?B?W3dzdXJramV1dHplamVid2k0Y3dnbXFd?=",
			expected:  true,
			errorDesc: "MIME encoded subject with newline separator should decode and validate",
		},
		{
			name:      "valid - MIME encoded with whitespace separator",
			subject:   "=?utf-8?B?W1JlcGx5IE5lZWRlZF0=?=  =?utf-8?B?Q29tbWFuZA==?=",
			expected:  true,
			errorDesc: "MIME encoded subject with multiple spaces separator should decode and validate",
		},

		// Invalid MIME-encoded subjects - wrong charset
		{
			name:      "invalid - MIME encoded with wrong charset",
			subject:   "=?iso-8859-1?B?W1JlcGx5IE5lZWRlZF0=?=",
			expected:  false,
			errorDesc: "MIME encoded subject with non-utf-8 charset should be invalid",
		},
		{
			name:      "invalid - MIME encoded with unknown charset",
			subject:   "=?ascii?B?W1JlcGx5IE5lZWRlZF0=?=",
			expected:  false,
			errorDesc: "MIME encoded subject with unknown charset should be invalid",
		},

		// Invalid MIME-encoded subjects - wrong encoding
		{
			name:      "invalid - MIME encoded with quoted-printable",
			subject:   "=?utf-8?Q?=5BReply=20Needed=5D?=",
			expected:  false,
			errorDesc: "MIME encoded subject with quoted-printable encoding should be invalid",
		},
		{
			name:      "invalid - MIME encoded with lowercase q encoding",
			subject:   "=?utf-8?q?test?=",
			expected:  false,
			errorDesc: "MIME encoded subject with lowercase q encoding should be invalid",
		},

		// Invalid MIME-encoded subjects - malformed
		{
			name:      "invalid - MIME encoded with invalid base64",
			subject:   "=?utf-8?B?InvalidBase64!!!?=",
			expected:  false,
			errorDesc: "MIME encoded subject with invalid base64 should be invalid",
		},
		{
			name:      "invalid - MIME encoded with missing parts",
			subject:   "=?utf-8?B?",
			expected:  false,
			errorDesc: "MIME encoded subject with missing parts should be invalid",
		},
		{
			name:      "invalid - MIME encoded with incomplete format",
			subject:   "=?utf-8?B?W1JlcGx5IE5lZWRlZF0=",
			expected:  false,
			errorDesc: "MIME encoded subject with incomplete format should be invalid",
		},
		{
			name:      "invalid - MIME encoded with mixed valid and invalid parts",
			subject:   "=?utf-8?B?W1JlcGx5IE5lZWRlZF0=?= =?iso-8859-1?B?Q29tbWFuZA==?=",
			expected:  false,
			errorDesc: "MIME encoded subject with mixed valid and invalid charset should be invalid",
		},
		{
			name:      "invalid - MIME encoded with mixed valid and invalid encoding",
			subject:   "=?utf-8?B?W1JlcGx5IE5lZWRlZF0=?= =?utf-8?Q?Q29tbWFuZA==?=",
			expected:  false,
			errorDesc: "MIME encoded subject with mixed valid and invalid encoding should be invalid",
		},

		// Edge cases - MIME markers in plain text
		{
			name:      "invalid - plain text with MIME-like pattern",
			subject:   "[Reply Needed] =?utf-8?B?test?=",
			expected:  false,
			errorDesc: "Plain text with MIME markers should fail if format is invalid",
		},
		{
			name:      "valid - plain text without MIME markers",
			subject:   "[Reply Needed]",
			expected:  true,
			errorDesc: "Plain text without MIME markers should validate normally",
		},
		{
			name:      "invalid - MIME encoded but missing tag after decode",
			subject:   "=?utf-8?B?U29tZSB0ZXh0?=",
			expected:  false,
			errorDesc: "MIME encoded subject without tag after decode should be invalid",
		},
		{
			name:      "valid - user example with two MIME parts",
			subject:   "=?utf-8?B?W1JlcGx5IE5lZWRlZF0gQ29tbWFuZCBDb25maXJtYXRpb24gUmVxdWlyZWQg?=\n =?utf-8?B?W3dzdXJramV1dHplamVid2k0Y3dnbXFd?=",
			expected:  true,
			errorDesc: "User's example with two MIME encoded parts separated by newline should decode and validate",
		},
		{
			name:      "valid - user example with two MIME parts (single line)",
			subject:   " =?utf-8?B?W1JlcGx5IE5lZWRlZF0gQ29tbWFuZCBDb25maXJtYXRpb24gUmVxdWlyZWQg?= =?utf-8?B?W3dzdXJramV1dHplamVid2k0Y3dnbXFd?=",
			expected:  true,
			errorDesc: "User's example with two MIME encoded parts on single line should decode and validate",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := types.ValidateForcedSubject(tc.subject)
			require.Equal(t, tc.expected, result, tc.errorDesc)
		})
	}
}
