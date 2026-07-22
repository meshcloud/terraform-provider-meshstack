---
name: acceptance-testing
description: Run and debug the meshStack provider acceptance tests (TF_ACC=1) against a local backend, including bringing that backend up. Use when asked to run acceptance tests, investigate acceptance-test failures, or correlate provider errors with backend behavior.
---

# Running & investigating acceptance tests

These tests run against a real local meshStack backend. Bring it up first (next section), then run
all commands here from the `terraform-provider-meshstack/` directory.

## Backend bring-up

The backend lives in the sibling **`meshfed-release`** repo (assume `../meshfed-release`); its
**`local-dev-stack`** skill brings up the full stack (docker infra + `multiplexing-block-runner`
mux, the three meshfed services, and the runner fan-out). Follow it in full; don't restart services
already running for another worktree.

The suite needs that **full mux + tf + manual fan-out** — there is no lighter "acceptance topology".
Both real runners are required:

- **`tf-block-runner`** (`:8300`) runs OpenTofu for terraform BBDs. The suite serves the bare fixture
  `internal/provider/testdata/tf-building-block` over git smart-HTTP (`git_http_server_test.go`) for
  it to clone, and asserts end-to-end sensitive-input decryption (`status.outputs.api_key_echo` ==
  supplied plaintext). A no-op runner can't decrypt and fails those subtests.
- **`manual-block-runner`** (`:8301`) no-ops the manual migration fixtures (move-from-v1/-v2).

Sensitive inputs decrypt out of the box: the dev seed pairs `building-blocks.pem` (public, on the
magic runner UUID) with the private key shipped in `tf-block-runner`.

## The `.env` (no need to hunt for it)

`.env` is reconstructible from the dev seed — don't go searching other worktrees for a copy. Set
`TF_ACC=1` and `MESHSTACK_ENDPOINT=http://localhost:8080`; `MESHSTACK_API_KEY` (the key uuid) and
`MESHSTACK_API_SECRET` (its `Secret.Raw`) are the `mkGlobalApiKey "terraform-provider-acceptance" …`
entry under `auth.openid.apiKeys` in `../meshfed-release/meshfed/api/src/main/resources/application-default.dhall`.

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

## 3. Mock vs. acceptance: keep the runs in lock-step

`ApplyAndTest` runs against the in-memory mock (`TF_ACC` unset) or the real backend (set);
`IsMockClientTest()` reports which. The mock guards the acceptance flow, so both modes should run
the **same** steps wherever they can.

- If a step or assertion CAN run in both modes, it SHOULD — a mock run that mirrors acceptance is a
  better regression guard than one that skips half the flow. Mock compute is cheap; divergence isn't.
- Only gate on `IsMockClientTest()` for what the mock genuinely can't reproduce: backend-only
  validations/errors, real provisioning runs, defaults the backend materializes (e.g. an operator
  input on upgrade), cross-workspace permission boundaries.
- Prefer a whole-subtest skip (`if IsMockClientTest() { t.Skip(...) }`, see `08`/`11`) over a
  per-step `SkipFunc`, so each `ApplyAndTest` runs identically in both modes. Gate a single
  `ConfigStateCheck` only when just one assertion diverges (see `05`/`06`).
- Every gate MUST say what the mock can't do and why. An unexplained gate is a bug.
