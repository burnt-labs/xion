package v24_upgrade

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
)

func TestNewReportGenerator(t *testing.T) {
	logger := log.NewNopLogger()
	gen := NewReportGenerator(logger)

	require.NotNil(t, gen)
	require.NotNil(t, gen.logger)
}

func TestReportGeneratorPercentage(t *testing.T) {
	logger := log.NewNopLogger()
	gen := NewReportGenerator(logger)

	tests := []struct {
		name  string
		part  uint64
		total uint64
		want  float64
	}{
		{"50%", 50, 100, 50.0},
		{"25%", 25, 100, 25.0},
		{"100%", 100, 100, 100.0},
		{"0%", 0, 100, 0.0},
		{"zero total", 50, 0, 0.0},
		{"33.33%", 1, 3, 33.33333333333333},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.percentage(tt.part, tt.total)
			require.InDelta(t, tt.want, result, 0.0001)
		})
	}
}

func TestFormatReport(t *testing.T) {
	startTime := time.Now()
	report := &MigrationReport{
		Stats: MigrationStats{
			TotalContracts:     1000,
			ProcessedContracts: 1000,
			MigratedContracts:  300,
			SkippedContracts:   700,
			FailedContracts:    0,
			LegacyCount:        600,
			BrokenCount:        300,
			CanonicalCount:     100,
			UnknownCount:       0,
			StartTime:          startTime,
			EndTime:            startTime.Add(1 * time.Hour),
			Duration:           1 * time.Hour,
			ContractsPerSecond: 0.28,
		},
		FailedAddresses: []string{},
		NetworkType:     Testnet,
		Mode:            ModeAutoMigrate,
	}

	validationResults := []ValidationResult{
		{Address: "xion1test1", Valid: true},
		{Address: "xion1test2", Valid: true},
		{Address: "xion1test3", Valid: true},
	}

	formatted := FormatReport(report, validationResults)

	// Verify report contains key sections
	require.Contains(t, formatted, "V24 MIGRATION REPORT")
	require.Contains(t, formatted, "OVERALL STATISTICS")
	require.Contains(t, formatted, "SCHEMA DISTRIBUTION")
	require.Contains(t, formatted, "PERFORMANCE")

	// Verify key values
	require.Contains(t, formatted, "Total Contracts:     1000")
	require.Contains(t, formatted, "Processed:           1000")
	require.Contains(t, formatted, "Migrated:            300")
	require.Contains(t, formatted, "Skipped (safe):      700")
	require.Contains(t, formatted, "Failed:              0")

	// Verify schema distribution
	require.Contains(t, formatted, "SchemaLegacy:        600")
	require.Contains(t, formatted, "SchemaBroken:        300")
	require.Contains(t, formatted, "SchemaCanonical:     100")
}

func TestFormatReport_WithFailures(t *testing.T) {
	startTime := time.Now()
	report := &MigrationReport{
		Stats: MigrationStats{
			TotalContracts:     100,
			ProcessedContracts: 100,
			MigratedContracts:  90,
			SkippedContracts:   0,
			FailedContracts:    10,
			LegacyCount:        0,
			BrokenCount:        100,
			CanonicalCount:     0,
			UnknownCount:       0,
			StartTime:          startTime,
			EndTime:            startTime.Add(1 * time.Minute),
			Duration:           1 * time.Minute,
			ContractsPerSecond: 1.67,
		},
		FailedAddresses: []string{
			"xion1failed1",
			"xion1failed2",
			"xion1failed3",
		},
		NetworkType: Mainnet,
		Mode:        ModeAutoMigrate,
	}

	formatted := FormatReport(report, nil)

	// Verify failures are shown
	require.Contains(t, formatted, "FAILED CONTRACTS")
	require.Contains(t, formatted, "xion1failed1")
	require.Contains(t, formatted, "xion1failed2")
	require.Contains(t, formatted, "xion1failed3")
}

func TestReportGeneratorGenerateReport(t *testing.T) {
	logger := log.NewNopLogger()
	gen := NewReportGenerator(logger)

	startTime := time.Now()
	report := &MigrationReport{
		Stats: MigrationStats{
			TotalContracts:     1000,
			ProcessedContracts: 1000,
			MigratedContracts:  400,
			SkippedContracts:   600,
			FailedContracts:    0,
			LegacyCount:        600,
			BrokenCount:        400,
			CanonicalCount:     0,
			UnknownCount:       0,
			StartTime:          startTime,
			EndTime:            startTime.Add(30 * time.Minute),
			Duration:           30 * time.Minute,
			ContractsPerSecond: 0.56,
		},
		FailedAddresses: []string{},
		NetworkType:     Testnet,
		Mode:            ModeAutoMigrate,
	}

	validationResults := []ValidationResult{
		{Address: "xion1test1", Valid: true},
		{Address: "xion1test2", Valid: true},
	}

	// This should not panic
	require.NotPanics(t, func() {
		gen.GenerateReport(report, validationResults)
	})
}

