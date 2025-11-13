#!/bin/bash
# Production-ready script to find corrupted contracts on XION
# Usage: ./find_corrupted_contracts.sh [LIMIT] [OUTPUT_FILE]

set -euxo pipefail

# Configuration
NODE="${NODE:-https://rpc.xion-testnet-2.burnt.com:443}"
LIMIT="${1:-1000}"
OUTPUT_FILE="${2:-corrupted_contracts.txt}"
QUERY_TIMEOUT=10

echo "======================================"
echo "XION Corrupted Contract Scanner"
echo "======================================"
echo "Node: $NODE"
echo "Limit: $LIMIT contracts"
echo "Output: $OUTPUT_FILE"
echo "Query timeout: ${QUERY_TIMEOUT}s"
echo ""

# Check dependencies
if ! command -v xiond &> /dev/null; then
  echo "❌ Error: xiond not found. Please install xiond."
  exit 1
fi

if ! command -v jq &> /dev/null; then
  echo "❌ Error: jq not found. Please install jq."
  exit 1
fi

# Clear output file
> "$OUTPUT_FILE"
echo "# XION Corrupted Contracts Report" >> "$OUTPUT_FILE"
echo "# Generated: $(date)" >> "$OUTPUT_FILE"
echo "# Node: $NODE" >> "$OUTPUT_FILE"
echo "" >> "$OUTPUT_FILE"

# Counters
total_checked=0
total_corrupted=0
wire_type_corruption=0
field_ordering_bugs=0
truncated_data=0
timeouts=0
other_errors=0

# Get all code IDs
echo "[1/3] Fetching code IDs..."
code_ids=$(xiond query wasm list-code --node "$NODE" --output json 2>/dev/null \
  | jq -r '.code_infos[].code_id' 2>/dev/null)

if [ -z "$code_ids" ]; then
  echo "❌ Error: Could not fetch code IDs from node"
  exit 1
fi

num_codes=$(echo "$code_ids" | wc -l | tr -d ' ')
echo "Found $num_codes code IDs"
echo ""

# Collect and check contracts
echo "[2/3] Scanning contracts..."
echo ""

for code_id in $code_ids; do
  if [ $total_checked -ge $LIMIT ]; then
    break
  fi

  # Fetch contracts for this code (limit to what we need)
  remaining=$((LIMIT - total_checked))
  contracts=$(xiond query wasm list-contract-by-code "$code_id" \
    --node "$NODE" \
    --limit "$remaining" \
    --output json 2>/dev/null \
    | jq -r '.contracts[]?' 2>/dev/null)

  if [ -z "$contracts" ]; then
    continue
  fi

  # Check each contract
  while IFS= read -r contract; do
    if [ -z "$contract" ]; then
      continue
    fi
    if [ $total_checked -ge $LIMIT ]; then
      break
    fi

    ((total_checked++))

    # Show progress every 10 contracts
    if [ $((total_checked % 10)) -eq 0 ]; then
      echo "Progress: $total_checked/$LIMIT | Corrupted: $total_corrupted (WT:$wire_type_corruption FO:$field_ordering_bugs TD:$truncated_data TO:$timeouts Other:$other_errors)"
    fi

    # Query ContractInfo
    output=$(timeout $QUERY_TIMEOUT xiond query wasm contract "$contract" --node "$NODE" --output json 2>&1)
    exit_code=$?

    # Check result
    if [ $exit_code -eq 124 ]; then
      # Timeout
      ((timeouts++))
      ((total_corrupted++))
      echo "TIMEOUT: $contract" >> "$OUTPUT_FILE"

    elif [ $exit_code -ne 0 ]; then
      # Error - classify it
      ((total_corrupted++))

      if echo "$output" | grep -qi "wireType 7\|wireType 6"; then
        # Invalid wire type = data corruption
        ((wire_type_corruption++))
        wire_type=$(echo "$output" | grep -oE "wireType [0-9]" | head -1)
        echo "WIRE_TYPE_CORRUPTION: $contract ($wire_type)" >> "$OUTPUT_FILE"

      elif echo "$output" | grep -qi "illegal wireType"; then
        # Valid wire type but illegal = field ordering bug
        ((field_ordering_bugs++))
        echo "FIELD_ORDERING_BUG: $contract" >> "$OUTPUT_FILE"

      elif echo "$output" | grep -qi "EOF\|unexpected\|truncated"; then
        # Truncated/malformed data
        ((truncated_data++))
        echo "TRUNCATED_DATA: $contract" >> "$OUTPUT_FILE"

      else
        # Other error
        ((other_errors++))
        error_msg=$(echo "$output" | grep -i "error" | head -1 | cut -c 1-100)
        echo "OTHER_ERROR: $contract | $error_msg" >> "$OUTPUT_FILE"
      fi
    fi

  done <<< "$contracts"
done

echo ""
echo "[3/3] Generating report..."
echo ""

# Final summary
cat >> "$OUTPUT_FILE" << EOF

================================================================================
SUMMARY
================================================================================
Total Contracts Checked: $total_checked
Total Corrupted: $total_corrupted ($(awk "BEGIN {printf \"%.2f\", ($total_corrupted/$total_checked)*100}")%)

Breakdown by Type:
  - Wire Type Corruption:  $wire_type_corruption (invalid wire types 6/7)
  - Field Ordering Bugs:   $field_ordering_bugs (should be fixed by v24)
  - Truncated Data:        $truncated_data (incomplete protobuf data)
  - Query Timeouts:        $timeouts (node didn't respond in ${QUERY_TIMEOUT}s)
  - Other Errors:          $other_errors (various unmarshal failures)

================================================================================
ANALYSIS
================================================================================
EOF

if [ $wire_type_corruption -gt 0 ]; then
  cat >> "$OUTPUT_FILE" << EOF

⚠️  WIRE TYPE CORRUPTION ($wire_type_corruption contracts):
   - These have INVALID protobuf wire types (6 or 7)
   - This is DATA CORRUPTION, not field ordering
   - Cannot be fixed by v24 migration
   - Requires manual intervention or redeployment
   - Contract state may still be accessible
EOF
fi

if [ $field_ordering_bugs -gt 0 ]; then
  cat >> "$OUTPUT_FILE" << EOF

⚠️  FIELD ORDERING BUGS ($field_ordering_bugs contracts):
   - These have valid wire types but wrong field positions
   - SHOULD have been fixed by v24 migration
   - If v24 ran successfully, this is unexpected
   - May indicate migration detection bug
EOF
fi

if [ $truncated_data -gt 0 ]; then
  cat >> "$OUTPUT_FILE" << EOF

⚠️  TRUNCATED DATA ($truncated_data contracts):
   - ContractInfo data is incomplete
   - Possible storage corruption or failed writes
   - Cannot be fixed by field swapping
   - Requires restoration from backup or redeployment
EOF
fi

# Print summary to console
echo "======================================"
echo "SCAN COMPLETE"
echo "======================================"
echo "Checked: $total_checked contracts"
echo "Corrupted: $total_corrupted ($(awk "BEGIN {printf \"%.2f\", ($total_corrupted/$total_checked)*100}")%)"
echo ""
echo "By Type:"
echo "  Wire Type Corruption: $wire_type_corruption"
echo "  Field Ordering Bugs:  $field_ordering_bugs"
echo "  Truncated Data:       $truncated_data"
echo "  Timeouts:             $timeouts"
echo "  Other Errors:         $other_errors"
echo ""
echo "Full report saved to: $OUTPUT_FILE"
echo "======================================"

exit 0
