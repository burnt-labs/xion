#!/usr/bin/env bash
set -Eeuo pipefail

# Run govulncheck over a module and gate the build on reachable findings.
#
# govulncheck reports at three levels of confidence. This script keeps them
# apart, because treating them alike is what makes dependency scanning noise:
#
#   reachable  a vulnerable symbol is called from this module's code
#   imported   a vulnerable package is imported, but no vulnerable symbol runs
#   required   a vulnerable module is in the graph, but none of its code links
#
# Only reachable findings gate the build, and only those absent from the
# allowlist. Everything else is reported so it stays visible without failing
# anyone's pull request.
#
# An advisory carrying no symbol data is reported as reachable by govulncheck
# as soon as the package is imported. Those go in the allowlist with a note
# rather than being special-cased here -- whether that matters is a judgement
# about a specific advisory, not something to infer.

usage() {
	cat >&2 <<'EOF'
usage: govulncheck-gate.sh --name NAME --allowlist FILE [--dir DIR] [--test] [--report-only]

  --name         label used in the report
  --allowlist    file of GO-YYYY-NNNN identifiers, # comments ignored
  --dir          module directory to scan (default: .)
  --test         include test files, needed for test-only modules
  --report-only  print the report without failing on new findings
EOF
	exit 2
}

DIR="."
NAME=""
ALLOWLIST=""
scan=(-scan symbol -format json)
REPORT_ONLY=""

while [[ $# -gt 0 ]]; do
	case "$1" in
	--dir) DIR="${2:-}"; shift 2 ;;
	--name) NAME="${2:-}"; shift 2 ;;
	--allowlist) ALLOWLIST="${2:-}"; shift 2 ;;
	--test) scan+=(-test); shift ;;
	--report-only) REPORT_ONLY="1"; shift ;;
	*) usage ;;
	esac
done

[[ -n "$NAME" && -n "$ALLOWLIST" ]] || usage

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT
findings="$workdir/findings.json"

# govulncheck exits 3 when it finds something, which is not an error here.
status=0
govulncheck -C "$DIR" "${scan[@]}" ./... >"$findings" || status=$?
if [[ $status -ne 0 && $status -ne 3 ]]; then
	echo "govulncheck failed for $NAME (exit $status)" >&2
	exit "$status"
fi

allow='[]'
if [[ -f "$ALLOWLIST" ]]; then
	allow=$({ grep -oE '^GO-[0-9]{4}-[0-9]+' "$ALLOWLIST" || true; } |
		jq -R -s 'split("\n") | map(select(length > 0))')
fi

# govulncheck -format json emits concatenated objects, which jq reads as a
# stream; -s collects them so findings can be cross-referenced with summaries.
read -r -d '' program <<'JQEOF' || true
def level:
  (.trace[0] // {}) as $frame
  | if ($frame | has("function")) then "reachable"
    elif ($frame | has("package")) then "imported"
    else "required" end;

def table($ids; $fixed; $summary):
  ["| Advisory | Fixed in | Summary |", "| --- | --- | --- |"]
  + ($ids | sort | map(
      "| [\(.)](https://pkg.go.dev/vuln/\(.)) | \($fixed[.] // "no fix") "
      + "| \(($summary[.] // "") | gsub("\\|"; "\\\\|")) |"))
  | join("\n");

def details($title; $ids; $fixed; $summary):
  if ($ids | length) == 0 then []
  else
    ["<details><summary>\($title) (\($ids | length))</summary>", "",
     table($ids; $fixed; $summary), "", "</details>", ""]
  end;

(map(select(has("osv")) | {(.osv.id): (.osv.summary // "")}) | add // {}) as $summary
| (map(select(has("finding")) | .finding)) as $findings
| ($findings | map(select(.fixed_version != null) | {(.osv): .fixed_version}) | add // {}) as $fixed
| ($findings | map(select(level == "reachable") | .osv) | unique) as $reachable
| (($findings | map(select(level == "imported") | .osv) | unique) - $reachable) as $imported
| (($findings | map(select(level == "required") | .osv) | unique) - $reachable - $imported) as $required
| ($reachable - $allow) as $blocking
| ($reachable - $blocking) as $accepted
| {
    blocking: $blocking,
    report: ([
      "## govulncheck — \($name)", "",
      "### Reachable and not allowlisted (\($blocking | length))", ""
    ]
    + (if ($blocking | length) == 0 then ["No new reachable advisories.", ""]
       else
         ["A vulnerable symbol is called from this module. Either upgrade the "
          + "dependency, or add the advisory to the allowlist with a reason if "
          + "the exposure is genuinely acceptable.", "",
          table($blocking; $fixed; $summary), ""]
       end)
    + details("Allowlisted, reachable"; $accepted; $fixed; $summary)
    + details("Imported — vulnerable package imported, no vulnerable symbol called";
              $imported; $fixed; $summary)
    + details("Required — vulnerable module in the graph, none of its code linked";
              $required; $fixed; $summary)
    | join("\n"))
  }
JQEOF

result=$(jq -s --argjson allow "$allow" --arg name "$NAME" "$program" <"$findings")

report=$(jq -r '.report' <<<"$result")
printf '%s\n' "$report"
if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
	printf '%s\n' "$report" >>"$GITHUB_STEP_SUMMARY"
fi

blocking=$(jq -r '.blocking | join(", ")' <<<"$result")
if [[ -n "$blocking" && -z "$REPORT_ONLY" ]]; then
	count=$(jq -r '.blocking | length' <<<"$result")
	echo >&2
	echo "$NAME: $count reachable advisory(ies) not in $ALLOWLIST: $blocking" >&2
	exit 1
fi
