#!/bin/bash

# Simplified coverage analysis script
# Usage: coverage-analyze.sh <coverage_file> <exclusion_patterns>

COVERAGE_FILE="${1:-coverage_filtered.out}"
EXCLUSION_PATTERNS="$2"

if [ ! -f "$COVERAGE_FILE" ]; then
    echo "❌ Coverage file not found: $COVERAGE_FILE"
    exit 1
fi

echo "📊 Analyzing coverage from: $COVERAGE_FILE"

# Function to build grep exclusion command
build_exclusion_grep() {
    local patterns="$1"
    local grep_cmd=""

    if [ -n "$patterns" ]; then
        # Convert space-separated patterns to grep exclusions
        for pattern in $patterns; do
            if [ -z "$grep_cmd" ]; then
                grep_cmd="grep -v \"$pattern\""
            else
                grep_cmd="$grep_cmd | grep -v \"$pattern\""
            fi
        done
    fi

    echo "$grep_cmd"
}

# Show excellent coverage (90-100%)
echo ""
echo "=== EXCELLENT COVERAGE (90-100%) ==="
excellent_coverage=$(go tool cover -func="$COVERAGE_FILE" | awk '$3 ~ /^9[0-9]\.[0-9]%$/ || $3 ~ /^100\.0%$/')
if [ -n "$excellent_coverage" ]; then
    echo "$excellent_coverage" | awk '{
        if ($3 == "100.0%") 
            print "🎯 " $0
        else 
            print "⭐ " $0
    }'
else
    echo "No functions with 90-100% coverage found"
fi

# Show good coverage (70-89%)
echo ""
echo "=== GOOD COVERAGE (70-89%) ==="
good_coverage=$(go tool cover -func="$COVERAGE_FILE" | awk '$3 ~ /^[78][0-9]\.[0-9]%$/')
if [ -n "$good_coverage" ]; then
    echo "$good_coverage" | awk '{print "✅ " $0}'
else
    echo "No functions with 70-89% coverage found"
fi

# Show ok coverage (50-69%)
echo ""
echo "=== OK COVERAGE (50-69%) ==="
ok_coverage=$(go tool cover -func="$COVERAGE_FILE" | awk '$3 ~ /^[56][0-9]\.[0-9]%$/')
if [ -n "$ok_coverage" ]; then
    echo "$ok_coverage" | awk '{print "⚠️  " $0}'
else
    echo "No functions with 50-69% coverage found"
fi

# Show low/no coverage (<50% including 0%)
echo ""
echo "=== LOW/NO COVERAGE (<50%) ==="
exclusion_cmd=$(build_exclusion_grep "$EXCLUSION_PATTERNS")

if [ -n "$exclusion_cmd" ]; then
    low_and_zero_coverage=$(eval "go tool cover -func=\"$COVERAGE_FILE\" | awk '\$3 ~ /^[0-4]?[0-9]\\.[0-9]%\$/' | $exclusion_cmd")
else
    low_and_zero_coverage=$(go tool cover -func="$COVERAGE_FILE" | awk '$3 ~ /^[0-4]?[0-9]\.[0-9]%$/')
fi

if [ -n "$low_and_zero_coverage" ]; then
    echo "$low_and_zero_coverage" | awk '{
            print "❌ " $0
    }'
else
    echo "No functions with low coverage found (after exclusions)"
fi

# Count low/no coverage functions (for exit code)
low_coverage_count=0
if [ -n "$exclusion_cmd" ]; then
    low_coverage_count=$(eval "go tool cover -func=\"$COVERAGE_FILE\" | awk '\$3 ~ /^[0-4]?[0-9]\\.[0-9]%\$/' | $exclusion_cmd | wc -l" | xargs)
else
    low_coverage_count=$(go tool cover -func="$COVERAGE_FILE" | awk '$3 ~ /^[0-4]?[0-9]\.[0-9]%$/' | wc -l | xargs)
fi

echo ""
echo "=== ANALYSIS SUMMARY ==="
total_functions=$(go tool cover -func="$COVERAGE_FILE" | grep -v "total:" | wc -l | xargs)
total_coverage=$(go tool cover -func="$COVERAGE_FILE" | tail -1 | awk '{print $3}')
perfect_count=$(go tool cover -func="$COVERAGE_FILE" | awk '$3 ~ /^100\.0%$/' | wc -l | xargs)
excellent_count=$(go tool cover -func="$COVERAGE_FILE" | awk '$3 ~ /^9[0-9]\.[0-9]%$/' | wc -l | xargs)
good_count=$(go tool cover -func="$COVERAGE_FILE" | awk '$3 ~ /^[78][0-9]\.[0-9]%$/' | wc -l | xargs)
ok_count=$(go tool cover -func="$COVERAGE_FILE" | awk '$3 ~ /^[56][0-9]\.[0-9]%$/' | wc -l | xargs)

echo "📊 Total functions analyzed: $total_functions"
echo "📈 Overall coverage: $total_coverage"
echo ""
echo "📋 Coverage Breakdown:"
total_excellent_count=$((perfect_count + excellent_count))
echo "⭐ Excellent (90-100%): $total_excellent_count (🎯 Perfect: $perfect_count)"
echo "✅ Good (70-89%): $good_count"
echo "⚠️  OK (50-69%): $ok_count"
echo "❌ Low/No (<50%): $low_coverage_count"

# Extract numeric value from total coverage for comparison
total_coverage_num=$(echo "$total_coverage" | sed 's/%//')

# Check both low coverage count and total coverage threshold
if [ "$low_coverage_count" -gt 0 ] || (command -v bc >/dev/null 2>&1 && [ $(echo "$total_coverage_num < 85" | bc -l) -eq 1 ]); then
    echo ""
    if [ "$low_coverage_count" -gt 0 ]; then
        echo "💡 Add tests for functions with low coverage"
    fi
    if command -v bc >/dev/null 2>&1 && [ $(echo "$total_coverage_num < 85" | bc -l) -eq 1 ]; then
        echo "⚠️  Total coverage $total_coverage is below 85% threshold"
    fi
    exit 1
else
    echo ""
    echo "✅ All coverage requirements met (>85% total, no low coverage functions)"
fi
