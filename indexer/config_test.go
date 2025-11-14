package indexer

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// TestDefaultConfig verifies the default indexer configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	require.False(t, config.Enabled, "indexer should be disabled by default")
}

// TestDefaultConfigTemplate verifies the default TOML template generation
func TestDefaultConfigTemplate(t *testing.T) {
	template := DefaultConfigTemplate()

	// Should contain the indexer section
	require.Contains(t, template, "[indexer]")
	require.Contains(t, template, "enabled = false")

	// Verify it's valid TOML by parsing it
	v := viper.New()
	v.SetConfigType("toml")
	err := v.ReadConfig(strings.NewReader(template))
	require.NoError(t, err, "template should be valid TOML")

	// Verify values can be read back
	require.False(t, v.GetBool("indexer.enabled"))
}

// TestConfigTemplateEnabled verifies template with enabled config
func TestConfigTemplateEnabled(t *testing.T) {
	config := Config{
		Enabled: true,
	}

	template := ConfigTemplate(config)

	require.Contains(t, template, "[indexer]")
	require.Contains(t, template, "enabled = true")

	// Verify it's valid TOML
	v := viper.New()
	v.SetConfigType("toml")
	err := v.ReadConfig(strings.NewReader(template))
	require.NoError(t, err)

	// Verify value
	require.True(t, v.GetBool("indexer.enabled"))
}

// TestConfigTemplateDisabled verifies template with disabled config
func TestConfigTemplateDisabled(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	template := ConfigTemplate(config)

	require.Contains(t, template, "[indexer]")
	require.Contains(t, template, "enabled = false")

	// Verify it's valid TOML
	v := viper.New()
	v.SetConfigType("toml")
	err := v.ReadConfig(strings.NewReader(template))
	require.NoError(t, err)

	// Verify value
	require.False(t, v.GetBool("indexer.enabled"))
}

// TestNewConfigFromOptions verifies config creation from app options
func TestNewConfigFromOptions(t *testing.T) {
	tests := []struct {
		name           string
		optionsContent string
		expectedConfig Config
	}{
		{
			name: "enabled true",
			optionsContent: `
[indexer]
enabled = true
`,
			expectedConfig: Config{Enabled: true},
		},
		{
			name: "enabled false",
			optionsContent: `
[indexer]
enabled = false
`,
			expectedConfig: Config{Enabled: false},
		},
		{
			name:           "missing config defaults to false",
			optionsContent: ``,
			expectedConfig: Config{Enabled: false},
		},
		{
			name: "other sections present",
			optionsContent: `
[api]
enable = true

[indexer]
enabled = true

[grpc]
enable = true
`,
			expectedConfig: Config{Enabled: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create viper instance and load config
			v := viper.New()
			v.SetConfigType("toml")

			if tt.optionsContent != "" {
				err := v.ReadConfig(strings.NewReader(tt.optionsContent))
				require.NoError(t, err)
			}

			// Create config from options
			config := NewConfigFromOptions(v)

			require.Equal(t, tt.expectedConfig.Enabled, config.Enabled)
		})
	}
}

// TestConfigRoundTrip verifies config can be written and read back
func TestConfigRoundTrip(t *testing.T) {
	// Create config
	originalConfig := Config{
		Enabled: true,
	}

	// Generate template
	template := ConfigTemplate(originalConfig)

	// Parse template back
	v := viper.New()
	v.SetConfigType("toml")
	err := v.ReadConfig(strings.NewReader(template))
	require.NoError(t, err)

	// Create new config from parsed template
	newConfig := NewConfigFromOptions(v)

	// Should match original
	require.Equal(t, originalConfig.Enabled, newConfig.Enabled)
}

// TestConfigTemplateFormat verifies the template formatting
func TestConfigTemplateFormat(t *testing.T) {
	config := Config{Enabled: true}
	template := ConfigTemplate(config)

	// Should have proper TOML section header
	require.Contains(t, template, "[indexer]")

	// Should have the enabled field
	require.Contains(t, template, "enabled")

	// Should have proper boolean formatting (not quoted)
	require.Contains(t, template, "enabled = true")
	require.NotContains(t, template, "enabled = \"true\"", "boolean should not be quoted")
}

// TestNewConfigFromOptionsInvalidTypes verifies handling of type mismatches
func TestNewConfigFromOptionsInvalidTypes(t *testing.T) {
	tests := []struct {
		name           string
		optionsContent string
		expectedConfig Config
		description    string
	}{
		{
			name: "string true converts to bool",
			optionsContent: `
[indexer]
enabled = "true"
`,
			expectedConfig: Config{Enabled: true},
			description:    "cast.ToBool should handle string 'true'",
		},
		{
			name: "string false converts to bool",
			optionsContent: `
[indexer]
enabled = "false"
`,
			expectedConfig: Config{Enabled: false},
			description:    "cast.ToBool should handle string 'false'",
		},
		{
			name: "number 1 converts to true",
			optionsContent: `
[indexer]
enabled = 1
`,
			expectedConfig: Config{Enabled: true},
			description:    "cast.ToBool should handle number 1 as true",
		},
		{
			name: "number 0 converts to false",
			optionsContent: `
[indexer]
enabled = 0
`,
			expectedConfig: Config{Enabled: false},
			description:    "cast.ToBool should handle number 0 as false",
		},
		{
			name: "invalid string converts to false",
			optionsContent: `
[indexer]
enabled = "invalid"
`,
			expectedConfig: Config{Enabled: false},
			description:    "cast.ToBool should default to false for invalid strings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			v.SetConfigType("toml")
			err := v.ReadConfig(strings.NewReader(tt.optionsContent))
			require.NoError(t, err)

			config := NewConfigFromOptions(v)
			require.Equal(t, tt.expectedConfig.Enabled, config.Enabled, tt.description)
		})
	}
}

// TestConfigStruct verifies the Config struct fields
func TestConfigStruct(t *testing.T) {
	config := Config{
		Enabled: true,
	}

	// Verify field is accessible
	require.True(t, config.Enabled)

	// Verify we can modify it
	config.Enabled = false
	require.False(t, config.Enabled)
}

// TestMultipleConfigInstances verifies each config instance is independent
func TestMultipleConfigInstances(t *testing.T) {
	config1 := Config{Enabled: true}
	config2 := Config{Enabled: false}
	config3 := DefaultConfig()

	// Each should have independent values
	require.True(t, config1.Enabled)
	require.False(t, config2.Enabled)
	require.False(t, config3.Enabled)

	// Modifying one shouldn't affect others
	config1.Enabled = false
	require.False(t, config1.Enabled)
	require.False(t, config2.Enabled)
	require.False(t, config3.Enabled)
}
