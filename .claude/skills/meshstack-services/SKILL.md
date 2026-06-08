---
name: meshstack-services
description: Bring up a clean local meshStack backend for acceptance testing — hard-restart the meshfed-release infrastructure (docker compose), start the four Gradle Spring Boot services in the background, and verify readiness via logs. Use before running acceptance tests, or when the backend is down / returning stale-data 409s. (The skill is named meshstack-services; the backend repo keeps its legacy name "meshfed-release".)
---

# meshStack backend (meshfed-release) for acceptance testing

Stand up a clean meshStack backend suitable for acceptance testing. The backend lives in the
`meshfed-release` repo — it may sit relatively at `../meshfed-release`; do not assume an
absolute path. Run all commands in this skill from the `meshfed-release/` directory unless
noted otherwise.

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

## 2. Start the 4 services in the background

The `:*:start` Gradle tasks run Spring Boot apps and **block forever** — never await them.
Use `nohup … &` and verify readiness via the logs.

```bash
: > /tmp/meshstack-api.log; : > /tmp/block-coordinator.log; : > /tmp/manual-runner.log; : > /tmp/replicator.log
nohup ./gradlew :meshfed:meshfed-api:start --console=plain > /tmp/meshstack-api.log 2>&1 &
nohup ./gradlew :buildingblocks:block-coordinator-api:start --console=plain > /tmp/block-coordinator.log 2>&1 &
nohup ./gradlew :buildingblocks:manual-block-runner:start --console=plain > /tmp/manual-runner.log 2>&1 &
nohup ./gradlew :meshfed:replicator:replicator-api:start --console=plain > /tmp/replicator.log 2>&1 &
```

**Readiness markers** (grep the logs; ~60–120s on first build):

| Service           | Log file                    | Marker                                  | Port  |
|-------------------|-----------------------------|-----------------------------------------|-------|
| meshfed-api       | `/tmp/meshstack-api.log`    | `Started ApplicationKt`                 | 8080  |
| block-coordinator | `/tmp/block-coordinator.log`| `Started BlockCoordinatorApiApplicationKt` | 8083 |
| manual-runner     | `/tmp/manual-runner.log`    | `Started BlockRunnerApplicationKt`      | 8180  |
| replicator        | `/tmp/replicator.log`       | `Started ApplicationKt`                 | —     |

Watch for `BUILD FAILED` / exceptions in the logs rather than waiting for a timeout.

### Startup pitfalls

- **Corrupted Kotlin incremental cache** fails meshfed-api with `Storage corrupted …/lookups.tab_i`,
  cascading to `Unresolved reference 'meshobjects'`. Fix and restart that service:
  ```bash
  rm -rf core/meshobjects/build/kotlin core/rest/build/kotlin
  ```
- Transient okhttp errors in `manual-runner.log` during startup-before-infra-ready are benign.

## Next

Once all four services report ready, run the suite — see the **`acceptance-testing`** skill.
