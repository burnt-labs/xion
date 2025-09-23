#!/bin/bash

# Script to run tests with coverage, excluding generated protobuf files

# List of directory patterns to exclude from testing (modify as needed)
# Examples: "./devnet/..." "./integration_tests/..." "./build/..."
TEST_EXCLUSIONS=(
    # Add directory exclusions here as needed
    # Example: "./devnet/..."
    # Example: "./integration_tests/..."
)

# List of patterns to exclude from low coverage reporting
# Add new exclusion patterns here as needed
COVERAGE_EXCLUSIONS=(
    "x/feeabs/types/params.go.*Validate"
    "github.com/burnt-labs/xion/x/xion/client/cli/tx.go:.*NewRegisterCmd"
    "github.com/burnt-labs/xion/x/xion/client/cli/tx.go:.*NewSignCmd"
    "github.com/burnt-labs/xion/x/xion/keeper/grpc_query.go:.*WebAuthNVerifyRegister"
    "github.com/burnt-labs/xion/x/xion/keeper/grpc_query.go:.*WebAuthNVerifyAuthenticate"
    "github.com/burnt-labs/xion/x/xion/keeper/mint.go:.*StakedInflationMintFn"
    # Add more exclusion patterns below as needed
    # Example: "x/module/types/file.go.*FunctionName"
    # Example: ".*defensive.*code.*pattern"
)

echo "Running tests with coverage on all directories..."

# If no exclusions are specified, run all tests
if [[ ${#TEST_EXCLUSIONS[@]} -eq 0 || "${TEST_EXCLUSIONS[0]}" =~ ^[[:space:]]*# ]]; then
    echo "No exclusions specified. Testing all packages..."
    go test ./... -coverprofile=coverage.out
else
    echo "Building list of packages to test (excluding specified directories)..."
    for exclusion in "${TEST_EXCLUSIONS[@]}"; do
        if [[ -n "$exclusion" && "$exclusion" != \#* ]]; then  # Skip empty lines and comments
            echo "Excluding from tests: $exclusion"
        fi
    done
    
    all_packages=$(go list ./...)
    test_packages=""
    
    for package in $all_packages; do
        should_exclude=false
        for exclusion in "${TEST_EXCLUSIONS[@]}"; do
            if [[ -n "$exclusion" && "$exclusion" != \#* ]]; then
                # Convert exclusion pattern to package pattern
                exclusion_pattern=$(echo "$exclusion" | sed 's|\./||' | sed 's|/\.\.\.$||')
                if [[ "$package" =~ github\.com/burnt-labs/xion/$exclusion_pattern ]]; then
                    should_exclude=true
                    break
                fi
            fi
        done
        
        if [[ "$should_exclude" == false ]]; then
            if [[ -z "$test_packages" ]]; then
                test_packages="$package"
            else
                test_packages="$test_packages $package"
            fi
        fi
    done
    
    echo "Testing packages: $test_packages"
    go test $test_packages -coverprofile=coverage.out
fi
test_exit_code=$?
if [[ $test_exit_code -ne 0 ]]; then
    echo ""
    echo "=== TEST FAILURE DETECTED ==="
    echo "❌ FAILURE: One or more tests failed (exit code $test_exit_code). Aborting coverage analysis."
    echo "(Tests were run on all packages except those in the exclusion list)"
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
echo "=== OK COVERAGE (50% - 99%) ==="
go tool cover -func=coverage_filtered.out | awk '$3 ~ /^[5-9][0-9]\.[0-9]%$/'

# Show modules with low coverage (less than 50%)
echo ""
echo "=== LOW COVERAGE (<50%) ==="
exclusion_cmd=$(build_exclusion_grep "${COVERAGE_EXCLUSIONS[@]}")
if [[ -n "$exclusion_cmd" ]]; then
    eval "go tool cover -func=coverage_filtered.out | awk '\$3 ~ /^[0-4]?[0-9]\\.[0-9]%\$/' | grep -v -E \"[^0-9]0.0%\" | $exclusion_cmd"
else
    go tool cover -func=coverage_filtered.out | awk '$3 ~ /^[0-4]?[0-9]\.[0-9]%$/' | grep -v -E "[^0-9]0.0%"
fi 

# Show modules with 0% coverage
echo ""
echo "=== NO COVERAGE (0%) ==="
go tool cover -func=coverage_filtered.out | grep -E "[^0-9]0.0%" 

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

# Check for low coverage items (excluding configured exclusions)
exclusion_cmd=$(build_exclusion_grep "${COVERAGE_EXCLUSIONS[@]}")
if [[ -n "$exclusion_cmd" ]]; then
    low_coverage_count=$(eval "go tool cover -func=coverage_filtered.out | awk '\$3 ~ /^[0-4]?[0-9]\\.[0-9]%\$/' | grep -v -E \"[^0-9]0.0%\" | $exclusion_cmd | wc -l")
else
    low_coverage_count=$(go tool cover -func=coverage_filtered.out | awk '$3 ~ /^[0-4]?[0-9]\.[0-9]%$/' | grep -v -E "[^0-9]0.0%" | wc -l)
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

# Check for low coverage functions
if [[ "$low_coverage_count" -gt 0 ]]; then
    echo "❌ FAILURE: Found $low_coverage_count function(s) with low coverage (<50%)"
    exit_code=1
else
    echo "✅ PASS: No functions with low coverage (<50%)"
fi

if [[ $exit_code -eq 0 ]]; then
    echo ""
    echo "🎉 All coverage requirements met!"
else
    echo ""
    echo "💥 Coverage requirements not met. Please improve test coverage."
fi

exit $exit_code
