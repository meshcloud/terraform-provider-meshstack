#!/usr/bin/env bash
#
# Render/update the single coverage comment on a pull request, matched by an HTML marker so a re-run
# rewrites the same comment instead of adding another. Used by the `test` job in
# .github/workflows/test.yml.
#
# Coverage is read from a binary coverage-data directory (GOCOVERDIR format). Expected layout,
# relative to the repo checkout / working dir:
#
#   covdata/unit   unit (mock client) coverage data
#
# Required env: REPO (owner/name), PR (pull request number), GH_TOKEN (for gh). Must run from the
# module root so `go tool cover -func` can resolve package paths. Safe to run when no coverage data
# was produced (e.g. tests failed before flushing) — it reports "n/a" rather than erroring.
set -euo pipefail

MARKER='<!-- coverage-report -->'

# True when a coverage-data dir actually contains data.
have_data() { ls "$1"/covmeta.* >/dev/null 2>&1; }

# total_percent <text-profile> -> total coverage percentage (e.g. "56.3%").
total_percent() { go tool cover -func="$1" | tail -1 | awk '{print $NF}'; }

# Up to 10 zero-coverage functions from a text profile. awk caps the output itself (instead of
# `| head -10`) so it never receives SIGPIPE from an early-closing reader — under `set -o pipefail`
# that would otherwise fail the step with exit 141.
top_uncovered() { awk '$3 == "0.0%" { print; if (++n == 10) exit }' "$1"; }

# The backticks in the cell below are a markdown code span, not a shell expansion.
# shellcheck disable=SC2016
UNIT_CELL='`n/a (no coverage produced)`'
UNCOVERED=""
if have_data covdata/unit; then
  go tool covdata textfmt -i=covdata/unit -o=unit.txt
  UNIT_CELL="\`$(total_percent unit.txt)\`"
  UNCOVERED=$(top_uncovered unit.txt)
fi

[ -n "$UNCOVERED" ] || UNCOVERED="(none — every function has some coverage)"

{
  echo "$MARKER"
  echo "## 📊 Test Coverage"
  echo ""
  echo "| Scope | Coverage |"
  echo "| --- | --- |"
  echo "| Unit tests (mock client) | ${UNIT_CELL} |"
  echo ""
  echo "<details><summary>Uncovered functions (unit run)</summary>"
  echo ""
  echo '```'
  echo "$UNCOVERED"
  echo '```'
  echo "</details>"
  echo ""
  echo "_Coverage is collected across all packages (\`-coverpkg=./...\`). The acceptance suite runs against a real meshStack backend in \`meshfed-release\` and reports separately, as the \"Acceptance Tests (meshStack backend)\" check._"
} > coverage.md

[ -z "${GITHUB_STEP_SUMMARY:-}" ] || cat coverage.md >> "$GITHUB_STEP_SUMMARY"

# Rewrite the marked comment if it exists, otherwise create it.
cid=$(gh api "repos/$REPO/issues/$PR/comments" --paginate \
  --jq "[.[] | select(.body | contains(\"$MARKER\"))] | last | .id" 2>/dev/null || true)
if [ -n "$cid" ] && [ "$cid" != "null" ]; then
  gh api -X PATCH "repos/$REPO/issues/comments/$cid" -F body=@coverage.md >/dev/null
  echo "updated coverage comment #$cid"
else
  gh pr comment "$PR" --body-file coverage.md
  echo "created coverage comment"
fi
