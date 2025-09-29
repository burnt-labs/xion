package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModuleConstants(t *testing.T) {
	require.Equal(t, "jwk", ModuleName)
	require.Equal(t, "jwk", StoreKey)
	require.Equal(t, "jwk", RouterKey)
}

func TestConstantsAreConsistent(t *testing.T) {
	// Ensure all constants are consistent with each other
	require.Equal(t, ModuleName, StoreKey)
	require.Equal(t, ModuleName, RouterKey)
}

func TestKeyPrefix(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		expected []byte
	}{
		{
			name:     "empty string",
			prefix:   "",
			expected: []byte(""),
		},
		{
			name:     "simple prefix",
			prefix:   "test",
			expected: []byte("test"),
		},
		{
			name:     "prefix with slash",
			prefix:   "audience/",
			expected: []byte("audience/"),
		},
		{
			name:     "complex prefix",
			prefix:   "audience/claim/",
			expected: []byte("audience/claim/"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := KeyPrefix(tt.prefix)
			require.Equal(t, tt.expected, result)
		})
	}
}
