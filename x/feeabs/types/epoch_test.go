package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEpochConstants(t *testing.T) {
	require.Equal(t, time.Minute*2, DefaultSwapPeriod)
	require.Equal(t, time.Minute*1, DefaultQueryPeriod)
	require.Equal(t, "swap", DefaultSwapEpochIdentifier)
	require.Equal(t, "query", DefaultQueryEpochIdentifier)
	require.Equal(t, int64(32), int64(ExponentialMaxJump))
	require.Equal(t, int64(4), int64(ExponentialOutdatedJump))
}

func TestKeyPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []byte(""),
		},
		{
			name:     "simple string",
			input:    "test",
			expected: []byte("test"),
		},
		{
			name:     "complex string",
			input:    "test-key-123",
			expected: []byte("test-key-123"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := KeyPrefix(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestEpochInfoValidate(t *testing.T) {
	tests := []struct {
		name  string
		epoch EpochInfo
		valid bool
	}{
		{
			name: "valid epoch info",
			epoch: EpochInfo{
				Identifier:              "test-epoch",
				StartTime:               time.Now(),
				Duration:                time.Hour,
				CurrentEpoch:            1,
				CurrentEpochStartHeight: 100,
				CurrentEpochStartTime:   time.Now(),
				EpochCountingStarted:    true,
			},
			valid: true,
		},
		{
			name: "valid epoch info with zero values",
			epoch: EpochInfo{
				Identifier:              "test-epoch",
				StartTime:               time.Time{},
				Duration:                time.Second,
				CurrentEpoch:            0,
				CurrentEpochStartHeight: 0,
				CurrentEpochStartTime:   time.Time{},
				EpochCountingStarted:    false,
			},
			valid: true,
		},
		{
			name: "empty identifier",
			epoch: EpochInfo{
				Identifier:              "",
				StartTime:               time.Now(),
				Duration:                time.Hour,
				CurrentEpoch:            1,
				CurrentEpochStartHeight: 100,
				CurrentEpochStartTime:   time.Now(),
				EpochCountingStarted:    true,
			},
			valid: false,
		},
		{
			name: "zero duration",
			epoch: EpochInfo{
				Identifier:              "test-epoch",
				StartTime:               time.Now(),
				Duration:                0,
				CurrentEpoch:            1,
				CurrentEpochStartHeight: 100,
				CurrentEpochStartTime:   time.Now(),
				EpochCountingStarted:    true,
			},
			valid: false,
		},
		{
			name: "negative current epoch",
			epoch: EpochInfo{
				Identifier:              "test-epoch",
				StartTime:               time.Now(),
				Duration:                time.Hour,
				CurrentEpoch:            -1,
				CurrentEpochStartHeight: 100,
				CurrentEpochStartTime:   time.Now(),
				EpochCountingStarted:    true,
			},
			valid: false,
		},
		{
			name: "negative current epoch start height",
			epoch: EpochInfo{
				Identifier:              "test-epoch",
				StartTime:               time.Now(),
				Duration:                time.Hour,
				CurrentEpoch:            1,
				CurrentEpochStartHeight: -1,
				CurrentEpochStartTime:   time.Now(),
				EpochCountingStarted:    true,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.epoch.Validate()
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestNewGenesisEpochInfo(t *testing.T) {
	identifier := "test-epoch"
	duration := time.Hour * 24

	epochInfo := NewGenesisEpochInfo(identifier, duration)

	require.Equal(t, identifier, epochInfo.Identifier)
	require.Equal(t, time.Time{}, epochInfo.StartTime)
	require.Equal(t, duration, epochInfo.Duration)
	require.Equal(t, int64(0), epochInfo.CurrentEpoch)
	require.Equal(t, int64(0), epochInfo.CurrentEpochStartHeight)
	require.Equal(t, time.Time{}, epochInfo.CurrentEpochStartTime)
	require.False(t, epochInfo.EpochCountingStarted)

	// Validate that the created epoch is valid
	err := epochInfo.Validate()
	require.NoError(t, err)
}

func TestNewGenesisEpochInfoWithDefaultValues(t *testing.T) {
	// Test with default swap epoch
	swapEpoch := NewGenesisEpochInfo(DefaultSwapEpochIdentifier, DefaultSwapPeriod)
	require.Equal(t, DefaultSwapEpochIdentifier, swapEpoch.Identifier)
	require.Equal(t, DefaultSwapPeriod, swapEpoch.Duration)
	err := swapEpoch.Validate()
	require.NoError(t, err)

	// Test with default query epoch
	queryEpoch := NewGenesisEpochInfo(DefaultQueryEpochIdentifier, DefaultQueryPeriod)
	require.Equal(t, DefaultQueryEpochIdentifier, queryEpoch.Identifier)
	require.Equal(t, DefaultQueryPeriod, queryEpoch.Duration)
	err = queryEpoch.Validate()
	require.NoError(t, err)
}
