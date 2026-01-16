package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestDecodePubKeyWithLimit(t *testing.T) {
	t.Run("within limit", func(t *testing.T) {
		data, err := types.DecodePubKeyWithLimit("Zm9v", 10)
		require.NoError(t, err)
		require.Equal(t, []byte("foo"), data)
	})

	t.Run("exceeds limit", func(t *testing.T) {
		_, err := types.DecodePubKeyWithLimit("Zm9v", 2)
		require.ErrorIs(t, err, types.ErrPubKeyTooLarge)
	})

	t.Run("invalid base64", func(t *testing.T) {
		_, err := types.DecodePubKeyWithLimit("!!", 10)
		require.ErrorIs(t, err, types.ErrInvalidPubKey)
	})
}
