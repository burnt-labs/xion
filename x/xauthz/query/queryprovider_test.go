package query_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/xauthz/query"
)

func TestNewProvider(t *testing.T) {
	t.Run("panics with nil keeper", func(t *testing.T) {
		require.Panics(t, func() {
			query.NewProvider(nil)
		})
	})
}
