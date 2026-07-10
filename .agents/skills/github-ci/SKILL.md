---
name: github-ci
description: Conventions for this repo's GitHub Actions workflows — pinning actions to full SHAs, updating action versions, the build/lint/generate/test jobs, and gotestsum coverage reporting. Use when editing .github/workflows/*.yml, bumping an action version, or debugging a CI job.
---

# GitHub Actions CI conventions

Workflows live in `.github/workflows/` (`test.yml`, `release.yml`). They follow the HashiCorp
[terraform-provider-scaffolding-framework](https://github.com/hashicorp/terraform-provider-scaffolding-framework)
template with two adjustments: **no Terraform version matrix** (single version from
`hashicorp/setup-terraform`) and a **separate `golangci` lint job** (not folded into build).

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
| `acceptance` | `TestAcc` suite on self-hosted `mesh-runners` against the full backend (meshfed `:latest` service containers). **Gates merge** (PRs and push to main). tofu comes from the nix-ci image; a truncation guard reds an incomplete run as UNKNOWN (re-run). |

`permissions` are minimal per job (`contents: read`; `test`/`acceptance` add `pull-requests: write`
for the coverage/PR comment, `golangci` adds `pull-requests: read` for `only-new-issues`).

## Companion meshfed-release changes

The `acceptance` job runs against the **last merged** meshfed-release backend — the `:latest`
service-container images, which rebuild from develop on merge. So a provider change that needs a
backend change fails acc here until the backend lands. That is the correct merge order, not a
workaround:

1. Open the companion PR in `meshfed-release` (same branch name). Its `terraform-provider-acceptance`
   job builds the backend from source, pairs this provider branch, and validates the pair.
2. Merge that PR → `:latest` rebuilds.
3. Re-run this provider PR's acceptance job → it now runs against the rebuilt backend → green → merge.

Never bypass a red acceptance check to merge provider-first: that ships a provider broken against the
released backend. On failure the PR comment links a branch-filtered meshfed-release PR search (it
404s for anyone without access to that private repo).

## Standard actions

| Action | Purpose |
|--------|---------|
| `actions/checkout` | Clone repo (the `test` job uses `fetch-depth: 0` for base-branch coverage comparison) |
| `actions/setup-go` | Install Go (`go-version-file: go.mod`, or `stable` for the lint job) |
| `golangci/golangci-lint-action` | Lint + format check with inline annotations |
| `hashicorp/setup-terraform` | Install Terraform CLI for doc generation (`terraform_wrapper: false`) |
| `goreleaser/goreleaser-action` | Build + release binaries (release.yml) |
| `crazy-max/ghaction-import-gpg` | Import GPG key for release signing (release.yml) |

## Coverage via gotestsum

- Tests run through [gotestsum](https://github.com/gotestyourself/gotestsum), installed as a Go
  tool dependency in `go.mod` (`tool gotest.tools/gotestsum`); invoked as `go tool gotestsum`
  (version managed by Dependabot, gomod ecosystem).
- Command: `go tool gotestsum --junitfile junit.xml --format testdox -- -coverprofile=coverage.out -coverpkg=./... ./...`
  — `-coverpkg=./...` gives accurate cross-package coverage.
- Coverage total + by-file detail go to `GITHUB_STEP_SUMMARY`; on PRs the summary is posted/updated
  as a comment via `gh pr comment … --edit-last` (official GitHub CLI, no third-party action).
