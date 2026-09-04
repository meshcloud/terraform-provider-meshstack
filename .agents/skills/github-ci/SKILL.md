---
name: github-ci
description: Conventions for this repo's GitHub Actions workflows — pinning actions to full SHAs, updating action versions, the build/lint/generate/test/acceptance jobs, the same-branch-name pairing with meshfed-release and meshstack-cli and its merge order, and gotestsum coverage reporting. Use when editing .github/workflows/*.yml, bumping an action version, or debugging a CI job.
---

# GitHub Actions CI conventions

Workflows live in `.github/workflows/` (`test.yml`, `release.yml`). They follow the HashiCorp
[terraform-provider-scaffolding-framework](https://github.com/hashicorp/terraform-provider-scaffolding-framework)
template with adjustments: **no Terraform version matrix**, a **separate `golangci` lint job** (not
folded into build), and **OpenTofu** as the test/acceptance CLI (Terraform is used only for doc
generation).

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
| `golangci` | `golangci-lint-action` with `only-new-issues: true` (annotates only changed code on PRs). On failure it prints the `golangci-lint run --fix` hint. |
| `generate` | `go generate` then fails on any diff — regenerate docs (`task generate`) and commit. |
| `test` | Unit/mock tests via gotestsum; posts coverage. Pins `TF_ACC_TERRAFORM_PATH` to a pre-installed tofu (`setup-opentofu`) — the mock tests still drive a real CLI, and auto-install races parallel exec against the download ("text file busy"). |
| `acceptance` | `TestAcc` suite on self-hosted `mesh-runners` against the full backend (meshfed `:latest` service containers). **Gates merge** (PRs and push to main). tofu comes from the nix-ci image; a truncation guard reds an incomplete run as UNKNOWN (re-run). Also builds against a same-named `meshstack-cli` branch when one exists (below). |

`permissions` are minimal per job (`contents: read`; `test`/`acceptance` add `pull-requests: write`
for the coverage/PR comment, `golangci` adds `pull-requests: read` for `only-new-issues`).

## Companion changes in meshfed-release and meshstack-cli

Three repos pair **by exact branch name** — this provider, `meshfed-release` (the backend) and
`meshstack-cli` (the shared API client). A branch called `BD-1234-thing` in any of them is picked up
by the others' CI, so a change spanning two or three repos is validated as a set before any side
merges. Name the branches identically from the start: no repo puts a rule on the name, and the names
only have to match, character for character.

A few names in `meshfed-release` are already taken. `master` and `develop` are the shared bases,
`finish-patch-release.yml` reads the release version out of a `patch/*` name, and the owners of
`dependabot/*` and `gh-readonly-queue/*` force-push and delete under those prefixes. Pick anything
else.

> Known gap: this provider's branch is `feature/scaffold-cli` and the companion `meshstack-cli`
> branch is `feat/scaffold-cli-and-move-client`. The two names differ, so the `acceptance` job finds
> no matching CLI branch and builds against the version `go.mod` pins instead. Renaming a branch
> with an open PR closes that PR, so this is a fact to live with until that branch merges, not
> something to fix.

### meshfed-release (the backend)

The `acceptance` job runs against the **last merged** meshfed-release backend — the `:latest`
service-container images, which rebuild from develop on merge. So a provider change that needs a
backend change fails acc here until the backend lands. That is the correct merge order, not a
workaround:

1. Open the companion PR in `meshfed-release` on a branch with the **exact same name**.
   meshfed-release's `terraform-provider-acceptance` job checks out the provider branch matching its
   own, builds the backend from source, and runs this repo's acceptance suite against backend +
   provider changes **combined**.
2. Merge that PR → `:latest` rebuilds.
3. Re-run this provider PR's acceptance job → it now runs against the rebuilt backend → green → merge.

Never bypass a red acceptance check to merge provider-first: that ships a provider broken against the
released backend. On failure the PR comment links a branch-filtered meshfed-release PR search (it
404s for anyone without access to that private repo).

### meshstack-cli (the shared client)

`go.mod` pins meshstack-cli to a released (today: pseudo-) version, so a provider change needing a new
client method would otherwise wait for a merge and a re-pin. Instead the `acceptance` job resolves a
same-named meshstack-cli branch (`git ls-remote --exit-code --heads`), checks it out into
`./meshstack-cli`, and writes a `go.work` pointing at it — `go.mod`/`go.sum` are never modified, so the
`build` job's `go mod tidy` diff guard keeps its meaning. **With no matching branch it changes
nothing** and the pinned version stands; there is deliberately no fallback to the CLI's default branch
(unlike the meshfed-release rule above) because falling back would mask a stale pin — and `main` in
meshstack-cli is currently an empty initial commit that would not even compile.

Merge order is the same shape: merge the CLI PR, re-pin `go.mod` to the new version, then merge here.
`task cli:link` / `task cli:unlink` are the local equivalent against `../meshstack-cli`.

## Standard actions

| Action | Purpose |
|--------|---------|
| `actions/checkout` | Clone repo (the `test` job uses `fetch-depth: 0` for base-branch coverage comparison) |
| `actions/setup-go` | Install Go (`go-version-file: go.mod`, or `stable` for the lint job) |
| `golangci/golangci-lint-action` | Lint + format check with inline annotations |
| `hashicorp/setup-terraform` | Install Terraform CLI for the `generate` job only (`terraform_wrapper: false`) |
| `opentofu/setup-opentofu` | Install OpenTofu for the `test` job; `TF_ACC_TERRAFORM_PATH` pins it (`tofu_wrapper: false`) |
| `goreleaser/goreleaser-action` | Build + release binaries (release.yml) |
| `crazy-max/ghaction-import-gpg` | Import GPG key for release signing (release.yml) |

## Coverage via gotestsum

- Tests run through [gotestsum](https://github.com/gotestyourself/gotestsum), installed as a Go
  tool dependency in `go.mod` (`tool gotest.tools/gotestsum`); invoked as `go tool gotestsum`
  (version managed by Dependabot, gomod ecosystem).
- Both test jobs emit **binary coverage data** (GOCOVERDIR), not a text profile — via
  `-args -test.gocoverdir=…` with `-coverpkg=./...` for cross-package attribution: the `test` job
  writes `covdata/unit`, the `acceptance` job writes `covdata/acc`.
- `covdata/unit` is uploaded as an artifact so the `acceptance` job (a different, self-hosted runner)
  downloads it and merges both with `go tool covdata` into one figure.
- A single PR comment (matched by an HTML marker) is written by `.github/scripts/coverage-comment.sh`
  in two stages: `unit` (Go Test job — unit figure, combined pending) then `combined` (acceptance job
  — rewrites it with the merged figure). Safe when a job produced no data (reports `n/a`).
