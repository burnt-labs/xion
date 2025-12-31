package types

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"
)

const (
	DefaultMaxPubKeySizeBytes uint64 = 1024
	// DefaultUploadChunkSize defines the default tier size (bytes) used to scale gas for pubkey uploads.
	DefaultUploadChunkSize uint64 = 20
	// DefaultUploadChunkGas defines the default gas cost per upload chunk.
	DefaultUploadChunkGas uint64 = 10_000
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
		MaxPubkeySizeBytes: DefaultMaxPubKeySizeBytes,
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
	if p.MaxPubkeySizeBytes == 0 {
		return errorsmod.Wrap(ErrInvalidParams, "max_pubkey_size_bytes must be positive")
	}

	for _, pubkey := range p.DkimPubkeys {
		if err := pubkey.Validate(); err != nil {
			return err
		}
		if err := ValidatePubKeySize(pubkey.PubKey, p.MaxPubkeySizeBytes); err != nil {
			return err
		}
	}
	return nil
}

func ValidatePubKeySize(pubKey string, maxSize uint64) error {
	_, err := DecodePubKeyWithLimit(pubKey, maxSize)
	return err
}

// GasCostForSize returns the gas cost for storing a payload of the given size.
func (p Params) GasCostForSize(size uint64) (uint64, error) {
	if err := p.Validate(); err != nil {
		return 0, err
	}

	if size == 0 {
		return 0, errorsmod.Wrap(ErrInvalidPubKey, "pubkey cannot be empty")
	}

	if size > p.MaxPubkeySizeBytes {
		return 0, errorsmod.Wrapf(ErrPubKeyTooLarge, "size %d > max %d", size, p.MaxPubkeySizeBytes)
	}

	chunks := (size + DefaultUploadChunkSize - 1) / DefaultUploadChunkSize
	return chunks * DefaultUploadChunkGas, nil
}
