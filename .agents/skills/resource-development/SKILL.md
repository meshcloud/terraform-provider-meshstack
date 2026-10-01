---
name: resource-development
description: How to develop meshStack Terraform resources and data sources — the implementation, example .tf files, the per-step test variants next to them, a good create→update→import TestAcc test (plancheck/statecheck, xknownvalue), and the cross-cutting schema/client design conventions (meshObject refs, DTOs, Id/Uuid naming, value receivers, list-query structs, preview API, computed-only outputs). Use when adding or reworking a resource/data source, writing its acceptance test, or applying the provider's schema/client conventions. Cites the cleanest existing examples to copy from.
---

# Developing resources & data sources

The end-to-end procedure for adding or reworking a meshStack resource or data source, plus the
schema/client design conventions that apply to all of them. This file is the walkthrough; load the
companion **`REFERENCE.md`** for the step-file layout, the `examples` composition API, when a
difference between steps is a variable rather than its own file, the `xknownvalue` state-check
helpers, and complete worked code examples (TestAcc test, data source test, computed-only field).

## Golden-path exemplars (copy these)

Mid-complexity, clean, and complete — prefer these over the large `building_block_*` files:

| Piece | File |
|---|---|
| Resource (CRUD + Schema + ImportState) | `internal/provider/project_resource.go` |
| Resource with validators/defaults/refs | `internal/provider/workspace_resource.go` |
| Data source | `internal/provider/project_data_source.go` |
| Example `.tf` (simple) | `examples/resources/meshstack_project/` (`resource.tf`, `import-by-string-id.tf`) |
| Example `.tf` (complex, multi-file + `test-support_*`) | `examples/resources/meshstack_building_block_definition/` |
| Resource test | `internal/provider/project_resource_test.go` + `examples/resources/meshstack_project/resource-test-*.tf` |
| Data source test | `internal/provider/project_data_source_test.go` + `examples/data-sources/meshstack_project/data-source-test-*.tf` |
| Step variables + a shared helper for a long flow | `internal/provider/building_block_resource_test.go` |
| Named subtests for multiple examples | `internal/provider/integration_resource_test.go` |
| State-check helpers | `internal/provider/acctest/xknownvalue/{not_empty_string,ref,map}.go` |

## Steps

1. **`internal/provider/<name>_resource.go`** — implement `resource.Resource` +
   `ResourceWithConfigure` (+ `ResourceWithImportState` for import). Standard schema shape:
   `metadata` (name `RequiresReplace`, computed `uuid` with `UseStateForUnknown`), `spec`,
   `status`. See `project_resource.go`.
