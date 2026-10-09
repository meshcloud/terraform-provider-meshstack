---
name: acceptance-testing
description: Run and debug the meshStack provider acceptance tests (TF_ACC=1) against a local backend. Use when asked to run acceptance tests, investigate acceptance-test failures, or correlate provider errors with backend behavior.
---

# Acceptance tests

The suite runs against a local meshStack backend. Run every command here from the repository root.

## In CI

A pull request here does not run the suite itself. `.github/workflows/test-acceptance.yml` asks the
*meshcloud-internal* `meshfed-release` repo for the run, which reports back as the required check
`Acceptance Tests (meshStack backend)` on the pull request's merge state. That run reads only
`meshstack-satellite.gradle` from this repo: the `TestAcc` filter, the services the suite needs and
its environment. To change what CI runs, change that file. The rest of the lane is documented in
`meshfed-release`'s `AGENTS.md` → "Satellite Acceptance Tests".

A fork pull request gets no run: a maintainer must adopt its branch here before it can merge.

## Why local only

Every test creates its own resources with random-suffixed names, so runs never collide and need no
particular database state. Keep it so: never hardcode a name that a parallel run could clash on.

`requireLocalMeshStack` (`provider_test.go`) pins the suite to `http://localhost`, so a failed
teardown is fixed by rebuilding the local backend and never touches a shared meshStack.

## Backend

*meshcloud-internal:* bring up the stack, runner included, with the `local-dev-stack` skill in
`../meshfed-release`. Leave alone any service that another worktree already runs.

The terraform tests clone `internal/provider/testdata/tf-building-block` over `file://`, so the
runner must run on the same filesystem as the tests. They also check that the runner decrypts a
sensitive input, which the module echoes as `status.outputs.api_key_echo`.

## `.env`

Rebuild `.env` from the dev seed: `TF_ACC=1`, `MESHSTACK_ENDPOINT=http://localhost:8080`, and as
`MESHSTACK_API_KEY` / `MESHSTACK_API_SECRET` the key uuid and `Secret.Raw` of the
`mkGlobalApiKey "terraform-provider-acceptance" …` entry in
`../meshfed-release/meshfed/api/src/main/resources/application-default.dhall`.

## Run

```bash
set -a && source .env && set +a
: > /tmp/acc-tests.log
nohup bash -c "TF_ACC=1 go test -count=1 ./internal/provider/ -parallel 8 -timeout 300s -v > /tmp/acc-tests.log 2>&1" &
```

- Run in the background and poll `/tmp/acc-tests.log` for at most 2–3 minutes. A hang or the 300s
  timeout usually means a backend service is down; check its log right away.
- One subtest: `-run 'TestAccBuildingBlock$/<subtest>'`.
- Read the log to investigate; do not re-run the suite piped through `grep`.
- `ApplyAndTest` sets `MESHSTACK_SKIP_VERSION_CHECK`, because the provider's version floor can be
  ahead of the local `develop` backend.

## Investigate

```bash
grep -E -- '--- (PASS|FAIL|SKIP)' /tmp/acc-tests.log
```

- **Grep for `panic:` first.** A panic aborts the test binary before Go prints any `--- FAIL`.
- Correlate provider errors with `/tmp/meshstack-api.log`: note its line count before a run, then
  `tail -n +<mark>` after it.

| Symptom                               | Likely cause                                            |
|---------------------------------------|---------------------------------------------------------|
| BB stuck `PENDING`                    | the runner or block-coordinator is not running          |
| Tenant delete `400`                   | replicator not running, or mandatory BBs still pending  |
| `409 BuildingBlockConflict` on delete | BB not in a final state                                 |
| `409 Conflict` (other)                | stale data from a previous run (tag definitions, etc.)  |
| `422` on bindings                     | groups/users referenced in examples don't exist locally |
| `400` on a request                    | request body / serialization mismatch                   |

## Mock and acceptance run the same steps

`ApplyAndTest` runs against the in-memory mock (`TF_ACC` unset) or the real backend (set);
`IsMockClientTest()` reports which. The mock guards the acceptance flow, so both modes run the same
steps wherever they can.

- Gate on `IsMockClientTest()` only what the mock cannot reproduce: backend-only validations and
  errors, real runs, defaults the backend materializes (e.g. an operator input on upgrade),
  cross-workspace permission boundaries.
- Prefer skipping a whole subtest (`if IsMockClientTest() { t.Skip(...) }`, see `08`/`11`) over a
  per-step `SkipFunc`, so each `ApplyAndTest` runs identically in both modes. Gate a single
  `ConfigStateCheck` only when just one assertion diverges (see `05`/`06`).
- Every gate says what the mock cannot do. An unexplained gate is a bug.
