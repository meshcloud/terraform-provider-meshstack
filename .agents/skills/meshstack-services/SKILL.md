---
name: meshstack-services
description: Bring up a clean local meshStack backend for acceptance testing. The backend (docker compose infra + the three meshfed-release Gradle services + the building-block runner fan-out) is brought up via the meshfed-release `local-dev-stack` skill; the acceptance suite uses that exact mux + tf + manual topology — there is no separate acceptance topology. Use before running acceptance tests, or when the backend is down / returning stale-data 409s.
---

# meshStack backend (meshfed-release) for acceptance testing

The backend lives in the **`meshfed-release`** repo (may sit at `../meshfed-release`). Hard-restarting
the docker compose infrastructure, starting the three meshfed services (meshfed-api,
block-coordinator, replicator), and starting the building-block runner fan-out — all with readiness
markers and startup pitfalls — is the **`local-dev-stack`** skill in that repo. Follow it in full;
this skill only records what is acceptance-specific.

> If services are already running on another worktree's code, do **not** restart them.

## Runner topology: the full mux + tf + manual fan-out

The acceptance suite needs the **same** multiplexer fan-out that `local-dev-stack` brings up and that
CI runs (`.github/workflows/test.yml`) — there is no separate, lighter "acceptance topology". Bring up
the standard mux + manual + tf runners per `local-dev-stack`; **both** real runners are required:

- the **`tf-block-runner`** (mux `:8300`) actually runs OpenTofu for terraform-implementation BBDs.
  The suite serves the committed bare fixture `internal/provider/testdata/tf-building-block` over git
  smart-HTTP (`git_http_server_test.go`) for the runner to clone, and asserts an end-to-end
  **sensitive-input decryption proof** (`status.outputs.api_key_echo` equals the supplied plaintext,
  in `building_block_resource_test.go`). A no-op runner cannot decrypt, so it would fail these
  subtests — do **not** try to substitute a single manual no-op runner.
- the **`manual-block-runner`** (mux `:8301`) no-ops the manual-implementation migration fixtures
  (move-from-v1 / move-from-v2), which carry no sensitive inputs.

Sensitive inputs decrypt out of the box: the dev seed registers `building-blocks.pem` (public) on the
magic runner UUID and `tf-block-runner` ships the matching private key, so meshfed's encrypt and the
runner's decrypt pair up. Readiness markers, ports, and the managed-runners API key are all in the
`local-dev-stack` skill.

## Next

Backend + the mux/manual/tf fan-out (per `local-dev-stack`) ready → run the suite (see the
**`acceptance-testing`** skill).