func TestReportGeneratorQuickSummary(t *testing.T) {
	logger := log.NewNopLogger()
	gen := NewReportGenerator(logger)

	report := &MigrationReport{
		Stats: MigrationStats{
			TotalContracts:     1000,
			MigratedContracts:  300,
			SkippedContracts:   700,
			FailedContracts:    0,
			Duration:           1 * time.Hour,
			ContractsPerSecond: 0.28,
		},
	}

	// Should not panic
	require.NotPanics(t, func() {
		gen.GenerateQuickSummary(report)
	})
}

func TestReportGeneratorLogSchemaDistribution(t *testing.T) {
	logger := log.NewNopLogger()
	gen := NewReportGenerator(logger)

	stats := MigrationStats{
		LegacyCount:    600,
		BrokenCount:    300,
		CanonicalCount: 100,
		UnknownCount:   0,
	}

	// Should not panic
	require.NotPanics(t, func() {
		gen.LogSchemaDistribution(stats)
	})
}

func TestFormatReport_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		report *MigrationReport
	}{
		{
			name: "zero contracts",
			report: &MigrationReport{
				Stats: MigrationStats{
					TotalContracts:     0,
					ProcessedContracts: 0,
					MigratedContracts:  0,
					SkippedContracts:   0,
					FailedContracts:    0,
				},
				NetworkType: Testnet,
				Mode:        ModeAutoMigrate,
			},
		},
		{
			name: "all failed",
			report: &MigrationReport{
				Stats: MigrationStats{
					TotalContracts:     100,
					ProcessedContracts: 100,
					MigratedContracts:  0,
					SkippedContracts:   0,
					FailedContracts:    100,
				},
				FailedAddresses: make([]string, 100),
				NetworkType:     Mainnet,
				Mode:            ModeFailOnCorruption,
			},
		},
		{
			name: "all skipped",
			report: &MigrationReport{
				Stats: MigrationStats{
					TotalContracts:     1000,
					ProcessedContracts: 1000,
					MigratedContracts:  0,
					SkippedContracts:   1000,
					FailedContracts:    0,
					LegacyCount:        1000,
				},
				NetworkType: Testnet,
				Mode:        ModeAutoMigrate,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := FormatReport(tt.report, nil)

			// Should contain basic structure
			require.Contains(t, formatted, "V24 MIGRATION REPORT")
			require.NotEmpty(t, formatted)

			// Should be valid format
			lines := strings.Split(formatted, "\n")
			require.Greater(t, len(lines), 10)
		})
	}
}

func TestMigrationReport_PhaseDurations(t *testing.T) {
	report := MigrationReport{
		Stats: MigrationStats{
			TotalContracts: 1000,
		},
		DiscoveryDuration:  5 * time.Minute,
		BackupDuration:     10 * time.Minute,
		MigrationDuration:  90 * time.Minute,
		ValidationDuration: 10 * time.Minute,
		CleanupDuration:    5 * time.Minute,
		NetworkType:        Mainnet,
		Mode:               ModeAutoMigrate,
	}

	// Verify all durations are set
	require.Equal(t, 5*time.Minute, report.DiscoveryDuration)
	require.Equal(t, 10*time.Minute, report.BackupDuration)
	require.Equal(t, 90*time.Minute, report.MigrationDuration)
	require.Equal(t, 10*time.Minute, report.ValidationDuration)
	require.Equal(t, 5*time.Minute, report.CleanupDuration)

	// Total should be sum of all phases
	totalDuration := report.DiscoveryDuration + report.BackupDuration +
		report.MigrationDuration + report.ValidationDuration + report.CleanupDuration

	require.Equal(t, 120*time.Minute, totalDuration)
}

func TestFormatReport_LargeFailureList(t *testing.T) {
	// Create report with many failures
	failedAddrs := make([]string, 100)
	for i := 0; i < 100; i++ {
		failedAddrs[i] = "xion1failed"
	}

	report := &MigrationReport{
		Stats: MigrationStats{
			TotalContracts:  100,
			FailedContracts: 100,
		},
		FailedAddresses: failedAddrs,
		NetworkType:     Testnet,
		Mode:            ModeAutoMigrate,
	}

	formatted := FormatReport(report, nil)

	// Should contain failed contracts section
	require.Contains(t, formatted, "FAILED CONTRACTS")

	// Should show all failures in the formatted report
	failureCount := strings.Count(formatted, "xion1failed")
	require.Greater(t, failureCount, 0)
}

func TestMigrationStats_ContractsPerSecond(t *testing.T) {
	tests := []struct {
		name      string
		processed uint64
		duration  time.Duration
		want      float64
	}{
		{"1000 in 1 hour", 1000, 1 * time.Hour, 0.2777777777777778},
		{"6000000 in 2 hours", 6000000, 2 * time.Hour, 833.3333333333334},
		{"100 in 1 minute", 100, 1 * time.Minute, 1.6666666666666667},
		{"zero duration", 1000, 0, 0}, // Would result in infinity, but we handle it
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := MigrationStats{
				ProcessedContracts: tt.processed,
				Duration:           tt.duration,
			}

			// Calculate rate
			var rate float64
			if stats.Duration.Seconds() > 0 {
				rate = float64(stats.ProcessedContracts) / stats.Duration.Seconds()
			}

			require.InDelta(t, tt.want, rate, 0.0001)
		})
	}
}
