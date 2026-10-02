package testlib

import (
	"crypto/sha256"
	"encoding/base64"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalAddressDerivationHashMatchesProofFixtureContract(t *testing.T) {
	contract, err := os.ReadFile(IntegrationTestPath("testdata", "contracts", "xion_account.wasm"))
	require.NoError(t, err)

	want, err := base64.StdEncoding.DecodeString(localAddressDerivationHash)
	require.NoError(t, err)
	got := sha256.Sum256(contract)

	require.Equal(t, want, got[:])
}
