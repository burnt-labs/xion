#!/bin/bash

# Script to run tests with coverage, excluding generated protobuf files

echo "Running tests with coverage (excluding .pb.go files)..."

# Run tests with coverage
go test ./x/... -coverprofile=coverage.out

# Filter out .pb.go and .pb.gw.go files from coverage report
grep -v "\.pb\.go:" coverage.out | grep -v "\.pb\.gw\.go:" > coverage_filtered.out

# Show coverage report without .pb.go files
echo "Coverage report (excluding generated files):"
go tool cover -func=coverage_filtered.out

# Generate HTML report without .pb.go files
go tool cover -html=coverage_filtered.out -o coverage.html

echo "HTML coverage report generated: coverage.html"
echo "Filtered coverage file: coverage_filtered.out"

# Show summary statistics
echo ""
echo "=== COVERAGE SUMMARY ==="
total_coverage=$(go tool cover -func=coverage_filtered.out | tail -1 | awk '{print $3}')
echo "Overall Coverage: $total_coverage"

# Show modules with 0% coverage
echo ""
echo "=== MODULES WITH 0% COVERAGE ==="
go tool cover -func=coverage_filtered.out | grep "0.0%" | grep -v "100.0%"

# Show modules with low coverage (less than 50%)
echo ""
echo "=== MODULES WITH LOW COVERAGE (<50%) ==="
go tool cover -func=coverage_filtered.out | awk '$3 ~ /^[0-4][0-9]\.[0-9]%$/ || $3 ~ /^[0-9]\.[0-9]%$/'
