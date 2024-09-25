package prover

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLoadR1CS(t *testing.T) {
	_, err := readR1CSFromFile("test/tx_auth.r1cs")
	require.NoError(t, err)

}
