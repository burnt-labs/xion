package types

import (
	"encoding/base64"

	errorsmod "cosmossdk.io/errors"
)

func DecodePubKey(pubKey string) ([]byte, error) {
	bz, err := base64.StdEncoding.DecodeString(pubKey)
	if err != nil {
		return nil, errorsmod.Wrap(ErrInvalidPubKey, err.Error())
	}

	return bz, nil
}

func DecodePubKeyWithLimit(pubKey string, maxDecodedSize uint64) ([]byte, error) {
	bz, err := base64.StdEncoding.DecodeString(pubKey)
	if err != nil {
		return nil, errorsmod.Wrap(ErrInvalidPubKey, err.Error())
	}

	if uint64(len(bz)) > maxDecodedSize {
		return nil, errorsmod.Wrapf(ErrPubKeyTooLarge, "decoded key size %d exceeds max %d", len(bz), maxDecodedSize)
	}

	return bz, nil
}
