# AGENTS.md — meshStack Terraform Provider

<role>
You are an expert Go engineer working on the meshStack Terraform Provider — the official provider
for managing meshStack resources via the meshObject API (`/api/meshobjects`), built on
[terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework) v1. This file
is the always-on source of truth for both AI agents and humans; detailed procedures live in
`.agents/skills/` and in [`DEVELOPMENT.md`](DEVELOPMENT.md).
</role>

> **This repository is public.** Write everything here so an external contributor with no meshcloud
> access can follow it. meshcloud-internal shortcuts — chiefly the private sibling repo
> `../meshfed-release` (the backend, its dev seed, and house-wide skills) — are genuinely useful but
> **optional**. Tag them clearly as internal, and never let understanding a rule *depend* on them.

## What you build

meshStack resources exposed as Terraform **resources** and **data sources**. Every meshObject shares
the same shape: `api_version`, `kind` (e.g. `meshProject`), `metadata` (name, uuid, timestamps),
`spec` (user config), `status` (system-managed). Adding or reworking one — implementation, example
`.tf`, `testconfig` builder, and a create→update→import acceptance test — is the
**`resource-development`** skill, which also owns the schema/client design conventions
(meshObject refs, DTOs, `Id`/`Uuid` naming, receivers, preview API, computed-only outputs).

## The agentic coding loop (IaC against meshStack)

Iterating on the provider means driving real Terraform against a real meshStack:

1. **Change** the resource/data source and its client — **`resource-development`**.
2. **Build** the dev provider and point Terraform at it via `dev_overrides` — [`DEVELOPMENT.md`](DEVELOPMENT.md).
3. **Reproduce, verify, or prototype** as a standalone config with the **`scratch-config`** skill,
   against **any meshStack you hold API credentials for** — a local dev stack *or* a remote
   dev/sandbox. Beyond debugging, it scaffolds demos and working starting points; a config that
   graduates out of git-ignored `scratch/` should be moved out, documented, and committed.
4. **Test** in both modes — unit (in-memory mock) and acceptance (real backend). Acceptance tests
   are **state-independent** (each creates its own resources with random-suffixed names — no
   collisions or interference), but a guard pins them to `http://localhost` so a failed cleanup is
   fixed by rebuilding the local backend DB, never by touching a shared meshStack. See
   [`DEVELOPMENT.md`](DEVELOPMENT.md) and the **`acceptance-testing`** skill.

<rules id="iterative-coding">
For any non-trivial change, **stress-test the plan before writing code**: have the design grilled —
walk each branch of the decision tree, resolve dependencies one at a time, and settle every open
question with a recommended answer. Catching a wrong turn at the plan stage is far cheaper than
after the code and tests exist. (meshcloud-internal: the `grill-me` / `grill-with-docs` skills in
`../meshfed-release/.agents/skills/` are the canonical procedure — but the practice stands on its
own and needs no private access.)
</rules>

## Authentication

The provider needs three env vars — `MESHSTACK_ENDPOINT`, `MESHSTACK_API_KEY`, and
`MESHSTACK_API_SECRET`. Bootstrapping them is **manual**; the full setup (what each var is, the
git-ignored `.env` convention, the *meshcloud-internal* dev-seed shortcut, and the never-target-prod
warning) lives once in [`DEVELOPMENT.md`](DEVELOPMENT.md) → Backends & authentication.

## Always-on rules

<rules id="always-on">

- **Lean comments.** A comment earns its place only by saying what the code cannot — the *why*, a
  trade-off, a non-obvious constraint. Don't restate what a name, type, or signature already conveys;
  prefer one sharp line over a paragraph. (*meshcloud-internal* deep-dive: meshfed-release
  `PRINCIPLES.md`.)
- **Lint only via `task lint`** (golangci-lint, which also enforces gci import ordering + gofmt). It
  already runs `govet`, so **do not run `go vet` separately**. Auto-fix with `task lint -- --fix`.
- **Changelog for user-facing changes.** Every feature, fix, or breaking change needs a
  `CHANGELOG.md` entry under the concrete anticipated next version — never an "Unreleased" heading,
  because `main` is always ready to tag. See **`changelog-management`**.
- **Conventional Commits** for messages (`feat:`, `fix:`, `docs:`, `chore:`, `feat!:` for breaking).

</rules>

## Authoring instructions & skills

When you add or edit a skill, a prompt, or this file, follow the *meshcloud-internal* **`write-a-skill`**
and **`writing-instructions`** skills in `../meshfed-release/.agents/skills/`: be specific and
direct, give the rationale behind each rule so it generalises, prefer positive instructions (say what
to do), and wrap examples in `<example>` tags. They are the house authority; agentic work does not
require them.

## Where things live

- [`DEVELOPMENT.md`](DEVELOPMENT.md) — the contributor how-to: build, `dev_overrides`, auth, unit +
  acceptance testing, lint, docs, CI.
- `.agents/skills/` — on-demand procedures loaded by name:

| Skill | Use for |
|---|---|
| `resource-development` | Add/rework a resource or data source + tests; schema/client design conventions |
| `modern-go` | Go 1.26 idioms (`new(expr)`, generics), the pointer/`omitempty` rule, the `go fix` sweep |
| `acceptance-testing` | Local backend bring-up; run & debug the acceptance suite |
| `scratch-config` | Standalone repro/debug/prototype against any meshStack you own |
| `changelog-management` | Pick the next version, maintain `CHANGELOG.md` |
| `github-ci` | GitHub Actions conventions, action SHA-pinning |

## Key directories

- `internal/provider/` — provider implementation (`provider.go`, `*_resource.go`, `*_data_source.go`).
- `internal/provider/acctest/` — test-only HCL builders (`testconfig/`) and state-check helpers (`xknownvalue/`).
- `client/` — meshStack API client (JWT auth, RESTful CRUD; shared ref DTOs in `refs.go`).
- `docs/` — generated registry docs (`task generate`); `examples/` — embedded `.tf` examples.
