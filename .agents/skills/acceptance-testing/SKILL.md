---
name: acceptance-testing
description: Run and debug the meshStack provider acceptance tests (TF_ACC=1) against a local backend. Use when asked to run acceptance tests, investigate acceptance-test failures, or correlate provider errors with backend behavior. Assumes the backend is already up — bring it up first with the meshstack-services skill.
---

# Running & investigating acceptance tests

These tests run against a real local meshStack backend. **Prerequisite:** the backend must be
up. First bring up a clean backend using the **`meshstack-services`** skill — this skill does
not duplicate the startup steps.

Run all commands here from the `terraform-provider-meshstack/` directory.

## 1. Run the acceptance tests

```bash
set -a && source .env && set +a   # exports MESHSTACK_ENDPOINT (http://localhost:8080), MESHSTACK_API_KEY, MESHSTACK_API_SECRET
: > /tmp/acc-tests.log
nohup bash -c "TF_ACC=1 go test -count=1 ./internal/provider/ -parallel 8 -timeout 300s -v > /tmp/acc-tests.log 2>&1" &
```

- `set -a && source .env && set +a` exports the `.env` vars to the `go test` child (plain
  `source` does not export them).
- Run in the **background** and poll `/tmp/acc-tests.log` with a hard cap (~2–3 min) — never
  block on a possibly-hung run. Individual tests complete in seconds; the full suite finishes
  well under the timeout. If it hangs or hits the 300s timeout, a backend service is likely
  down — check its log immediately rather than waiting.
- Target a single test: `-run 'TestAccBuildingBlock$/<subtest>'`.
- Always tee/redirect output to `/tmp` and **read the log** to investigate — do not re-run the
  suite piped through `grep`.
- `ApplyAndTest` sets `MESHSTACK_SKIP_VERSION_CHECK` so the provider's anticipated-next-release
  version floor can run ahead of the local `develop` backend — expected, no action needed.

## 2. Investigate

```bash
# Pass/fail/skip summary
grep -E -- '--- (PASS|FAIL|SKIP)' /tmp/acc-tests.log
```

- **Grep for `panic:` first.** A panic aborts the whole test binary before Go prints any
  `--- FAIL` summaries, so "0 FAIL" is misleading.
- **Correlate provider errors with backend behavior** in `/tmp/meshstack-api.log`: mark the
  line count before a run, then `tail -n +<mark>` after to see only that run's output.

**Common failure causes:**

| Symptom                          | Likely cause                                            |
|----------------------------------|---------------------------------------------------------|
| BB stuck `PENDING`               | block-coordinator or manual-block-runner not running, or the manual runner started with the wrong `RUNNER_UUID` (must match `SharedBuildingBlockRunnerUuid`, `98520496-…`) |
| Tenant delete `400`              | replicator not running, or mandatory BBs still pending  |
| `409 BuildingBlockConflict` on delete | BB not in a final state                            |
| `409 Conflict` (other)           | stale data from a previous run (tag definitions, etc.)  |
| `422` on bindings                | groups/users referenced in examples don't exist locally |
| `400` on a request              | request body / serialization mismatch                   |
