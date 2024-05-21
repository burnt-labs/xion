package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// AudienceKeyPrefix is the prefix to retrieve all Audience
	AudienceKeyPrefix = "Audience/value/"

	// AudienceClaimKeyPrefix is the prefix for audience claims
	AudienceClaimKeyPrefix = "AudienceClaim/value/"
)

// AudienceKey returns the store key to retrieve an Audience from the index fields
func AudienceKey(
	aud string,
) []byte {
	var key []byte

	audBytes := []byte(aud)
	key = append(key, audBytes...)
	key = append(key, []byte("/")...)

	return key
}

func AudienceClaimKey(hash []byte) []byte {
	var key []byte

	key = append(key, hash...)
	key = append(key, []byte("/")...)

	return key
}
