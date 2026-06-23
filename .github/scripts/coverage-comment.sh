#!/usr/bin/env bash
#
# Render/update the single coverage comment on a pull request (matched by an HTML marker so both
# CI jobs target the same comment). Used by .github/workflows/test.yml in two stages:
#
#   coverage-comment.sh unit       (Go Test job)      post unit coverage; combined row pending
#   coverage-comment.sh combined   (Acceptance job)   merge unit + acceptance coverage and rewrite
#
# Coverage is read from binary coverage-data directories (GOCOVERDIR format) so unit and acceptance
# data — produced on different runners and carried between them as an artifact — can be merged with
# `go tool covdata merge`. Expected layout (relative to the repo checkout / working dir):
#
#   covdata/unit   unit (mock client) coverage data
#   covdata/acc    acceptance coverage data (combined stage only)
#
# Required env: REPO (owner/name), PR (pull request number), GH_TOKEN (for gh). Must run from the
# module root so `go tool cover -func` can resolve package paths. Safe to run when no coverage data
# was produced (e.g. tests failed before flushing) — it reports "n/a" rather than erroring.
set -euo pipefail

MODE="${1:?usage: coverage-comment.sh <unit|combined>}"
MARKER='<!-- coverage-report -->'

# True when a coverage-data dir actually contains data.
have_data() { ls "$1"/covmeta.* >/dev/null 2>&1; }

# total_percent <text-profile> -> total coverage percentage (e.g. "56.3%").
total_percent() { go tool cover -func="$1" | tail -1 | awk '{print $NF}'; }

# Up to 10 zero-coverage functions from a text profile. awk caps the output itself (instead of
# `| head -10`) so it never receives SIGPIPE from an early-closing reader — under `set -o pipefail`
# that would otherwise fail the step with exit 141.
top_uncovered() { awk '$3 == "0.0%" { print; if (++n == 10) exit }' "$1"; }

# Unit figure (shown in both stages); computed from the unit data dir when present.
UNIT_CELL='`n/a`'
if have_data covdata/unit; then
  go tool covdata textfmt -i=covdata/unit -o=unit.txt
  UNIT_CELL="\`$(total_percent unit.txt)\`"
fi

case "$MODE" in
  unit)
    COMBINED_CELL='⏳ acceptance job running…'
    if have_data covdata/unit; then
      UNCOVERED=$(top_uncovered unit.txt)
    else
      UNIT_CELL='`n/a (no coverage produced)`'
      UNCOVERED=""
    fi
    DETAILS="Uncovered functions (unit run)"
    NOTE="_Coverage is collected across all packages (\`-coverpkg=./...\`). The combined figure is filled in once the acceptance job finishes._"
    ;;
  combined)
    # Merge whichever coverage-data dirs actually have data.
    inputs=""
    have_data covdata/unit && inputs="covdata/unit"
    have_data covdata/acc && inputs="${inputs:+$inputs,}covdata/acc"
    if [ -n "$inputs" ]; then
      mkdir -p covdata/merged
      go tool covdata merge -i="$inputs" -o=covdata/merged
      go tool covdata textfmt -i=covdata/merged -o=merged.txt
      go tool cover -func=merged.txt > merged-summary.txt
      COMBINED_CELL="\`$(total_percent merged.txt)\`"
      UNCOVERED=$(top_uncovered merged-summary.txt)
    else
      COMBINED_CELL='`n/a (no coverage produced)`'
      UNCOVERED=""
    fi
    DETAILS="Uncovered functions (combined run)"
    NOTE="_Combined with \`go tool covdata merge\` over the unit and acceptance coverage data. The acceptance suite runs \`./internal/provider\` against the live backend; coverage is attributed across all packages (\`-coverpkg=./...\`)._"
    ;;
  *)
    echo "unknown mode: $MODE (expected 'unit' or 'combined')" >&2
    exit 2
    ;;
esac

[ -n "$UNCOVERED" ] || UNCOVERED="(none — every function has some coverage)"

# Acceptance status annotation for the combined row (combined stage only). The workflow passes in
# ACC_COMPLETE (did the run reach gotestsum's summary?) and ACC_OUTCOME (pass/fail). An incomplete
# run is a self-hosted-runner truncation — its result is unknown, so it must never read as a pass.
ACC_TAG=""
ACC_NOTE=""
if [ "$MODE" = combined ]; then
  case "${ACC_COMPLETE:-}" in
    false)
      ACC_TAG=" — 🛑 acceptance incomplete"
      ACC_NOTE="🛑 **The acceptance run did not complete** (the self-hosted runner truncated or killed it before gotestsum reported a summary). The combined figure above therefore omits acceptance and the acceptance result is **unknown — not a pass**; re-run the job."
      ;;
    true)
      case "${ACC_OUTCOME:-}" in
        success) ACC_TAG=" — ✅ acceptance passed" ;;
        failure)
          ACC_TAG=" — ❌ acceptance failed"
          ACC_NOTE="❌ **Acceptance tests failed** (advisory — the job runs against the last *merged* meshfed-release backend, so a change needing a companion backend PR fails here until that backend merges; the real gate is meshfed-release's \`terraform-provider-acceptance\` job)."
          ;;
      esac
      ;;
  esac
fi

{
  echo "$MARKER"
  echo "## 📊 Test Coverage"
  echo ""
  echo "| Scope | Coverage |"
  echo "| --- | --- |"
  echo "| Unit tests (mock client) | ${UNIT_CELL} |"
  echo "| Combined (unit + acceptance) | ${COMBINED_CELL}${ACC_TAG} |"
  echo ""
  [ -z "$ACC_NOTE" ] || { echo "$ACC_NOTE"; echo ""; }
  echo "<details><summary>${DETAILS}</summary>"
  echo ""
  echo '```'
  echo "$UNCOVERED"
  echo '```'
  echo "</details>"
  echo ""
  echo "$NOTE"
} > coverage.md

[ -z "${GITHUB_STEP_SUMMARY:-}" ] || cat coverage.md >> "$GITHUB_STEP_SUMMARY"

# Rewrite the marked comment if it exists, otherwise create it.
cid=$(gh api "repos/$REPO/issues/$PR/comments" --paginate \
  --jq "[.[] | select(.body | contains(\"$MARKER\"))] | last | .id" 2>/dev/null || true)
if [ -n "$cid" ] && [ "$cid" != "null" ]; then
  gh api -X PATCH "repos/$REPO/issues/comments/$cid" -F body=@coverage.md >/dev/null
  echo "updated coverage comment #$cid ($MODE)"
else
  gh pr comment "$PR" --body-file coverage.md
  echo "created coverage comment ($MODE)"
fi
