package types

import (
	"encoding/base64"
	"math"
	"strings"

	errorsmod "cosmossdk.io/errors"
)

func DecodePubKey(pubKey string) ([]byte, error) {
	if err := rejectBase64Whitespace(pubKey); err != nil {
		return nil, err
	}

	bz, err := base64.StdEncoding.DecodeString(pubKey)
	if err != nil {
		return nil, errorsmod.Wrap(ErrInvalidPubKey, err.Error())
	}

	return bz, nil
}

func DecodePubKeyWithLimit(pubKey string, maxDecodedSize uint64) ([]byte, error) {
	if err := rejectBase64Whitespace(pubKey); err != nil {
		return nil, err
	}

	maxEncodedLen, err := maxBase64EncodedLen(maxDecodedSize)
	if err != nil {
		return nil, err
	}

	if len(pubKey) > maxEncodedLen {
		return nil, errorsmod.Wrapf(ErrPubKeyTooLarge, "encoded key length %d exceeds max %d", len(pubKey), maxEncodedLen)
	}

	bz, err := base64.StdEncoding.DecodeString(pubKey)
	if err != nil {
		return nil, errorsmod.Wrap(ErrInvalidPubKey, err.Error())
	}

	if uint64(len(bz)) > maxDecodedSize {
		return nil, errorsmod.Wrapf(ErrPubKeyTooLarge, "decoded key size %d exceeds max %d", len(bz), maxDecodedSize)
	}

	return bz, nil
}

func maxBase64EncodedLen(maxDecodedSize uint64) (int, error) {
	// EncodedLen requires an int; guard against overflow even though module params are small.
	if maxDecodedSize > math.MaxInt {
		return 0, errorsmod.Wrapf(ErrPubKeyTooLarge, "max_pubkey_size_bytes %d exceeds supported range", maxDecodedSize)
	}

	return base64.StdEncoding.EncodedLen(int(maxDecodedSize)), nil
}

func rejectBase64Whitespace(pubKey string) error {
	if strings.IndexFunc(pubKey, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\r' || r == '\t'
	}) != -1 {
		return errorsmod.Wrap(ErrInvalidPubKey, "base64 key contains whitespace")
	}

	return nil
}
