# Developing the meshStack Terraform Provider

Contributor guide: how to build, run, test, and extend the provider. For the always-on rules and
the agentic-coding loop see [`AGENTS.md`](AGENTS.md); this file is the detailed how-to it points at.
End users consuming the released provider want the [`README.md`](README.md) instead.

## Prerequisites

- **Go 1.26+** (`go.mod` targets `go 1.26`).
- **[Task](https://taskfile.dev)** — the task runner for every workflow below (`Taskfile.yml`).
- A meshStack with **API credentials** for anything that talks to a backend (tests, manual runs) —
  see [Backends & authentication](#backends--authentication).

Or skip installing these: `flake.nix` provides a complete dev shell with every tool pinned (Go,
Task, golangci-lint, …). Run `nix develop` to enter it, or `nix develop --command task testacc` for
a one-off.

## Build & install

```bash
task build     # compile ./terraform-provider-meshstack
task install   # go install into $GOBIN (for dev_overrides, below)
task clean      # remove build artifacts
```

### Run a dev build from real Terraform (`dev_overrides`)

To exercise the just-built provider from your own `.tf` configs (rather than the released registry
version), point Terraform at your `$GOBIN` in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "meshcloud/meshstack"                       = "<GOBIN>"
    "registry.terraform.io/meshcloud/meshstack" = "<GOBIN>"
  }
  # Everything else still resolves from its normal registry.
  direct {}
}
```

Replace `<GOBIN>` with `go env GOBIN` (or `go env GOPATH` + `/bin`). Run `task install` after each
change. With a dev override Terraform prints a warning and **skips `terraform init`** — run
`plan`/`apply` directly. For a scoped, git-ignored playground that keeps this override out of your
global `~/.terraformrc`, use the **`scratch-config`** skill — note it wires up a *different* target
(the repo-root binary via `task build`, through a scoped `TF_CLI_CONFIG_FILE`), so don't mix the two:
this global setup points at `$GOBIN` and rebuilds with `task install`.

## Backends & authentication

The provider authenticates with a meshStack **API key** (`MESHSTACK_API_KEY` = key UUID,
`MESHSTACK_API_SECRET` = its secret) against `MESHSTACK_ENDPOINT`. Bootstrapping these is manual
today:

- **Bring your own meshStack** (default): create an API key in the meshStack you administer and set
  `MESHSTACK_ENDPOINT` / `MESHSTACK_API_KEY` / `MESHSTACK_API_SECRET`. A `.env` in the repo root is
  the convention (it is git-ignored); `set -a && source .env && set +a` exports it to child
  processes (plain `source` does not).
- **meshcloud-internal — local dev stack**: the `.env` for a local backend is reconstructible from
  the `meshfed-release` dev seed; the **`acceptance-testing`** skill documents the exact values.

> Acceptance tests are **state-independent by design**: each run creates its own resources
> (workspaces and the like) with random-suffixed names, so concurrent runs and pre-existing data
> never collide or interfere. A test-harness guard (`provider_test.go`, `DefaultTestPreCheck`)
> nonetheless pins them to `http://localhost` — not because they harm other data, but as a *cleanup*
> safety net: teardown can still fail, and a throwaway local backend can be wiped and rebuilt clean,
> whereas a shared meshStack can't. Manual/scratch runs are not guarded and may target any
> dev/sandbox meshStack you own, but **never production**.
>
> Persistent-instance coverage is complementary and lives elsewhere: the *meshcloud-internal*
> `../meshstack-smoke-tests` run against a real, persistent `meshstack-dev` — always exercising the
> provider from this repo's `main`/trunk (not a released version), to catch provider bugs before a
> release.

## Testing

All tests call `ApplyAndTest`, which auto-selects its mode:

| Mode | Trigger | Backend | Command |
|---|---|---|---|
| **Unit** | `TF_ACC` unset | in-memory mock client | `task test` |
| **Acceptance** | `TF_ACC=1` | real local meshStack | `task testacc` |

```bash
task test                          # unit (mock) tests
task test -- -run=TestValidation   # filter by name
task testacc                       # acceptance tests (needs a local backend + .env)
task testacc -- -run=BuildingBlock # filter by name
```

- Keep the two modes in **lock-step**: a step or assertion that *can* run in both *should*. Gate on
  `IsMockClientTest()` only for what the mock genuinely can't reproduce, and always say why.
- Running and debugging the acceptance suite (backend bring-up, log correlation, common failures)
  is the **`acceptance-testing`** skill.
- Reproducing a bug or a single failing test as a standalone config — or scaffolding a demo /
  working starting point — is the **`scratch-config`** skill.

### Adding a resource / data source (and its tests)

Adding or reworking a resource or data source — the implementation, example `.tf` files, the
`testconfig` builder, and a good create→update→import `TestAcc` test — is the
**`resource-development`** skill. It also owns the schema/client design conventions (meshObject
refs, DTOs, `Id`/`Uuid` naming, receivers, preview API, computed-only outputs). In short:

1. `internal/provider/<name>_resource.go` — CRUD + `Schema`.
2. `client/` — typed API client methods.
3. `provider.go` — register it.
4. `examples/resources/meshstack_<name>/` — example `.tf`.
5. `internal/provider/acctest/testconfig/build_<name>.go` — a builder.
6. `task generate` — regenerate `docs/`.
7. Update `CHANGELOG.md`.

## Lint & formatting

Lint runs **only** via `task lint` (golangci-lint v2, config in `.golangci.yml`), which also
enforces formatting — gci import ordering (stdlib → third-party → local, blank-line separated) and
gofmt. `govet` is a configured linter, so do **not** run `go vet` separately.

```bash
task lint            # golangci-lint (includes format check)
task lint -- --fix   # auto-fix formatting and fixable lint issues
```

For the occasional go1.26 `go fix` modernizer sweep, see the **`modern-go`** skill.

## Docs, changelog & CI

- **Registry docs** are generated — run `task generate` after schema changes and commit `docs/`.
- **Changelog**: every user-facing change needs a `CHANGELOG.md` entry. The top section always
  names the concrete anticipated next version (never "Unreleased") — `main` is always ready to tag.
  See the **`changelog-management`** skill.
- **Commits** follow Conventional Commits (`feat:`, `fix:`, `docs:`, `chore:`, `feat!:` for
  breaking).
- **CI/CD**: GitHub Actions pin every action to a full SHA; the acceptance job gates merge and runs
  against the last-merged `meshfed-release` backend. See the **`github-ci`** skill.

## Skills index

The on-demand procedures under `.agents/skills/` are indexed in **[`AGENTS.md`](AGENTS.md) →
"Where things live"** — the single canonical list, referenced from there rather than duplicated
here.
