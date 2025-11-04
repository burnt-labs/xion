package app

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
)

// CustomConfig defines a custom app.toml configuration file
// that contains default wasm configuration and
// indexer configuration
type CustomConfig struct {
	serverconfig.Config
	Wasm wasmtypes.NodeConfig `mapstructure:"wasm" json:"wasm"`
}

func CustomconfigTemplate(config wasmtypes.NodeConfig) string {
	return serverconfig.DefaultConfigTemplate + wasmtypes.ConfigTemplate(config)
}

func DefaultConfig() (string, any) {
	// Default SDK config params
	serverConfig := serverconfig.DefaultConfig()
	serverConfig.MinGasPrices = "0uxion"

	// Default x/wasm configuration
	wasmConfig := wasmtypes.DefaultNodeConfig()
	simulationLimit := uint64(50_000_00)
	wasmConfig.SimulationGasLimit = &simulationLimit // 50M Gas
	wasmConfig.SmartQueryGasLimit = 50_000_00        // 50M Gas
	wasmConfig.MemoryCacheSize = 1024                // 1GB memory caache

	customConfig := CustomConfig{
		Config: *serverConfig,
		Wasm:   wasmConfig,
	}

	return CustomconfigTemplate(wasmConfig), customConfig
}
