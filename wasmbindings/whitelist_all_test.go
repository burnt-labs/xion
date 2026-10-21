package wasmbinding

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func WhitelistedQueryPaths() []string {
	var paths []string
	stargateWhitelist.Range(func(path, _ any) bool {
		paths = append(paths, path.(string))
		return true
	})
	return paths
}

func TestAllWhitelistedPathsEnsured(t *testing.T) {
	paths := WhitelistedQueryPaths()
	require.NotEmpty(t, paths)
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			msg, err := GetWhitelistedQuery(path)
			require.NoError(t, err)
			require.NotNil(t, msg)
		})
	}
}
