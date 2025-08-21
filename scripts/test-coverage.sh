#!/bin/bash

# Script to run tests with coverage, excluding generated protobuf files

# List of patterns to exclude from low coverage reporting
# Add new exclusion patterns here only if needed
COVERAGE_EXCLUSIONS=(
    "x/feeabs/types/params.go.*Validate"
    "github.com/burnt-labs/xion/x/xion/client/cli/tx.go:.*NewRegisterCmd"
    "github.com/burnt-labs/xion/x/xion/client/cli/tx.go:.*NewSignCmd"
    "github.com/burnt-labs/xion/x/xion/keeper/grpc_query.go:.*WebAuthNVerifyRegister"
    "github.com/burnt-labs/xion/x/xion/keeper/grpc_query.go:.*WebAuthNVerifyAuthenticate"
    "github.com/burnt-labs/xion/x/xion/keeper/mint.go:.*StakedInflationMintFn"
    "github.com/burnt-labs/xion/x/xion/keeper/genesis.go:.*InitGenesis"
)

echo "Running tests with coverage (excluding .pb.go files)..."

# Run tests with coverage (only x/ modules for now). Fail fast if tests fail.
go test ./x/... -coverprofile=coverage.out
test_exit_code=$?
if [[ $test_exit_code -ne 0 ]]; then
    echo ""
    echo "=== TEST FAILURE DETECTED ==="
    echo "❌ FAILURE: One or more tests failed (exit code $test_exit_code). Aborting coverage analysis."
    echo "(Adjust the package pattern in this script if you intend to include additional modules beyond ./x/.)"
    exit 1
fi

# Filter out .pb.go and .pb.gw.go files from coverage report
grep -v "\.pb\.go:" coverage.out | grep -v "\.pb\.gw\.go:" > coverage_filtered.out

# Show coverage report without .pb.go files
echo "Coverage report (excluding generated files):"
go tool cover -func=coverage_filtered.out

# Generate HTML report without .pb.go files
go tool cover -html=coverage_filtered.out -o coverage.html

echo "HTML coverage report generated: coverage.html"
echo "Filtered coverage file: coverage_filtered.out"

# Function to build grep exclusion command from array
build_exclusion_grep() {
    local exclusions=("$@")
    local grep_cmd=""
    
    for pattern in "${exclusions[@]}"; do
        if [[ -n "$pattern" && "$pattern" != \#* ]]; then  # Skip empty lines and comments
            if [[ -z "$grep_cmd" ]]; then
                grep_cmd="grep -v \"$pattern\""
            else
                grep_cmd="$grep_cmd | grep -v \"$pattern\""
            fi
        fi
    done
    
    echo "$grep_cmd"
}

# Show modules with ok coverage (50% - 99%)
echo ""
echo "=== OK COVERAGE (80% - 99%) ==="
go tool cover -func=coverage_filtered.out | awk '$3 ~ /^[89][0-9]\.[0-9]%$/'

# Show modules with low coverage (<=50%), including 0%
echo ""
echo "=== LOW COVERAGE (<80%) ==="
exclusion_cmd=$(build_exclusion_grep "${COVERAGE_EXCLUSIONS[@]}")
if [[ -n "$exclusion_cmd" ]]; then
    eval "go tool cover -func=coverage_filtered.out | awk '\$3 ~ /^[0-7]?[0-9]\\.[0-9]%\$/' | $exclusion_cmd"
else
    go tool cover -func=coverage_filtered.out | awk '$3 ~ /^[0-7]?[0-9]\.[0-9]%$/'
fi 

 

# Show summary statistics
echo ""
echo "=== COVERAGE SUMMARY ==="
total_coverage=$(go tool cover -func=coverage_filtered.out | tail -1 | awk '{print $3}')
echo "Overall Coverage: $total_coverage"

# Check for failures and exit with error code if needed
echo ""
echo "=== COVERAGE VALIDATION ==="

# Extract numeric value from total coverage (remove % sign)
total_numeric=$(echo "$total_coverage" | sed 's/%//')

# Check for low coverage items (<=50%), excluding configured exclusions
exclusion_cmd=$(build_exclusion_grep "${COVERAGE_EXCLUSIONS[@]}")
if [[ -n "$exclusion_cmd" ]]; then
    low_coverage_count=$(eval "go tool cover -func=coverage_filtered.out | awk '\$3 ~ /^[0-4]?[0-9]\\.[0-9]%\$/' | $exclusion_cmd | wc -l")
else
    low_coverage_count=$(go tool cover -func=coverage_filtered.out | awk '$3 ~ /^[0-4]?[0-9]\.[0-9]%$/' | wc -l)
fi

# Remove any leading/trailing whitespace from count
low_coverage_count=$(echo "$low_coverage_count" | xargs)

exit_code=0

# Check total coverage threshold
if (( $(echo "$total_numeric < 85" | bc -l) )); then
    echo "❌ FAILURE: Total coverage ($total_coverage) is below 85% threshold"
    exit_code=1
else
    echo "✅ PASS: Total coverage ($total_coverage) meets 85% threshold"
fi

# Check for low coverage functions (<=50%, includes 0%)
if [[ "$low_coverage_count" -gt 0 ]]; then
    echo "❌ FAILURE: Found $low_coverage_count function(s) with low coverage (<=50%)"
    exit_code=1
else
    echo "✅ PASS: No functions with low coverage (<=50%)"
fi

if [[ $exit_code -eq 0 ]]; then
    echo ""
    echo "🎉 All coverage requirements met!"
else
    echo ""
    echo "💥 Coverage requirements not met. Please improve test coverage."
fi

exit $exit_code
