---
name: meshstack-services
description: Bring up a clean local meshStack backend for acceptance testing — hard-restart the meshfed-release infrastructure (docker compose), start the three meshfed-release Gradle services (meshfed-api, block-coordinator, replicator) plus the manual block runner from the separate building-block-runner repo, and verify readiness via logs. Use before running acceptance tests, or when the backend is down / returning stale-data 409s. (The skill is named meshstack-services; the backend repo keeps its legacy name "meshfed-release".)
---

# meshStack backend (meshfed-release) for acceptance testing

Stand up a clean meshStack backend suitable for acceptance testing. Most of it lives in the
`meshfed-release` repo (may sit relatively at `../meshfed-release`); the manual block runner has
been extracted into a separate `building-block-runner` repo (may sit at `../building-block-runner`).
Do not assume absolute paths. Unless noted otherwise, run infrastructure and meshfed-release
service commands from the `meshfed-release/` directory.

> If services are already running on another worktree's code, do **not** restart them — that
> would disrupt an in-progress session.

## 1. Hard restart infrastructure

```bash
cd meshfed-release/
docker compose down && docker compose up -d
```

- **Volumes:** only keycloak is named/persistent. mariadb & ravendb are ephemeral → recreated
  clean on `down`+`up`. A clean DB avoids stale-data `409` conflicts from previous runs.
  mariadb re-seeds via `./ci/backend/container/initdb.d`.
- **Wait for readiness, bounded** — don't block indefinitely:
  - ravendb: healthcheck reports `healthy`
  - mariadb: log shows `ready for connections` / init process done

## 2. Start the services in the background

The `:*:start` / `:*:bootRun` Gradle tasks run Spring Boot apps and **block forever** — never
await them. Use `nohup … &` and verify readiness via the logs.

### 2a. Three meshfed-release services (from `meshfed-release/`)

```bash
: > /tmp/meshstack-api.log; : > /tmp/block-coordinator.log; : > /tmp/replicator.log
nohup ./gradlew :meshfed:meshfed-api:start --console=plain > /tmp/meshstack-api.log 2>&1 &
nohup ./gradlew :buildingblocks:block-coordinator-api:start --console=plain > /tmp/block-coordinator.log 2>&1 &
nohup ./gradlew :meshfed:replicator:replicator-api:start --console=plain > /tmp/replicator.log 2>&1 &
```

### 2b. Manual block runner (from the `building-block-runner` repo)

The manual block runner has been **moved out of `meshfed-release`** into the separate
`building-block-runner` repo (it may sit relatively at `../building-block-runner`); the old
`meshfed-release` `:buildingblocks:manual-block-runner:start` task no longer exists. Start the
runner from the `building-block-runner` repo.

```bash
cd ../building-block-runner   # extracted from meshfed-release
: > /tmp/manual-runner.log
RUNNER_UUID=98520496-627d-43e6-82da-ce499179ff3f \
  nohup ./gradlew :manual-block-runner:bootRun --console=plain > /tmp/manual-runner.log 2>&1 &
```

- **`RUNNER_UUID` is critical.** The runner only picks up runs assigned to its own UUID. It
  **must** equal the provider's `SharedBuildingBlockRunnerUuid`
  (`98520496-627d-43e6-82da-ce499179ff3f`, see `internal/provider/building_block_runner.go`),
  which all manual BBD examples/tests default to. The repo's built-in default
  (`46b7c17a-…`) does **not** match — using it leaves manual building blocks stuck at `PENDING`.
- Other config defaults are fine locally: it polls `RUNNER_API_URL` (default
  `http://localhost:8080`) every ~10s with basic auth `RUNNER_API_USERNAME`/`RUNNER_API_PASSWORD`
  (default `bb-api`/`guest`). Override via env or a `runner-config.yml` only if your local
  setup differs.

> The **manual** runner is what acceptance testing uses — it no-ops both manual and
> terraform-implementation BBDs (echoes inputs as outputs). Keep it for the acceptance suite.
> To instead run a *terraform-implementation* BBD end-to-end in `scratch/` (real `git clone` +
> OpenTofu) — only for ad-hoc play, never the acceptance suite — swap this manual runner for the
> `tf-block-runner`; see the optional section in the **`scratch-config-testing`** skill.

**Readiness markers** (grep the logs; ~60–120s on first build):

| Service           | Log file                    | Marker                                     | Port |
|-------------------|-----------------------------|--------------------------------------------|------|
| meshfed-api       | `/tmp/meshstack-api.log`    | `Started ApplicationKt`                    | 8080 |
| block-coordinator | `/tmp/block-coordinator.log`| `Started BlockCoordinatorApiApplicationKt` | 8083 |
| replicator        | `/tmp/replicator.log`       | `Started ApplicationKt`                     | —    |
| manual-runner     | `/tmp/manual-runner.log`    | `Started BlockRunnerApplication`           | — (no HTTP; polls every ~10s) |

Watch for `BUILD FAILED` / exceptions in the logs rather than waiting for a timeout.

### Startup pitfalls

- **Corrupted Kotlin incremental cache** fails meshfed-api with `Storage corrupted …/lookups.tab_i`,
  cascading to `Unresolved reference 'meshobjects'`. Fix and restart that service:
  ```bash
  rm -rf core/meshobjects/build/kotlin core/rest/build/kotlin
  ```
- Transient okhttp errors in `manual-runner.log` during startup-before-infra-ready are benign.

## Next

Once all four services (three meshfed-release + the manual runner) report ready, run the
suite — see the **`acceptance-testing`** skill.
