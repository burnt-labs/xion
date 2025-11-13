package indexer

import (
	"fmt"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/spf13/cast"
)

// Config defines the indexer configuration.
type Config struct {
	Enabled bool `mapstructure:"enabled" json:"enabled"`
}

func DefaultConfig() Config {
	return Config{
		Enabled: false,
	}
}

// DefaultConfigTemplate returns the default TOML snippet for the indexer configuration.
func DefaultConfigTemplate() string {
	return ConfigTemplate(DefaultConfig())
}

// ConfigTemplate returns the TOML snippet for the indexer configuration.
func ConfigTemplate(c Config) string {
	return fmt.Sprintf(`
[indexer]
enabled = %t
`, c.Enabled)
}

func NewConfigFromOptions(opts servertypes.AppOptions) Config {
	enabled := cast.ToBool(opts.Get("indexer.enabled"))
	return Config{
		Enabled: enabled,
	}
}
