package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// AudienceKeyPrefix is the prefix to retrieve all Audience
	AudienceKeyPrefix = "Audience/value/"
)

// AudienceKey returns the store key to retrieve a Audience from the index fields
func AudienceKey(
	aud string,
) []byte {
	var key []byte

	audBytes := []byte(aud)
	key = append(key, audBytes...)
	key = append(key, []byte("/")...)

	return key
}
