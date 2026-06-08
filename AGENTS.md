# AGENTS.md — meshStack Terraform Provider

Conventions for working in this repo. This is the single source of truth for both AI agents
and humans. Deeper, on-demand procedures live in skills under `.agents/skills/` and are
referenced from the relevant sections below:
- **`new-resource-datasource`** — end-to-end walkthrough for adding a resource/data source + its TestAcc test.
- **`github-ci`** — GitHub Actions workflow conventions and action SHA-pinning.
- **`modern-go`** — Go 1.26 `new(expression)` and the codebase's generics.
- **`changelog-management`** — pick the next version and maintain `CHANGELOG.md`.
- **`meshstack-services`** / **`acceptance-testing`** — bring up the local backend and run/debug the suite.

Official Terraform Provider for managing meshStack resources via the meshObject API
(`/api/meshobjects`). Standard [terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework) v1.
API docs: https://docs.meshcloud.io/api/index.html#mesh_objects

## Key directories

- **`internal/provider/`** — provider implementation (`provider.go`, `*_resource.go`, `*_data_source.go`).
- **`internal/provider/acctest/testconfig/`** — HCL config fluent API (`Config`, builder functions like `testconfig.Workspace(t)`). Test-only.
- **`internal/provider/acctest/xknownvalue/`** — state-check helpers (`NotEmptyString`, `Ref`, `MapExact`).
- **`client/`** — meshStack API client (JWT auth, RESTful CRUD).
- **`docs/`** — auto-generated registry docs (`task generate`).
- **`examples/`** — embedded `.tf` example files only; `embed.go` exposes `ReadTfFile` / `ReadTestSupportTfFile`.

## Naming convention

If a variable name contains an acronym of 2+ letters, only the first letter is uppercase:
`Id` not `ID`, `Uuid` not `UUID`.

## meshObject schema structure

All resources follow: `api_version`, `kind` (e.g. `meshProject`), `metadata` (name, uuid,
timestamps), `spec` (user config), `status` (system-managed).

**meshEntity references** (e.g. `project_role_ref`): user provides `name` (required); system
sets `kind` (computed, `stringdefault.StaticString("meshProjectRole")`); validate kind with
`stringvalidator.OneOf()`; use `stringplanmodifier.UseStateForUnknown()` for kind.

## Testing

### Test modes — `ApplyAndTest`

All tests call `ApplyAndTest(t, testCase)`, which auto-selects mode:

- **Unit mode** (default, `TF_ACC` unset): in-memory mock client. Run with `task test`.
- **Acceptance mode** (`TF_ACC=1`): real local meshStack. Run with `task testacc`.

Skip tests incompatible with a mode via `IsMockClientTest()`:

```go
func TestAccSomething(t *testing.T) {
    if IsMockClientTest() { t.Skip("requires real meshStack") } // or !IsMockClientTest() for mock-only
    ApplyAndTest(t, resource.TestCase{...})
}
```

### Running tests

```bash
task test                        # unit (mock) tests
task testacc                     # acceptance tests
task test -- -run=TestValidation # filter by name
task lint                        # golangci-lint; add `-- --fix` to auto-fix
```

