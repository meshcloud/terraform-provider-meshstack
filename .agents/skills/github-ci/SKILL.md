---
name: github-ci
description: Conventions for this repo's GitHub Actions workflows — pinning actions to full SHAs, updating action versions, the build/lint/shellcheck/generate/test jobs, how the acceptance suite is requested from meshfed-release, and gotestsum coverage reporting. Use when editing .github/workflows/*.yml, bumping an action version, or debugging a CI job.
---

# GitHub Actions CI conventions

Workflows live in `.github/workflows/` (`test.yml`, `ci-request-integration.yml`, `release.yml`).
They follow the HashiCorp
[terraform-provider-scaffolding-framework](https://github.com/hashicorp/terraform-provider-scaffolding-framework)
template with adjustments: **no Terraform version matrix**, a **separate `golangci` lint job** (not
folded into build), and **OpenTofu** as the test/acceptance CLI (Terraform is used only for doc
generation).

Every job in `test.yml` runs on GitHub-hosted runners and needs neither a backend nor a credential.
The acceptance suite is the one exception, and it does not run here at all — see
[Acceptance tests run in `meshfed-release`](#acceptance-tests-run-in-meshfed-release).

## Action pinning (the main rule)

- **Pin every action to a full 40-char commit SHA**, never a mutable tag.
- **Add a version comment** after the SHA for readability.

```yaml
# Good
- uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
- uses: golangci/golangci-lint-action@1e7e51e771db61008b38414a730f564565cf7c20 # v9.2.0
# Bad — mutable tag
- uses: actions/checkout@v6
```

### Updating an action version

```bash
gh api repos/actions/checkout/releases/latest --jq '.tag_name'              # find latest tag
gh api repos/actions/checkout/git/refs/tags/v6.0.2 --jq '.object.sha'        # resolve tag → SHA
```

Then update both the SHA and the `# vX.Y.Z` comment. Use latest stable versions; check periodically.

**Adding a *new* action** (a step not yet in the workflows): resolve the **latest** release with the
two commands above and pin to that SHA. Do **not** copy a SHA/version out of a scaffolding template,
another repo, or an old example — that is how an action lands already several majors behind on the
day it is introduced.

## Jobs in `test.yml`

All gated on `build` succeeding first.

| Job | What it does |
|---|---|
| `build` | `go mod tidy` then `go build`; fails if `go mod tidy` produces a diff (commit the tidy). |
| `golangci` | Builds the `go.mod`-pinned golangci-lint with `go install`, then runs `golangci-lint-action` with `install-mode: none` and `only-new-issues: true` (annotates only changed code on PRs). On failure it prints the `task lint -- --fix` hint. |
| `generate` | `go generate` then fails on any diff — regenerate docs (`task generate`) and commit. |
| `shellcheck` | Shellcheck over `.github/scripts/*.sh`; reproduce with `task lint:shell`. |
| `test` | Unit/mock tests via gotestsum; posts coverage. Pins `TF_ACC_TERRAFORM_PATH` to a pre-installed tofu (`setup-opentofu`) — the mock tests still drive a real CLI, and auto-install races parallel exec against the download ("text file busy"). |

`permissions` are minimal per job (`contents: read`; `test` adds `pull-requests: write` for the
coverage comment, `golangci` adds `pull-requests: read` for `only-new-issues`).

**Never add `paths-ignore` to either trigger.** A workflow skipped that way never reports its
checks, so a required check on it stays "expected" forever and the pull request can never merge. A
skipped *job* reports success; a skipped *workflow* does not. Skip inside a job instead.

## Acceptance tests run in `meshfed-release`

The `TestAcc` suite needs a whole meshStack backend, private container images and a seeded database,
so it runs in the private mono repo `meshfed-release` rather than here. `ci-request-integration.yml`
only asks for that run:

- Triggers on `pull_request_target` (`opened`, `synchronize`, `reopened`) and on the push to `main`.
  The push trigger is not optional — it is what surfaces a provider/backend regression before a
  release tag.
- **It must never check out the pull request.** `pull_request_target` runs in the base repo's context
  with its secrets, so checking out contributor code there is the classic "pwn request" hole. Reading
  `github.event` and calling an API is passive use of that context and safe. The file has no
  `actions/checkout` for exactly this reason; do not add one.
- It mints a token from the satellite GitHub App (`SATELLITE_GH_APP_ID` /
  `SATELLITE_GH_APP_PRIVATE_KEY`, **organization** secrets), downscoped to `actions: write` at mint
  time, and runs `gh workflow run ci-satellite.yml --repo meshcloud/meshfed-release --ref develop`.
- `--ref develop` is deliberate, never a feature ref: the orchestration logic should be the trusted
  default-branch version even when the code under test sits on a feature branch.
- The branch name travels via `env:`, never interpolated into a `run:` line — a branch name is shell
  source.

The result comes back as a check run named exactly **`Acceptance Tests (meshStack backend)`** on the
pull request head SHA, with `details_url` pointing at the private run (visible to anyone who can see
the check). The name is a required check in this repo's ruleset, so it must not drift. The ruleset is
code, in `infra/github/main.tf`, and that module is applied by hand — it carries an apply-order note,
because the check it requires only starts reporting once `ci-request-integration.yml` is on the
default branch.

What runs there is the pull request's **merge state**, not its head alone, so a pull request does not
need rebasing before it can be verified.

**A companion backend change needs no merge-order dance any more.** Open the `meshfed-release` PR on
a branch with the **exact same, `feature/`-prefixed name** (its branch rules require the prefix) and
the lane builds that backend from source, so the pair is verified before either side merges. The
same holds for a companion change in another meshcloud Go module — see *Which revision of what the
lane tests* below.

**Fork pull requests are not supported, on purpose.** The dispatcher skips them with a notice and the
private workflow fails loudly if it is ever handed one. That single guard is the whole trust model: a
head branch inside `meshcloud/terraform-provider-meshstack` means its author has write access, so the
code under test is always trusted. A fork pull request therefore cannot merge while the check is
required — a maintainer has to adopt the branch into this repository first.

### What `meshstack-satellite.gradle` declares

`meshstack-satellite.gradle` in this repository's root is everything `meshfed-release` knows that is
specific to this repository. Its build picks this repository up as a Gradle project and runs

```bash
./gradlew :terraform-provider-meshstack:acceptanceTest
```

from the meshfed checkout, which starts the backend services and then runs `go test` here.

The file declares the `go test -run` regex and the environment the suite needs, and nothing else. No
terraform-specific string is left on the meshfed side, so changing which tests CI runs, or which
variable they need, is a change to this file and to nothing else. `onPath('tofu')` marks a value
resolved with `command -v` when the task runs, because the test framework wants a real binary path
and that path differs per machine. A variable already set in the environment wins over the file's
value.

**A repository the meshfed side lists as a satellite must ship this file.** Registration is that
list plus this file: the meshfed build skips a sibling directory that has none, so deleting or
renaming it here silently leaves the pull request with no acceptance run.

Running the suite yourself needs none of this — see the **`acceptance-testing`** skill.

### The sibling layout and the shared `go.work`

Every satellite sits **next to** the meshfed checkout, mirroring the flat `meshcloud` GitHub org. No
satellite is ever a subdirectory of the meshfed checkout. This is the layout the
`acceptance-testing` skill already assumes in the other direction, for `../meshfed-release`.

```text
<parent>/
  go.work                        # written by meshfed's ./gradlew goWork
  meshfed-release/
  terraform-provider-meshstack/  # this repository
  meshstack-cli/                 # a future satellite
```

`./gradlew goWork` in the meshfed checkout writes that one `go.work` into the parent directory, with
a `use ./<repo>` line per satellite it finds. Nothing is written inside this repository. Go searches
upwards for a `go.work`, so a plain `go build` or `go test` here then resolves every
`github.com/meshcloud/…` import to the sibling source tree rather than to the version `go.mod` pins.
`GOWORK=off` gets the pinned versions back for one command.

### Which revision of what the lane tests

The lane tests every repository **at head**, matched by branch name:

| Repository | Revision under test |
|---|---|
| this one | the pull request's merge state |
| `meshfed-release` | the branch of the same name, else `develop` |
| every other satellite (`meshstack-cli`, …) | the branch of the same name, else `main` |

So three pull requests opened under one `feature/`-prefixed branch name — mono repo, CLI, provider —
are verified as one set before any of them merges. The shared `go.work` is what makes that work
without editing a single `go.mod`: nothing is `go get`-ed and nothing is `replace`-d.

This repo's own CI is the other case. It has no `go.work`, so its build and unit tests run against
exactly the versions `go.mod` pins, which is what a release ships.

### What re-runs the suite, and what does not

The task is keyed on Go's own account of its inputs: `go list -deps -test -json ./...` names every
source, test and embedded file the suite compiles, and those files, plus `go.mod`, `go.sum`, the Go
version and the declared `run`/`environment`, are the key. So a pull request that only touches
`README.md`, `CHANGELOG.md`, `docs/`, `templates/` or `infra/` gets its check back from the cache
without a backend run. A comment-only Go edit does re-run the suite, because Go's object files carry
line numbers.

## Standard actions

| Action | Purpose |
|--------|---------|
| `actions/checkout` | Clone repo (the `test` job uses `fetch-depth: 0` for base-branch coverage comparison) |
| `actions/setup-go` | Install Go — always `go-version-file: go.mod`, the lint job included, because that Go builds the linter |
| `golangci/golangci-lint-action` | Lint + format check with inline annotations; `install-mode: none`, so it runs the golangci-lint the step before it built |
| `hashicorp/setup-terraform` | Install Terraform CLI for the `generate` job only (`terraform_wrapper: false`) |
| `opentofu/setup-opentofu` | Install OpenTofu for the `test` job; `TF_ACC_TERRAFORM_PATH` pins it (`tofu_wrapper: false`) |
| `goreleaser/goreleaser-action` | Build + release binaries (release.yml) |
| `crazy-max/ghaction-import-gpg` | Import GPG key for release signing (release.yml) |
| `actions/create-github-app-token` | Mint the downscoped token that dispatches the acceptance run (ci-request-integration.yml) |

## Coverage via gotestsum

- Tests run through [gotestsum](https://github.com/gotestyourself/gotestsum), installed as a Go
  tool dependency in `go.mod` (`tool gotest.tools/gotestsum`); invoked as `go tool gotestsum`
  (version managed by Dependabot, gomod ecosystem).
- The `test` job emits **binary coverage data** (GOCOVERDIR) into `covdata/unit`, not a text
  profile — via `-args -test.gocoverdir=…` with `-coverpkg=./...` for cross-package attribution. The
  binary form is what lets the instrumented binaries flush coverage on exit even when a test fails.
- A single PR comment, matched by an HTML marker so a re-run rewrites it instead of adding another,
  is written by `.github/scripts/coverage-comment.sh`. Safe when the job produced no data (reports
  `n/a`). Acceptance coverage is not part of that figure: that suite runs in another repository.
