package app

import (
	"strings"
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"
)

func TestCustomconfigTemplate(t *testing.T) {
	// Create a default wasm config
	wasmConfig := wasmtypes.DefaultNodeConfig()

	// Test the template generation
	template := CustomconfigTemplate(wasmConfig)

	// Verify template is not empty
	require.NotEmpty(t, template)

	// Verify it contains wasm configuration markers
	require.Contains(t, template, "wasm", "Template should contain wasm configuration")

	// Verify it's a valid template with expected sections
	require.True(t, strings.Contains(template, "[") && strings.Contains(template, "]"),
		"Template should contain TOML sections")
}

func TestDefaultConfig(t *testing.T) {
	// Call DefaultConfig
	template, configInterface := DefaultConfig()

	// Verify template is returned
	require.NotEmpty(t, template)
	require.NotNil(t, configInterface)

	// Type assert to CustomConfig
	customConfig, ok := configInterface.(CustomConfig)
	require.True(t, ok, "Config should be of type CustomConfig")

	// Verify MinGasPrices is set correctly
	require.Equal(t, "0uxion", customConfig.MinGasPrices)

	// Verify Wasm config is set
	require.NotNil(t, customConfig.Wasm)
	require.NotNil(t, customConfig.Wasm.SimulationGasLimit)
	require.Equal(t, uint64(50_000_00), *customConfig.Wasm.SimulationGasLimit)
	require.Equal(t, uint64(50_000_00), customConfig.Wasm.SmartQueryGasLimit)
	require.Equal(t, uint32(1024), customConfig.Wasm.MemoryCacheSize)

	// Verify template contains wasm configuration
	require.Contains(t, template, "wasm")
}
