#!/bin/bash

# Coverage analysis script with detailed reporting and validation
# Usage: coverage-analyze.sh <coverage_file> <coveragerc_file> [threshold]

COVERAGE_FILE="${1:-coverage_filtered.out}"
COVERAGERC_FILE="${2:-.coveragerc}"
COVERAGE_THRESHOLD="${3}"

if [ ! -f "$COVERAGE_FILE" ]; then
    echo "❌ Coverage file not found: $COVERAGE_FILE"
    exit 1
fi

echo "📊 Analyzing coverage from: $COVERAGE_FILE"

# Parse .coveragerc file and extract threshold and exclusions
parse_coveragerc() {
    local file="$1"
    local section=""

    if [ ! -f "$file" ]; then
        return
    fi

    while IFS= read -r line || [ -n "$line" ]; do
        # Trim whitespace
        line=$(echo "$line" | xargs)

        # Skip comments and empty lines
        if [[ "$line" =~ ^# || -z "$line" ]]; then
            continue
        fi

        # Check for section headers
        if [[ "$line" =~ ^\[(.+)\]$ ]]; then
            section="${BASH_REMATCH[1]}"
            continue
        fi

        # Parse key-value pairs in [run] section
        if [[ "$section" == "run" && "$line" =~ ^threshold[[:space:]]*=[[:space:]]*(.+)$ ]]; then
            COVERAGERC_THRESHOLD="${BASH_REMATCH[1]}"
        fi

        # Collect exclusion patterns from [exclude] section
        if [[ "$section" == "exclude" && ! "$line" =~ ^# ]]; then
            if [ -n "$COVERAGERC_EXCLUSIONS" ]; then
                COVERAGERC_EXCLUSIONS="$COVERAGERC_EXCLUSIONS $line"
            else
                COVERAGERC_EXCLUSIONS="$line"
            fi
        fi
    done < "$file"

    echo "📋 Loaded configuration from: $file"
}

# Global variables for parsed config
COVERAGERC_THRESHOLD=""
COVERAGERC_EXCLUSIONS=""

# Parse the .coveragerc file
parse_coveragerc "$COVERAGERC_FILE"

# Use threshold from command line, or .coveragerc, or default to 85
if [ -z "$COVERAGE_THRESHOLD" ]; then
    if [ -n "$COVERAGERC_THRESHOLD" ]; then
        COVERAGE_THRESHOLD="$COVERAGERC_THRESHOLD"
    else
        COVERAGE_THRESHOLD="85"
    fi
fi

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

# Use exclusions from parsed config
EXCLUSION_PATTERNS="$COVERAGERC_EXCLUSIONS"

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
if [ "$low_coverage_count" -gt 0 ] || (command -v bc >/dev/null 2>&1 && [ $(echo "$total_coverage_num < $COVERAGE_THRESHOLD" | bc -l) -eq 1 ]); then
    echo ""
    if [ "$low_coverage_count" -gt 0 ]; then
        echo "💡 Add tests for functions with low coverage"
    fi
    if command -v bc >/dev/null 2>&1 && [ $(echo "$total_coverage_num < $COVERAGE_THRESHOLD" | bc -l) -eq 1 ]; then
        echo "⚠️  Total coverage $total_coverage is below ${COVERAGE_THRESHOLD}% threshold"
    fi
    exit 1
else
    echo ""
    echo "✅ All coverage requirements met (>${COVERAGE_THRESHOLD}% total, no low coverage functions)"
fi
