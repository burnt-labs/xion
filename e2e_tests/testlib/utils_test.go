package testlib

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeAutoCLIBinaryArg(t *testing.T) {
	for _, value := range []string{"0", "10", "multi-auth-salt", "salt with spaces", "\x00binary"} {
		t.Run(value, func(t *testing.T) {
			encoded := encodeAutoCLIBinaryArg(value)
			decoded, err := hex.DecodeString(encoded)
			require.NoError(t, err)
			require.Equal(t, []byte(value), decoded)
		})
	}
}
