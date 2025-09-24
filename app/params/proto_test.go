package params

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMakeEncodingConfig(t *testing.T) {
	config := MakeEncodingConfig()

	require.NotNil(t, config)
	require.NotNil(t, config.InterfaceRegistry)
	require.NotNil(t, config.Codec)
	require.NotNil(t, config.TxConfig)
	require.NotNil(t, config.Amino)

	// Test that the interface registry is properly initialized
	require.IsType(t, config.InterfaceRegistry, config.InterfaceRegistry)

	// Test that the codec is properly initialized
	require.IsType(t, config.Codec, config.Codec)

	// Test that the tx config is properly initialized
	require.IsType(t, config.TxConfig, config.TxConfig)

	// Test that the amino codec is properly initialized
	require.IsType(t, config.Amino, config.Amino)
}