Direct invocations (always tee output to `/tmp` and read the log; don't re-run piped through grep):

```bash
go test -count=1 -parallel 4 -timeout 300s ./internal/provider/ 2>&1 | tee /tmp/test-unit.log
set -a && source .env && set +a   # exports .env to the go test child (plain `source` doesn't)
TF_ACC=1 go test -count=1 -parallel 4 -timeout 600s ./internal/provider/ 2>&1 | tee /tmp/test-acc.log
```

To bring up the backend services, see the **`meshstack-services`** skill; to run/debug the
full acceptance suite, see the **`acceptance-testing`** skill.

### Acceptance test authoring (testconfig, builders, state checks)

Adding or reworking a resource/data source means writing a `testconfig` HCL builder, a TestAcc
test, and example `.tf` files. That reference is **task-specific, not always-on context**, so it
lives in the **`new-resource-datasource`** skill: `SKILL.md` is the walkthrough; its companion
`REFERENCE.md` holds the full `testconfig` `Config` API, the builder rules, the builder-chain
table, the `xknownvalue` state-check helpers, and the dependency-first example conventions.

One always-on rule worth stating here, because it is easy to violate without loading the skill:
when a resource has several suffixed example files (`resource_01_github.tf`,
`resource_02_azure_devops.tf`), each gets its own **named** `t.Run()` subtest — never a generic
loop. The top-level function calls `t.Parallel()`; each `ApplyAndTest` also parallelizes.

## Lint policy

Lint runs **only** via `task lint` → `golangci-lint` (config in `.golangci.yml`, golangci-lint
v2). `.golangci.yml` already enables `govet` as a linter, so **do not run `go vet` separately**.
Formatting (gci import ordering: stdlib → third-party → local, blank-line separated; plus gofmt)
is enforced by the same tool — fix with `task lint -- --fix`. Depguard rules isolate concerns by
directory (e.g. `clientmock` is test-only; use `hclog`, never the `log` package).

## Code review

Verify `CHANGELOG.md` has entries for all user-facing changes (features, fixes, breaking
changes). The top section always names the concrete anticipated next version — never
"Unreleased"; `main` is always ready to tag. See the **`changelog-management`** skill for the
versioning policy and format.

## CI/CD & action pinning

GitHub Actions workflows pin every action to a full 40-char SHA with a `# vX.Y.Z` comment —
never mutable tags. Full conventions (jobs, update procedure, action table, gotestsum coverage)
are in the **`github-ci`** skill.

## Adding a new resource

See the **`new-resource-datasource`** skill for the full walkthrough (implementation, example
`.tf`, testconfig builder, and a good TestAcc test) with code exemplars. In short:

1. Create `*_resource.go` in `internal/provider/` with CRUD + Schema methods.
2. Add API client methods in `client/`.
3. Register in `provider.go`.
4. Add example in `examples/resources/*/`.
5. Add a builder in `internal/provider/acctest/testconfig/build_<resource>.go`.
6. Run `task generate` for docs.
7. Update `CHANGELOG.md`.

## Preview API resources

Resources/data sources using a preview API (HTTP client constructed with an `apiVersion`
ending in `-preview`) must append the standard disclaimer to their `MarkdownDescription` via
`previewDisclaimer()` (`internal/provider/schema_utils.go`) — do **not** inline a custom string.

```go
resp.Schema = schema.Schema{ MarkdownDescription: "Describe the resource." + previewDisclaimer() }
```

**Cross-repo compatibility handshake.** When a change here is driven by a **breaking** change to a
`-preview` meshObject API (renamed/removed field, changed type, changed required/optional), it must
be coordinated with the backend in `meshfed-release`. That repo's `terraform-provider-compat` skill
requires (a) a matching provider PR to land alongside the API change — otherwise already-released
provider versions silently break against the new backend — and (b) a minimum-compatible-version
entry in `TerraformProviderVersionRequirements.kt` (keyed by the media type in
`MeshHalMediaTypes.kt`), so meshStack can surface a clear "needs provider ≥ vX.Y.Z" message instead
of a cryptic failure. Communicate the provider version carrying the adaptation back to that side.

## Computed-only output fields (TF model struct embedding)

When a resource/data source needs a computed output **derived from API response fields** (not
stored in the client struct), use a local model struct that holds the client fields plus the
extra computed field — do **not** modify client types or call `SetAttribute` after `generic.Set`.
Use the model struct for `generic.Set`/`generic.Get`; extract the embedded client fields when
calling the API. Full pattern and example in the **`new-resource-datasource`** skill.

## Client receiver & data structure rules

- **Value receivers** (not pointer) for all client implementation structs and mock clients; do
  **not** return pointers from `new*Client` functions (interface is satisfied by value).
- **Pointers + `omitempty`** only for fields actually nullable in the backend API; non-nullable
  fields use value types without `omitempty`.

## Modern Go (Go 1.26 + generics)

`go.mod` targets Go 1.26. Prefer the `new(expression)` builtin for inline pointers
(`new("hello")`, `new(int64(1))`, `new(fmt.Sprintf(...))`) over removed helpers like `ptr.To`.
Reuse the codebase's generics (`MeshObjectClient[M]`, mock `Store[M]`, `Variant[X,Y]`,
`Pollable[T]`, and the `generic.Set`/`generic.Get`/`ValueTo`/`ValueFrom` TF conversion layer)
rather than `any`/reflection. See the **`modern-go`** skill for details and real examples.
