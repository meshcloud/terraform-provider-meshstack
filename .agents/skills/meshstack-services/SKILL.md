---
name: meshstack-services
description: Bring up a clean local meshStack backend for acceptance testing. The backend (docker compose infra + the three meshfed-release Gradle services) is brought up via the meshfed-release `local-dev-stack` skill; this skill adds only the acceptance-test-specific part — run a single manual block runner so terraform-implementation BBDs are no-op'd. Use before running acceptance tests, or when the backend is down / returning stale-data 409s.
---

# meshStack backend (meshfed-release) for acceptance testing

The backend lives in the **`meshfed-release`** repo (may sit at `../meshfed-release`). Hard-restarting
the docker compose infrastructure and starting the three meshfed services (meshfed-api,
block-coordinator, replicator) — with readiness markers and startup pitfalls — is the
**`local-dev-stack`** skill in that repo. Follow its infra + services steps, then return here for the
acceptance-test runner.

> If services are already running on another worktree's code, do **not** restart them.

## Runner: a single manual block runner (acceptance topology)

The acceptance suite needs the **manual** runner to claim **all** runs and no-op them — it echoes
inputs as outputs for *both* manual and terraform-implementation BBDs, so the suite never executes
real OpenTofu. Run **only** the manual runner, polling meshfed directly as the shared runner UUID —
**not** the multiplexer fan-out from `local-dev-stack`, which routes terraform-implementation BBDs to
the real `tf-block-runner` (that would actually run tofu).

```bash
cd ../building-block-runner    # the runner was extracted out of meshfed-release
: > /tmp/manual-runner.log
RUNNER_UUID=98520496-627d-43e6-82da-ce499179ff3f \
  RUNNER_API_CLIENT_ID=<local managed-runners client id> \
  RUNNER_API_CLIENT_SECRET=<local managed-runners secret> \
  nohup ./gradlew :manual-block-runner:bootRun --console=plain > /tmp/manual-runner.log 2>&1 &
```

- **`RUNNER_UUID` is critical** — it must equal the provider's `SharedBuildingBlockRunnerUuid`
  (`98520496-627d-43e6-82da-ce499179ff3f`, see `internal/provider/building_block_runner.go`), which
  all BBD examples/tests default to. The runner's built-in default (`46b7c17a-…`) does **not** match
  and leaves building blocks stuck at `PENDING`.
- **Auth: prefer the API key.** Authenticate with the local managed-runners API key via
  `RUNNER_API_CLIENT_ID` / `RUNNER_API_CLIENT_SECRET`. The seeded local dev values are in the
  meshfed-release `local-dev-stack` skill (kept out of this public repo). The legacy basic auth
  `RUNNER_API_USERNAME` / `RUNNER_API_PASSWORD` still works as a deprecated fallback, but the API key
  takes precedence when both are set.

Ready marker: `Started BlockRunnerApplication` in `/tmp/manual-runner.log` (no HTTP port; polls ~10s).

## Next

Backend (per `local-dev-stack`) + this manual runner ready → run the suite (see the
**`acceptance-testing`** skill).
