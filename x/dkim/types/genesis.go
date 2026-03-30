package types

import (
	"bytes"
	"fmt"
)

// d line is used by starport scaffolding # genesis/types/import

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultDkimPubKeys returns the default DKIM public keys for major email providers.
// These records are included in the default genesis state so they're available
// immediately when the chain initializes without requiring a governance proposal.
// Each record includes a pre-computed PoseidonHash of the public key.
func DefaultDkimPubKeys() []DkimPubKey {
	records := []DkimPubKey{
		{
			Domain:   "gmail.com",
			Selector: "20230601",
			PubKey:   "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAntvSKT1hkqhKe0xcaZ0x+QbouDsJuBfby/S82jxsoC/SodmfmVs2D1KAH3mi1AqdMdU12h2VfETeOJkgGYq5ljd996AJ7ud2SyOLQmlhaNHH7Lx+Mdab8/zDN1SdxPARDgcM7AsRECHwQ15R20FaKUABGu4NTbR2fDKnYwiq5jQyBkLWP+LgGOgfUF4T4HZb2PY2bQtEP6QeqOtcW4rrsH24L7XhD+HSZb1hsitrE0VPbhJzxDwI4JF815XMnSVjZgYUXP8CxI1Y0FONlqtQYgsorZ9apoW1KPQe8brSSlRsi9sXB/tu56LmG7tEDNmrZ5XUwQYUUADBOu7t1niwXwIDAQAB",
			Version:  Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:  KeyType_KEY_TYPE_RSA_UNSPECIFIED,
		},
		{
			Domain:   "icloud.com",
			Selector: "1a1hai",
			PubKey:   "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA1ZEfbkf4TbO2TDZI67WhJ6G8Dwk3SJyAbBlE/QKdyXFZB4HfEU7AcuZBzcXSJFE03DlmyOkUAmaaR8yFlwooHyaKRLIaT3epGlL5YGowyfItLly2k0Jj0IOICRxWrB378b7qMeimE8KlH1UNaVpRTTi0XIYjIKAOpTlBmkM9a/3Rl4NWy8pLYApXD+WCkYxPcxoAAgaN8osqGTCJ5r+VHFU7Wm9xqq3MZmnfo0bzInF4UajCKjJAQa+HNuh95DWIYP/wV77/PxkEakOtzkbJMlFJiK/hMJ+HQUvTbtKW2s+t4uDK8DI16Rotsn6e0hS8xuXPmVte9ZzplD0fQgm2qwIDAQAB",
			Version:  Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:  KeyType_KEY_TYPE_RSA_UNSPECIFIED,
		},
		{
			Domain:   "outlook.com",
			Selector: "selector1",
			PubKey:   "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvWyktrIL8DO/+UGvMbv7cPd/Xogpbs7pgVw8y9ldO6AAMmg8+ijENl/c7Fb1MfKM7uG3LMwAr0dVVKyM+mbkoX2k5L7lsROQr0Z9gGSpu7xrnZOa58+/pIhd2Xk/DFPpa5+TKbWodbsSZPRN8z0RY5x59jdzSclXlEyN9mEZdmOiKTsOP6A7vQxfSya9jg5N81dfNNvP7HnWejMMsKyIMrXptxOhIBuEYH67JDe98QgX14oHvGM2Uz53if/SW8MF09rYh9sp4ZsaWLIg6T343JzlbtrsGRGCDJ9JPpxRWZimtz+Up/BlKzT6sCCrBihb/Bi3pZiEBB4Ui/vruL5RCQIDAQAB",
			Version:  Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:  KeyType_KEY_TYPE_RSA_UNSPECIFIED,
		},
		{
			Domain:   "proton.me",
			Selector: "ck677gxvmnehzmitcrhii5zb3q.protonmail",
			PubKey:   "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAn46zCm3zHBmS1zePKxA+RIXw41Nu6l91NpLLBnWnrcZ35H/843XNWPZEQ0OgGwx/yqTXETMLXIDjGEWlK1E1mpdguqu+3s7SuIHoo5+i6mgyxJguljkwc3dk8ojnJ6VVUPnDh5GJArkAhXxEb1aOK1BVGM0yDlmYdmaOfd48qcx5iODP/MFc8pivfxEXTIL+aUz7+X69lMiwUSHpWYL3/a5X3nLD0zEntxv08xs8J/rpuRg4v+OXEOhcNvhkeiRZqJBdpJTkoEZfGvdTct+U0YYC69NW0ClUcKio2uDPmxU1xvfvHbSTW2gHYk8RpYZaxLACULdMo+Vt4Na/oIR+swIDAQAB",
			Version:  Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:  KeyType_KEY_TYPE_RSA_UNSPECIFIED,
		},
		{
			Domain:   "yahoo.com",
			Selector: "s1024",
			PubKey:   "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDrEee0Ri4Juz+QfiWYui/E9UGSXau/2P8LjnTD8V4Unn+2FAZVGE3kL23bzeoULYv4PeleB3gfmJiDJOKU3Ns5L4KJAUUHjFwDebt0NP+sBK0VKeTATL2Yr/S3bT/xhy+1xtj4RkdV7fVxTn56Lb4udUnwuxK4V5b5PdOKj/+XcwIDAQAB",
			Version:  Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:  KeyType_KEY_TYPE_RSA_UNSPECIFIED,
		},
		{
			Domain:   "fastmail.com",
			Selector: "fm2",
			PubKey:   "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvfZ2hWGUdZ5l9BjjJyPyKhPuN9dsjjGHk7p+64dfhtvxxaptrlsMTNprSwGbEgVI+GIp9dqXoqYNH9LF9KIkJUyVmbaQKo8Ho8BEBSpkiqIkW4AE7D1ppO6pLcqulAx2kszcvonKyu8KuQiVrLXsi+zFpVGgdO1E2qkCabJXBgc1qPaY/iWJB1SjXM60ERlEJd6MlWjfYHgK4qNguhVoHwKjCvFdCdAfk47VS/SCOKsX0WvXEcQCjuLAiF7RM+ZzUj1poaXsgzm2XFEUjTrgJv2OQvZr3FS5WYPvYlDeMUR0AeKITOajuK9dvIkEgLYYLd6rOwq5+yIXPB0jt8dpYQIDAQAB",
			Version:  Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:  KeyType_KEY_TYPE_RSA_UNSPECIFIED,
		},
	}

	// Pre-compute PoseidonHash for each record
	for i := range records {
		hash, err := ComputePoseidonHash(records[i].PubKey)
		if err != nil {
			panic("failed to compute poseidon hash for default DKIM record " + records[i].Domain + ": " + err.Error())
		}
		records[i].PoseidonHash = hash.Bytes()
	}

	return records
}

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// d line is used by starport scaffolding # genesis/types/default
		Params:      DefaultParams(),
		DkimPubkeys: DefaultDkimPubKeys(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	params := gs.Params
	if params.MaxPubkeySizeBytes == 0 {
		params.MaxPubkeySizeBytes = DefaultMaxPubKeySizeBytes
	}
	if params.MinRsaKeyBits == 0 {
		params.MinRsaKeyBits = DefaultMinRSAKeyBits
	}
	if err := params.Validate(); err != nil {
		return err
	}
	if err := ValidateDkimPubKeys(gs.DkimPubkeys, params); err != nil {
		return err
	}
	for _, revoked := range gs.RevokedPubkeys {
		pubKeyBytes, err := DecodePubKeyWithLimit(revoked, params.MaxPubkeySizeBytes)
		if err != nil {
			return err
		}
		// RevokedPubkeys entries may be either:
		//   - a 32-byte SHA-256 hash (produced by CanonicalizeRSAPublicKey), or
		//   - a full DER-encoded RSA public key (legacy direct-storage entries).
		// Attempting to ParseRSAPublicKey on a 32-byte hash always fails because
		// 32 bytes is not valid ASN.1, causing ValidateGenesis to reject any chain
		// export that contains revoked keys (SEC-653). Accept 32-byte entries as
		// valid SHA-256 hashes and only attempt RSA parsing for longer byte slices.
		if len(pubKeyBytes) == 32 {
			// Valid SHA-256 hash from CanonicalizeRSAPublicKey; no further parsing needed.
			continue
		}
		if _, err := ParseRSAPublicKey(pubKeyBytes); err != nil {
			return fmt.Errorf("invalid revoked pubkey: not a 32-byte sha256 hash and not a valid RSA public key: %w", err)
		}
	}
	return nil
}

//nolint:staticcheck // ST1016
func (d *DkimPubKey) Equal(v interface{}) bool {
	if v == nil {
		return d == nil
	}

	v1, ok := v.(*DkimPubKey)
	if !ok {
		v2, ok := v.(DkimPubKey)
		if ok {
			v1 = &v2
		} else {
			return false
		}
	}
	if v1 == nil {
		return d == nil
	} else if d == nil {
		return false
	}
	if d.Domain != v1.Domain {
		return false
	}
	if d.PubKey != v1.PubKey {
		return false
	}
	if d.Selector != v1.Selector {
		return false
	}
	if d.Version != v1.Version {
		return false
	}
	if d.KeyType != v1.KeyType {
		return false
	}
	if !bytes.Equal(d.PoseidonHash, v1.PoseidonHash) {
		return false
	}
	return true
}
