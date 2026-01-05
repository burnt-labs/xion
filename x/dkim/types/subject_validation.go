package types

import (
	"encoding/base64"
	"regexp"
	"strings"
)

// maliciousPatterns contains patterns for potentially malicious strings
// that could be used for email header injection attacks.
// This addresses security issue #2 from the ZKEmail audit report.
var maliciousPatterns = []string{
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

// containsMaliciousStrings checks if the subject contains any malicious patterns
// that could be used for email header injection attacks.
//
// This function addresses security issue #2 from the audit report by:
//  1. Rejecting subjects containing email header field names (case-insensitive)
//  2. Rejecting subjects containing newline characters that could split headers
//  3. Preventing header injection attacks where an attacker could manipulate
//     the "from" address or other critical email headers
//
// Returns true if malicious content is detected, false otherwise.
func containsMaliciousStrings(subject string) bool {
	subjectLower := strings.ToLower(subject)

	// Check for newline characters that could be used for header injection
	if strings.Contains(subject, "\r") || strings.Contains(subject, "\n") {
		return true
	}

	// Check for email header field names (case-insensitive)
	// These could be used to inject additional headers or manipulate the email
	for _, pattern := range maliciousPatterns {
		if strings.Contains(subjectLower, pattern) {
			return true
		}
	}

	// Additional check: look for patterns that could be header-like
	// Pattern: word followed by colon (potential header field)
	headerPattern := regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9-]*\s*:`)
	matches := headerPattern.FindAllString(subject, -1)
	for _, match := range matches {
		matchLower := strings.ToLower(strings.TrimSpace(match))
		// Allow only the valid email prefixes (re:, fwd:)
		if matchLower != "re:" && matchLower != "fwd:" {
			return true
		}
	}

	return false
}

// decodeMIMESubject decodes RFC 2047 MIME-encoded email subject strings.
//
// Format: =?charset?encoding?encoded-text?=
// - charset: must be "utf-8" (case-insensitive)
// - encoding: must be "B" for base64 (case-insensitive)
// - encoded-text: base64 encoded content
//
// Multiple encoded parts can be separated by whitespace.
//
// Returns the decoded string and true if successful, or empty string and false if:
// - The string is not MIME encoded (plain string)
// - The charset is not utf-8
// - The encoding is not B (base64)
// - The format is invalid
func decodeMIMESubject(encoded string) (string, bool) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return "", false
	}

	// Pattern to match MIME encoded parts: =?charset?encoding?text?=
	// The pattern matches: =? followed by charset, ? encoding, ? text, ?=
	mimePattern := regexp.MustCompile(`=\?([^?]+)\?([BbQq])\?([^?]+)\?=`)
	matches := mimePattern.FindAllStringSubmatch(encoded, -1)

	// If no matches found, it's not MIME encoded (plain string)
	if len(matches) == 0 {
		return "", false
	}

	var decodedParts []string

	// Process each MIME encoded part
	for _, match := range matches {
		if len(match) != 4 {
			return "", false
		}

		charset := strings.ToLower(strings.TrimSpace(match[1]))
		encoding := strings.ToUpper(strings.TrimSpace(match[2]))
		encodedText := match[3]

		// Only support utf-8 charset
		if charset != "utf-8" {
			return "", false
		}

		// Only support base64 encoding (B)
		if encoding != "B" {
			return "", false
		}

		// Decode base64
		decodedBytes, err := base64.StdEncoding.DecodeString(encodedText)
		if err != nil {
			// Try with padding if it fails
			decodedBytes, err = base64.StdEncoding.DecodeString(encodedText + "==")
			if err != nil {
				return "", false
			}
		}

		decodedParts = append(decodedParts, string(decodedBytes))
	}

	// Join all decoded parts with spaces (MIME parts are typically separated by whitespace)
	return strings.Join(decodedParts, " "), true
}

// ValidateForcedSubject is a more explicit version that validates each component
// separately for better debugging and clarity.
//
// This function handles both plain strings and RFC 2047 MIME-encoded strings.
// For MIME-encoded strings, it only supports utf-8 charset with base64 encoding.
// If the subject is MIME-encoded but uses unsupported charset or encoding, it returns false.
//
// Security: This function validates against malicious strings (e.g., "from:", "to:", etc.)
// and newline characters that could be used for email header injection attacks,
// addressing security issue #2 from the audit report.
func ValidateForcedSubject(subject string) bool {
	subject = strings.TrimSpace(subject)

	if subject == "" {
		return false
	}

	// Try to decode MIME-encoded subject
	decoded, isMIME := decodeMIMESubject(subject)
	if isMIME {
		// If it's MIME encoded, use the decoded version
		subject = decoded
	} else if strings.Contains(subject, "=?") {
		// If it contains MIME markers but failed to decode, it's invalid format
		// This handles cases like =?utf-8?Q?...?= (quoted-printable) or =?iso-8859-1?B?...?= (wrong charset)
		return false
	}
	// Otherwise, it's a plain string, continue with validation

	subject = strings.TrimSpace(subject)
	if subject == "" {
		return false
	}

	// Security check: reject subjects containing malicious strings (audit issue #2)
	if containsMaliciousStrings(subject) {
		return false
	}

	// After security validation, trim line endings for pattern matching
	subject = strings.TrimRight(subject, "\r\n")

	// Check if the required tag exists
	if !strings.Contains(subject, "[Reply Needed]") {
		return false
	}

	// Find the position of the tag
	tagIndex := strings.Index(subject, "[Reply Needed]")
	if tagIndex == -1 {
		return false
	}

	// Extract the prefix part (everything before the tag)
	prefix := strings.TrimSpace(subject[:tagIndex])

	// If there's a prefix, validate it consists only of valid email prefixes
	if prefix != "" {
		// Remove all valid prefixes and check if anything remains
		remaining := prefix

		validPrefixes := []string{"Re:", "RE:", "re:", "Fwd:", "FWD:", "fwd:"}

		for {
			removed := false
			for _, p := range validPrefixes {
				// Case-insensitive check
				if len(remaining) >= len(p) && strings.EqualFold(remaining[:len(p)], p) {
					remaining = strings.TrimSpace(remaining[len(p):])
					removed = true
					break
				}
			}
			if !removed {
				break
			}
		}

		// If anything remains after removing all valid prefixes, it's invalid
		if remaining != "" {
			return false
		}
	}

	// The tag must be present and there can be optional trailing content
	return true
}