2. **`github.com/meshcloud/meshstack-cli/client`** — add the API client methods (typed via
   `MeshObjectClient[M]`), in the [meshstack-cli](https://github.com/meshcloud/meshstack-cli)
   repository, which ships them before the provider can use them.
3. **`provider.go`** — register the resource/data source in the provider's lists.
4. **`examples/resources/meshstack_<name>/resource.tf`** — only the single resource block; put
   any dependencies (data sources, providers) in `test-support_*.tf`. Never hardcode
   identifiers — reference data sources / resources (see `REFERENCE.md` → Dependency-first).
5. **`examples/resources/meshstack_<name>/resource-test-<index>.tf`** — one per test step (see below).
6. **`internal/provider/<name>_resource_test.go`** — a `TestAcc<Name>` test (see below).
7. `task generate` (docs) and update `CHANGELOG.md`.

## meshObject reference attributes ({kind, uuid|name})

Build any reference to another meshObject with **`meshRefByUuid` / `meshRefByName`**
(`schema_utils.go`) — never hand-roll the `{kind, uuid|name}` block. Always pass
`meshRefOptions{Kind, Description}` (both are needed — `Kind` sets the discriminator + OneOf
validation, `Description` the block docs); then set at most one behaviour flag:

- no flag → **required input** (the common case for a resource's own spec refs): block and
  identifier both Required;
- `Output: true` → **computed output** (a resource's own `.ref` or any data-source ref; `kind`
  stays known at plan);
- `OptionalComputed: true` → an **input meshStack may default** (e.g. `runner_ref`): block and
  identifier Optional+Computed;
- `InSet: true` → a ref **hashed as an opaque set element** (nested in a `SetNestedAttribute`
  object like `project_role_ref`, or the set's own element type like `mandatory_building_block_refs`
  / `dependency_refs`): block stays Required but the identifier is Optional+Computed with an
  `AlsoRequires` guard, because a set element whose identifier is unknown at plan can't be hashed
  and a plain Required identifier would fail. See the `meshRefOptions` godoc for the full rationale.

Only refs that carry extra fields (`target_ref`, `building_block_definition_version_ref`) stay
bespoke.

On the client side these refs deserialize into the two shared DTO structs in meshstack-cli's
`client/refs.go` — `NamedRef` (`{name, kind}`) and `UuidRef` (`{uuid, kind}`), the counterparts of `meshRefByName` /
`meshRefByUuid`. Use one of them for any `{name|uuid, kind}` field rather than declaring a new
named type; a ref that adds fields (e.g. `MeshBuildingBlockV2DefinitionVersionRef`'s `content_hash`)
**embeds** the matching struct by value — both `json` and `tfsdk` reflection promote the embedded
fields. Only refs mixing name *and* uuid (`MeshBuildingBlockV2TargetRef`) stay bespoke.

## Client & schema conventions

Cross-cutting rules for the schema and its backing client, beyond the ref/DTO shape above:

- **A computed value never sits inside a container configuration writes.** `metadata` and `spec` hold
  inputs only; system-managed values belong top-level or in a fully computed container (`status`,
  `ref`, `version_latest`). The reason is testability, not tidiness: OpenTofu's mock value generator
  copies a config-written container verbatim and never descends into it, so a computed leaf under
  `metadata`/`spec` is null under `mock_provider` and cannot be given a value by `override_*` either
  — which makes any module wiring that reads it impossible to unit test. Depth is irrelevant; the
  config-written ancestor is the only thing that disqualifies the subtree, and `Optional`+`Computed`
  does not rescue it. When the API returns such a value inside `spec`, keep the client field
  (`tfsdk:"-"`) and expose it from `status` via a local model struct — see the computed-only output
  field pattern in `REFERENCE.md`. Tracked in #272.
- **Naming — `Id`/`Uuid`, never `ID`/`UUID`.** For any acronym of 2+ letters only the first letter
  is uppercase (`TenantId`, `ProjectUuid`) — it keeps mixed identifiers like `apiKeyId` readable.
- **Value receivers for client structs.** All client implementation and mock structs use value
  (not pointer) receivers, and `new*Client` functions return the value — the interface is satisfied
  by value, so a pointer return would only invite nil handling.
- **List query params go through a json-tagged struct, not an ad-hoc map.** A `List` method (and
  its interface signature) builds one query struct and hands it **by value** to
  `internal.WithUrlQuery`, which names each param from the `json` tag and drops zero-value fields
  (an implicit `omitempty` — no pointer or `,omitempty` needed; use a pointer only to send an
  explicit zero). Reach for a `map[string]string` only in the rare verbatim case where a zero value
  must still be transmitted (e.g. `page=0` in the paginator), which a struct would omit.
- **Pointer + `,omitzero` = actually-nullable only.** The `modern-go` skill is the single home for
  this rule — value-typed fields take neither, and a pointer never takes `,omitempty`.
- **Preview-API resources carry the shared disclaimer.** When a resource/data source's HTTP client
  uses an `apiVersion` ending in `-preview`, append `previewDisclaimer()` (`schema_utils.go`) to its
  `MarkdownDescription` — never inline a custom string. A **breaking** change to a `-preview`
  meshObject API needs a cross-repo handshake (meshcloud-internal — see meshfed-release's
  `terraform-provider-compat` skill): a matching provider PR landing alongside the API change plus a
  minimum-provider-version entry, so meshStack can surface "needs provider ≥ vX.Y.Z" instead of a
  cryptic failure.

## The step files

Nothing builds HCL in Go. Each test step applies a checked-in file next to the documented example,
and the test only names which files to concatenate:

- `examples/<kind>s/meshstack_<name>/<kind>-test-<index>.tf` — the example as step `<index>` applies
  it (the data sources the documented example reads swapped for resources the test creates, names
  built from `var.suffix`). The index is what links the file to the step.
- `examples/<kind>s/meshstack_<name>/test-support_<name>.tf` — the prerequisites those step files
  reference, plus the `variable` blocks the test fills.

The test assembles a step with
`examples.Resource.TestStepConfig(t, "<name>", <index>, "<support>"…)`, and one whose subject needs
another example's resources composes that example's step with `examples.JoinTestStepConfigs`
instead of duplicating it — `project_group_binding_data_source_test.go` stacks three. Resource
addresses are plain string constants (`meshstack_project.example`).

**A scalar the flow dials between steps is a `variable` with the example's value as its default; a
structural difference is its own step file.** Full rules, the composition API and that split are in
`REFERENCE.md`; `examples/README.md` has the file conventions, including the `-test-` filter that
keeps the step files out of the generated docs.

## The TestAcc test

A good test is multi-step (create → update → import) and asserts with `plancheck` (the planned
action) + `statecheck`/`xknownvalue` (resulting state). Prefer the `xknownvalue` helpers
(`NotEmptyString`, `Ref`, `MapExact`) over raw `knownvalue` where they fit. Pass the run's random
suffix via `ConfigVariables` — **including on an import step**, which needs the same variables as
the step before it, because the framework re-applies that step's config to build the import plan and
a missing variable fails the whole case. See `REFERENCE.md` for the full worked example.

## Data source test

Reference a **resource attribute** (so Terraform infers the dependency — never `depends_on`), and
compose the resource example's step config so the data source reads back what it created. Full
example in `REFERENCE.md`.

## Multiple example files → named subtests

When a resource has several suffixed example files (`resource_01_github.tf`,
`resource_02_azure_devops.tf`), give **each** its own named `t.Run()` — never a generic loop. The
top-level function calls `t.Parallel()`; each `ApplyAndTest` also parallelizes. See
`integration_resource_test.go`.

```go
func TestAccIntegrationResource(t *testing.T) {
    t.Parallel()
    t.Run("01_github", func(t *testing.T) {
        ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{{
            Config:          integrationStepConfig(t, 1),
            ConfigVariables: NewVariablesWithSuffix(acctest.RandString(8)),
            // ... checks against githubIntegrationAddr
        }}})
    })
    t.Run("02_azure_devops", func(t *testing.T) {
        ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{{
            Config:          integrationStepConfig(t, 2),
            ConfigVariables: NewVariablesWithSuffix(acctest.RandString(8)),
        }}})
    })
}
```

A file with many steps sharing one dependency chain factors it into a small local helper
(`integrationStepConfig` above, `tenantStepConfig`, `bbWorkspaceStepConfig`) rather than repeating
the `JoinTestStepConfigs` call in every step.

## Computed-only output fields

When a resource/data source needs a computed output **derived from API response fields** (not
stored on the client struct), use a local model struct holding the client's `tfsdk:`-tagged
fields plus the derived field — do **not** modify client types or call `SetAttribute` after
`generic.Set`. Full pattern and example in `REFERENCE.md`.

## Conventions worth flagging in review

- **Any new `Computed` attribute: is an ancestor written by configuration?** If yes, it belongs in
  `status` (or another fully computed container) instead — see the schema conventions above. This is
  answerable from the schema diff alone, so it is worth checking on every review.
- **Response-only pointer fields are never nil in responses.** Client DTO structs are reused for
  both requests and responses, so system-managed, response-only fields (e.g. a `Status *...`) are
  pointers *only* so they can be omitted from request payloads. On a GET the backend always
  populates them. Do **not** add `if dto.Status == nil` guards in response→state mapping; a review
  should flag a newly added one. (Genuine guards in polling loops, where a transient read may
  precede status, are a different case.)
- **Mock secret behaviour goes through `backendSecretBehavior`.** The mock client
  (`internal/clientmock`) must hash/validate sensitive inputs via the shared
  `backendSecretBehavior` helper, not a bespoke sha256 routine, so every resource hashes secrets
  identically. It walks the DTO via reflection and only mutates **addressable** fields, so secrets
  that live in a `map` value must be reachable by address — model such inputs as
  `map[string]*T` (pointer values) so the walker can reach and rewrite them.

## Running it

Iterate with the mock client first (`task test -- -run TestAcc<Name>`), then against a real
backend (`task testacc -- -run TestAcc<Name>`). For the local backend bring-up + suite runbook see
the **`acceptance-testing`** skill.

If the change also needs a **`meshfed-release` backend change**, open the provider PR and the
backend PR on branches with the **identical name**: meshfed-release CI pairs them by name and runs
this repo's acceptance suite against both combined, so neither side has to merge first.
