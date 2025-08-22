package indexer

const (
	FlagIndexerEnabled = "indexer-enabled"
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
