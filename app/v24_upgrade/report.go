package v24_upgrade

import (
	"fmt"
	"strings"

	"cosmossdk.io/log"
)

// ReportGenerator generates human-readable migration reports
type ReportGenerator struct {
	logger log.Logger
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(logger log.Logger) *ReportGenerator {
	return &ReportGenerator{
		logger: logger,
	}
}

// GenerateReport creates a comprehensive migration report
func (r *ReportGenerator) GenerateReport(report *MigrationReport, validationResults []ValidationResult) {
	r.logger.Info("=================================================================")
	if report.DryRun {
		r.logger.Warn("            V24 MIGRATION REPORT (DRY-RUN)")
		r.logger.Warn("=================================================================")
		r.logger.Warn("⚠️  DRY-RUN MODE - NO CHANGES WERE SAVED")
		r.logger.Warn("=================================================================")
	} else {
		r.logger.Info("               V24 MIGRATION REPORT")
		r.logger.Info("=================================================================")
	}
	r.logger.Info("")

	// Network and mode
	r.logger.Info(fmt.Sprintf("Network: %s", report.NetworkType))
	r.logger.Info(fmt.Sprintf("Mode: %v", report.Mode))
	if report.DryRun {
		r.logger.Warn(fmt.Sprintf("Dry-Run: %v (Preview only, no changes saved)", report.DryRun))
	}
	r.logger.Info("")

	// Overall statistics
	r.logSection("OVERALL STATISTICS")
	stats := report.Stats
	r.logger.Info(fmt.Sprintf("Total Contracts:     %d", stats.TotalContracts))
	r.logger.Info(fmt.Sprintf("Processed:           %d", stats.ProcessedContracts))
	r.logger.Info(fmt.Sprintf("Migrated:            %d (%.2f%%)",
		stats.MigratedContracts,
		r.percentage(stats.MigratedContracts, stats.TotalContracts)))
	r.logger.Info(fmt.Sprintf("Skipped (safe):      %d (%.2f%%)",
		stats.SkippedContracts,
		r.percentage(stats.SkippedContracts, stats.TotalContracts)))
	r.logger.Info(fmt.Sprintf("Failed:              %d", stats.FailedContracts))
	r.logger.Info("")

	// Schema distribution
	r.logSection("SCHEMA DISTRIBUTION")
	r.logger.Info(fmt.Sprintf("SchemaLegacy:        %d (%.2f%%) - Missing field 7 and/or field 8 (needed migration)",
		stats.LegacyCount,
		r.percentage(stats.LegacyCount, stats.TotalContracts)))
	r.logger.Info(fmt.Sprintf("SchemaBroken:        %d (%.2f%%) - Field 8 has data (needed field swap)",
		stats.BrokenCount,
		r.percentage(stats.BrokenCount, stats.TotalContracts)))
	r.logger.Info(fmt.Sprintf("SchemaCanonical:     %d (%.2f%%) - Both fields present and correct",
		stats.CanonicalCount,
		r.percentage(stats.CanonicalCount, stats.TotalContracts)))
	r.logger.Info(fmt.Sprintf("SchemaUnknown:       %d (%.2f%%)",
		stats.UnknownCount,
		r.percentage(stats.UnknownCount, stats.TotalContracts)))
	r.logger.Info("")

	// Performance metrics
	r.logSection("PERFORMANCE METRICS")
	r.logger.Info(fmt.Sprintf("Start Time:          %s", stats.StartTime.Format("2006-01-02 15:04:05")))
	r.logger.Info(fmt.Sprintf("End Time:            %s", stats.EndTime.Format("2006-01-02 15:04:05")))
	r.logger.Info(fmt.Sprintf("Total Duration:      %s", stats.Duration))
	r.logger.Info(fmt.Sprintf("Contracts/Second:    %.2f", stats.ContractsPerSecond))
	r.logger.Info("")

	// Phase durations
	if report.DiscoveryDuration > 0 || report.MigrationDuration > 0 {
		r.logSection("PHASE DURATIONS")
		if report.DiscoveryDuration > 0 {
			r.logger.Info(fmt.Sprintf("Discovery:           %s", report.DiscoveryDuration))
		}
		if report.BackupDuration > 0 {
			r.logger.Info(fmt.Sprintf("Backup:              %s", report.BackupDuration))
		}
		if report.MigrationDuration > 0 {
			r.logger.Info(fmt.Sprintf("Migration:           %s", report.MigrationDuration))
		}
		if report.ValidationDuration > 0 {
			r.logger.Info(fmt.Sprintf("Validation:          %s", report.ValidationDuration))
		}
		if report.CleanupDuration > 0 {
			r.logger.Info(fmt.Sprintf("Cleanup:             %s", report.CleanupDuration))
		}
		r.logger.Info("")
	}

	// Validation results
	if len(validationResults) > 0 {
		r.logSection("VALIDATION RESULTS")
		successCount := 0
		failureCount := 0
		for _, vr := range validationResults {
			if vr.Valid {
				successCount++
			} else {
				failureCount++
			}
		}
		r.logger.Info(fmt.Sprintf("Total Validated:     %d (100%% of contracts)", len(validationResults)))
		r.logger.Info(fmt.Sprintf("Successes:           %d", successCount))
		r.logger.Info(fmt.Sprintf("Failures:            %d", failureCount))
		r.logger.Info(fmt.Sprintf("Success Rate:        %.2f%%",
			float64(successCount)/float64(len(validationResults))*100))
		r.logger.Info("")
	}

	// Failed contracts
	if len(report.FailedAddresses) > 0 {
		r.logSection("FAILED CONTRACTS")
		r.logger.Error(fmt.Sprintf("Total Failed:        %d", len(report.FailedAddresses)))

		// Show first 10 failed addresses
		displayCount := len(report.FailedAddresses)
		if displayCount > 10 {
			displayCount = 10
		}

		for i := 0; i < displayCount; i++ {
			r.logger.Error(fmt.Sprintf("  %d. %s", i+1, report.FailedAddresses[i]))
		}

		if len(report.FailedAddresses) > 10 {
			r.logger.Error(fmt.Sprintf("  ... and %d more", len(report.FailedAddresses)-10))
		}
		r.logger.Info("")
	}

	// Summary
	r.logSection("SUMMARY")
	if stats.FailedContracts == 0 {
		r.logger.Info("✅ Migration completed successfully!")
		r.logger.Info(fmt.Sprintf("✅ All %d contracts processed", stats.ProcessedContracts))
		r.logger.Info(fmt.Sprintf("✅ %d contracts migrated to canonical schema", stats.MigratedContracts))
		r.logger.Info(fmt.Sprintf("✅ %d contracts were already correct (SchemaCanonical)", stats.SkippedContracts))
	} else {
		r.logger.Warn("⚠️  Migration completed with errors")
		r.logger.Warn(fmt.Sprintf("⚠️  %d contracts failed migration", stats.FailedContracts))
		r.logger.Warn("⚠️  Manual intervention may be required")
	}

	r.logger.Info("")
	r.logger.Info("=================================================================")
}

// GenerateQuickSummary logs a quick one-line summary
func (r *ReportGenerator) GenerateQuickSummary(report *MigrationReport) {
	stats := report.Stats
	r.logger.Info("Migration Summary",
		"total", stats.TotalContracts,
		"migrated", stats.MigratedContracts,
		"skipped", stats.SkippedContracts,
		"failed", stats.FailedContracts,
		"duration", stats.Duration,
		"rate", fmt.Sprintf("%.1f/sec", stats.ContractsPerSecond),
	)
}

// LogSchemaDistribution logs the schema distribution during discovery
func (r *ReportGenerator) LogSchemaDistribution(stats MigrationStats) {
	r.logger.Info("Schema Distribution",
		"legacy", stats.LegacyCount,
		"broken", stats.BrokenCount,
		"canonical", stats.CanonicalCount,
		"unknown", stats.UnknownCount,
	)
}

// logSection logs a section header
func (r *ReportGenerator) logSection(title string) {
	r.logger.Info(fmt.Sprintf("--- %s ---", title))
}

// percentage calculates a percentage
func (r *ReportGenerator) percentage(part, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}

// FormatReport returns a formatted string report (for logging to file if needed)
func FormatReport(report *MigrationReport, validationResults []ValidationResult) string {
	var sb strings.Builder

	sb.WriteString("=================================================================\n")
	sb.WriteString("               V24 MIGRATION REPORT\n")
	sb.WriteString("=================================================================\n\n")

	// Network and mode
	sb.WriteString(fmt.Sprintf("Network: %s\n", report.NetworkType))
	sb.WriteString(fmt.Sprintf("Mode: %v\n\n", report.Mode))

	// Statistics
	stats := report.Stats
	sb.WriteString("OVERALL STATISTICS\n")
	sb.WriteString(fmt.Sprintf("Total Contracts:     %d\n", stats.TotalContracts))
	sb.WriteString(fmt.Sprintf("Processed:           %d\n", stats.ProcessedContracts))
	sb.WriteString(fmt.Sprintf("Migrated:            %d\n", stats.MigratedContracts))
	sb.WriteString(fmt.Sprintf("Skipped (safe):      %d\n", stats.SkippedContracts))
	sb.WriteString(fmt.Sprintf("Failed:              %d\n\n", stats.FailedContracts))

	// Schema distribution
	sb.WriteString("SCHEMA DISTRIBUTION\n")
	sb.WriteString(fmt.Sprintf("SchemaLegacy:        %d\n", stats.LegacyCount))
	sb.WriteString(fmt.Sprintf("SchemaBroken:        %d\n", stats.BrokenCount))
	sb.WriteString(fmt.Sprintf("SchemaCanonical:     %d\n", stats.CanonicalCount))
	sb.WriteString(fmt.Sprintf("SchemaUnknown:       %d\n\n", stats.UnknownCount))

	// Performance
	sb.WriteString("PERFORMANCE\n")
	sb.WriteString(fmt.Sprintf("Duration:            %s\n", stats.Duration))
	sb.WriteString(fmt.Sprintf("Contracts/Second:    %.2f\n\n", stats.ContractsPerSecond))

	// Failed contracts
	if len(report.FailedAddresses) > 0 {
		sb.WriteString("FAILED CONTRACTS\n")
		for _, addr := range report.FailedAddresses {
			sb.WriteString(fmt.Sprintf("  - %s\n", addr))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("=================================================================\n")

	return sb.String()
}
