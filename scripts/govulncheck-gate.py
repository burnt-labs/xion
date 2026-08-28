#!/usr/bin/env python3
"""Run govulncheck over a module and gate the build on reachable findings.

govulncheck reports at three levels of confidence. This script keeps them
apart, because treating them alike is what makes dependency scanning noise:

  reachable  a vulnerable symbol is called from this module's code
  imported   a vulnerable package is imported, but no vulnerable symbol is called
  required   a vulnerable module is in the graph, but none of its code is linked

Only reachable findings gate the build, and only those not listed in the
allowlist. Everything else is reported so it stays visible without failing
anyone's pull request.

An advisory carrying no symbol information is reported as reachable by
govulncheck as soon as the package is imported. Those land in the allowlist
with a note rather than being special-cased here -- the distinction is a
judgement about a specific advisory, not something to infer.
"""

import argparse
import json
import os
import subprocess
import sys


def parse_stream(raw):
    """govulncheck -format json emits concatenated objects, not JSON lines."""
    decoder = json.JSONDecoder()
    objects = []
    i = 0
    while i < len(raw):
        while i < len(raw) and raw[i] in " \n\r\t":
            i += 1
        if i >= len(raw):
            break
        obj, i = decoder.raw_decode(raw, i)
        objects.append(obj)
    return objects


def load_allowlist(path):
    allowed = {}
    if not os.path.exists(path):
        return allowed
    with open(path) as handle:
        for line in handle:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            osv, _, reason = line.partition("#")
            allowed[osv.strip()] = reason.strip()
    return allowed


def classify(objects):
    """Bucket every finding by how strong the evidence of exposure is."""
    summaries = {}
    fixes = {}
    buckets = {"reachable": set(), "imported": set(), "required": set()}
    for obj in objects:
        osv = obj.get("osv")
        if isinstance(osv, dict):
            summaries[osv["id"]] = osv.get("summary", "")
        finding = obj.get("finding")
        if not finding:
            continue
        osv_id = finding["osv"]
        trace = finding.get("trace") or []
        frame = trace[0] if trace else {}
        if frame.get("function"):
            level = "reachable"
        elif frame.get("package"):
            level = "imported"
        else:
            level = "required"
        buckets[level].add(osv_id)
        if finding.get("fixed_version"):
            fixes[osv_id] = finding["fixed_version"]
    # A finding surfaces once per level; the strongest evidence wins.
    buckets["imported"] -= buckets["reachable"]
    buckets["required"] -= buckets["reachable"] | buckets["imported"]
    return buckets, summaries, fixes


def render(name, buckets, summaries, fixes, allowed, blocking):
    def table(ids):
        lines = ["| Advisory | Fixed in | Summary |", "| --- | --- | --- |"]
        for osv in sorted(ids):
            summary = summaries.get(osv, "").replace("|", "\\|")
            lines.append(f"| [{osv}](https://pkg.go.dev/vuln/{osv}) "
                         f"| {fixes.get(osv, 'no fix')} | {summary} |")
        return "\n".join(lines)

    out = [f"## govulncheck — {name}", ""]
    if blocking:
        out += [f"### Reachable and not allowlisted ({len(blocking)})", "",
                "A vulnerable symbol is called from this module. Either upgrade "
                "the dependency, or add the advisory to the allowlist with a "
                "reason if the exposure is genuinely acceptable.", "",
                table(blocking), ""]
    else:
        out += ["### Reachable and not allowlisted (0)", "",
                "No new reachable advisories.", ""]

    accepted = buckets["reachable"] & set(allowed)
    if accepted:
        out += [f"<details><summary>Allowlisted, reachable "
                f"({len(accepted)})</summary>", "", table(accepted),
                "", "</details>", ""]
    for level, blurb in (("imported", "vulnerable package imported, no "
                                      "vulnerable symbol called"),
                         ("required", "vulnerable module in the graph, none "
                                      "of its code linked")):
        if buckets[level]:
            out += [f"<details><summary>{level.title()} — {blurb} "
                    f"({len(buckets[level])})</summary>", "",
                    table(buckets[level]), "", "</details>", ""]
    return "\n".join(out)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--dir", default=".", help="module directory to scan")
    parser.add_argument("--name", required=True, help="label used in reports")
    parser.add_argument("--allowlist", required=True)
    parser.add_argument("--test", action="store_true",
                        help="include test files, needed for test-only modules")
    parser.add_argument("--report-only", action="store_true",
                        help="print the report without failing on new findings")
    args = parser.parse_args()

    command = ["govulncheck", "-scan", "symbol", "-format", "json"]
    if args.test:
        command.append("-test")
    command.append("./...")

    result = subprocess.run(command, cwd=args.dir, capture_output=True,
                            text=True)
    # govulncheck exits 3 when it finds something, which is not an error here.
    if result.returncode not in (0, 3):
        sys.stderr.write(result.stderr)
        return result.returncode or 1

    buckets, summaries, fixes = classify(parse_stream(result.stdout))
    allowed = load_allowlist(args.allowlist)
    blocking = sorted(buckets["reachable"] - set(allowed))

    report = render(args.name, buckets, summaries, fixes, allowed, blocking)
    print(report)
    summary_path = os.environ.get("GITHUB_STEP_SUMMARY")
    if summary_path:
        with open(summary_path, "a") as handle:
            handle.write(report + "\n")

    if blocking and not args.report_only:
        sys.stderr.write(
            f"\n{args.name}: {len(blocking)} reachable advisory(ies) not in "
            f"{args.allowlist}: {', '.join(blocking)}\n")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
