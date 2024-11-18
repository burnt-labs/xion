package cli

import (
	"fmt"
	"net"
	"strings"
)

// getDKIMPublicKey retrieves the DKIM public key for the given selector and domain.
func GetDKIMPublicKey(selector, domain string) (string, error) {
	// Construct the DNS query name for the DKIM public key
	dkimDNSName := fmt.Sprintf("%s._domainkey.%s", selector, domain)

	// Perform a TXT lookup for the DKIM record
	txtRecords, err := net.LookupTXT(dkimDNSName)
	if err != nil {
		return "", fmt.Errorf("failed to lookup TXT records: %v", err)
	}

	// Extract and concatenate the TXT records, which contain the public key
	for _, record := range txtRecords {
		// Ensure we get the public key portion only
		if strings.Contains(record, "p=") {
			parts := strings.Split(record, "; ")
			for _, part := range parts {
				// Check if the part starts with "p="
				if strings.HasPrefix(part, "p=") {
					// Remove the "p=" prefix and return the public key value
					part = strings.TrimPrefix(part, "p=")
					// some DKIM records are trialing with ";"
					return strings.TrimSuffix(part, ";"), nil
				}
			}
		}
	}

	return "", fmt.Errorf("DKIM public key not found for selector %s", selector)
}
