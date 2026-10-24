package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeMIMESubjectBlank(t *testing.T) {
	for _, in := range []string{"", "   ", "\t\n"} {
		decoded, ok := decodeMIMESubject(in)
		require.False(t, ok)
		require.Empty(t, decoded)
	}
}
