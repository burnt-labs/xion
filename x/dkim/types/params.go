package types

import (
	"encoding/json"
)

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	vkeyIdentifier := uint64(1)
	dkimDomain := "gmail.com"
	dkimSelector := "20230601"
	dkimPubkey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAntvSKT1hkqhKe0xcaZ0x+QbouDsJuBfby/S82jxsoC/SodmfmVs2D1KAH3mi1AqdMdU12h2VfETeOJkgGYq5ljd996AJ7ud2SyOLQmlhaNHH7Lx+Mdab8/zDN1SdxPARDgcM7AsRECHwQ15R20FaKUABGu4NTbR2fDKnYwiq5jQyBkLWP+LgGOgfUF4T4HZb2PY2bQtEP6QeqOtcW4rrsH24L7XhD+HSZb1hsitrE0VPbhJzxDwI4JF815XMnSVjZgYUXP8CxI1Y0FONlqtQYgsorZ9apoW1KPQe8brSSlRsi9sXB/tu56LmG7tEDNmrZ5XUwQYUUADBOu7t1niwXwIDAQAB"
	gPubKeyHash, err := ComputePoseidonHash(dkimPubkey)
	if err != nil {
		panic(err)
	}

	return Params{
		VkeyIdentifier: vkeyIdentifier,
		DkimPubkeys: []DkimPubKey{{
			Domain:       dkimDomain,
			Selector:     dkimSelector,
			PubKey:       dkimPubkey,
			PoseidonHash: gPubKeyHash.Bytes(), // []byte(gPubKeyHash)
		}},
	}
}

// Stringer method for Params.
func (p Params) String() string {
	bz, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	return string(bz)
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	for _, pubkey := range p.DkimPubkeys {
		if err := pubkey.Validate(); err != nil {
			return err
		}
	}
	return nil
}
