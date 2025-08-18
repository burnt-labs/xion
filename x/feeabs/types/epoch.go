package types

import (
	"errors"
	"time"
)

const (
	DefaultSwapPeriod           = time.Minute * 2
	DefaultQueryPeriod          = time.Minute * 1
	DefaultSwapEpochIdentifier  = "swap"
	DefaultQueryEpochIdentifier = "query"
	// assume that query period is after 1 minute, thus a maximum of 32 minutes is enough
	// todo: should be a parameter
	// 1, 2, 4, 8, 16, 32
	ExponentialMaxJump = 32
	// after 4 jump, a connection is considered outdated
	ExponentialOutdatedJump = 4
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

// Validate also validates epoch info.
func (epoch EpochInfo) Validate() error {
	if epoch.Identifier == "" {
		return errors.New("epoch identifier should NOT be empty")
	}
	if epoch.Duration == 0 {
		return errors.New("epoch duration should NOT be 0")
	}
	if epoch.CurrentEpoch < 0 {
		return errors.New("epoch CurrentEpoch must be non-negative")
	}
	if epoch.CurrentEpochStartHeight < 0 {
		return errors.New("epoch CurrentEpoch must be non-negative")
	}
	return nil
}

func NewGenesisEpochInfo(identifier string, duration time.Duration) EpochInfo {
	return EpochInfo{
		Identifier:              identifier,
		StartTime:               time.Time{},
		Duration:                duration,
		CurrentEpoch:            0,
		CurrentEpochStartHeight: 0,
		CurrentEpochStartTime:   time.Time{},
		EpochCountingStarted:    false,
	}
}
