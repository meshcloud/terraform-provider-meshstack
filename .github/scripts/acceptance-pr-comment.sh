#!/usr/bin/env bash
#
# Maintain the single acceptance advisory comment on a pull request (matched by an HTML marker, so the
# job updates one comment instead of appending). Used by .github/workflows/test.yml:
#
#   acceptance-pr-comment.sh
#
# It posts/updates a warning when the acceptance run failed or did not complete, and deletes the
# comment once acceptance goes green. A cancelled or skipped run leaves any existing comment untouched.
#
# Required env:
#   GH_TOKEN     token for gh
#   REPO         owner/name
#   PR           pull request number
#   OUTCOME      outcome of the acceptance test step (success|failure|cancelled|skipped)
#   COMPLETE     "true" when gotestsum reported a summary, i.e. the run was not truncated
#   SERVER_URL   github.server_url
#   RUN_ID       github.run_id
#   RUN_ATTEMPT  github.run_attempt
#   HEAD_REF     github.head_ref
# Optional env:
#   RUNNER_NAME  set by the Actions runner; used to resolve this job's URL
set -euo pipefail

MARKER='<!-- acceptance-advisory -->'

# Link straight to THIS acceptance job, not the run: the run page shows green on purpose (the acc step
# is continue-on-error), so the failing/truncated step is only visible inside the job. Resolve the
# job's html_url by matching the ephemeral runner that ran it; fall back to the run URL if the lookup
# fails.
RUN_URL="$SERVER_URL/$REPO/actions/runs/$RUN_ID"
JOB_URL=$(gh api "repos/$REPO/actions/runs/$RUN_ID/attempts/$RUN_ATTEMPT/jobs" \
  --jq ".jobs[] | select(.runner_name == \"${RUNNER_NAME:-}\") | .html_url" 2>/dev/null | head -n1 || true)
[ -n "$JOB_URL" ] || JOB_URL="$RUN_URL"

# Best-effort link to a same-branch companion PR in the (private) meshfed-release repo. This public
# repo's token cannot query that repo, so we build a branch-filtered PR search URL by convention rather
# than confirming one exists — it resolves for authorized users and 404s for everyone else.
MESHFED_PR_SEARCH="$SERVER_URL/meshcloud/meshfed-release/pulls?q=is%3Apr+head%3A${HEAD_REF}"

existing=$(gh api "repos/$REPO/issues/$PR/comments" --paginate \
  --jq ".[] | select(.body | startswith(\"$MARKER\")) | .id" | head -n1 || true)

upsert() { # $1 = body
  if [ -n "$existing" ]; then
    gh api -X PATCH "repos/$REPO/issues/comments/$existing" -f body="$1" >/dev/null
    echo "updated advisory comment $existing"
  else
    gh api -X POST "repos/$REPO/issues/$PR/comments" -f body="$1" >/dev/null
    echo "posted advisory comment"
  fi
}

# Incomplete run (truncated/killed by the runner): the result is unknown, so surface it loudly and
# never delete an existing warning by mistaking it for a pass.
if [ "$COMPLETE" != "true" ]; then
  upsert "$(printf '%s\n%s\n\n%s' "$MARKER" \
    "🛑 **The acceptance run did not complete.** The self-hosted runner truncated or killed the test process before gotestsum reported a summary, so the result is **unknown — not a pass**. Re-run the job; if it recurs it is a runner-infra issue." \
    "🔗 [View the acceptance job]($JOB_URL)")"
  exit 0
fi

# Completed run — act only on a definitive pass/fail; ignore cancelled/skipped.
case "$OUTCOME" in
success)
  if [ -n "$existing" ]; then
    gh api -X DELETE "repos/$REPO/issues/comments/$existing"
    echo "acceptance green — deleted advisory comment $existing"
  else
    echo "acceptance green — no advisory comment to delete"
  fi
  ;;
failure)
  upsert "$(printf '%s\n%s\n\n%s' "$MARKER" \
    "❌ **Acceptance tests failed — this blocks merge.** The job runs against the last *merged* meshfed-release backend (\`:latest\`). If this change needs a companion backend change, get its meshfed-release PR green and **merged first** (that rebuilds \`:latest\`), then **re-run this job**; otherwise it is a regression to fix. Measured coverage for this run is in the coverage comment above." \
    "🔗 [Failing acceptance job]($JOB_URL) · [companion meshfed-release PR — same branch]($MESHFED_PR_SEARCH) (private; 404 without access)")"
  ;;
*)
  echo "acceptance outcome '$OUTCOME' — leaving any existing comment untouched"
  ;;
esac
