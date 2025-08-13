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

# Show modules with ok coverage (less than 80% - 99%)
echo ""
echo "=== OK COVERAGE (80% - 99%) ==="
go tool cover -func=coverage_filtered.out | awk '$3 ~ /^[8-9][0-9]\.[0-9]%$/'

# Show modules with low coverage (less than 80%)
echo ""
echo "=== LOW COVERAGE (<80%) ==="
go tool cover -func=coverage_filtered.out | awk '$3 ~ /^[0-7]?[0-9]\.[0-9]%$/' | grep -v -E "[^0-9]0.0%" 

# Show modules with 0% coverage
echo ""
echo "=== NO COVERAGE (0%) ==="
go tool cover -func=coverage_filtered.out | grep -E "[^0-9]0.0%" 

# Show summary statistics
echo ""
echo "=== COVERAGE SUMMARY ==="
total_coverage=$(go tool cover -func=coverage_filtered.out | tail -1 | awk '{print $3}')
echo "Overall Coverage: $total_coverage"
