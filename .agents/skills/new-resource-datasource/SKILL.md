---
name: new-resource-datasource
description: End-to-end walkthrough for adding a new meshStack resource or data source — the implementation, the example .tf files, the testconfig builder, and a good TestAcc test (create→update→import, plancheck/statecheck, xknownvalue). Use when adding or substantially reworking a resource/data source, or when writing its acceptance test. Cites the cleanest existing examples to copy from.
---

# Adding a resource / data source (with a good TestAcc test)

The end-to-end procedure for adding a new meshStack resource or data source. This file is the
walkthrough; load the companion **`REFERENCE.md`** for the full `testconfig` `Config` API, builder
rules, the builder-chain table, the `xknownvalue` state-check helpers, and complete worked code
examples (builder, TestAcc test, data source test, computed-only field).

## Golden-path exemplars (copy these)

Mid-complexity, clean, and complete — prefer these over the large `building_block_*` files:

| Piece | File |
|---|---|
| Resource (CRUD + Schema + ImportState) | `internal/provider/project_resource.go` |
| Resource with validators/defaults/refs | `internal/provider/workspace_resource.go` |
| Data source | `internal/provider/project_data_source.go` |
| Example `.tf` (simple) | `examples/resources/meshstack_project/` (`resource.tf`, `import-by-string-id.tf`) |
| Example `.tf` (complex, multi-file + `test-support_*`) | `examples/resources/meshstack_building_block_definition/` |
| testconfig builder | `internal/provider/acctest/testconfig/build_project.go` |
| Resource test (create→update→import) | `internal/provider/project_resource_test.go` |
| Data source test | `internal/provider/project_data_source_test.go` |
| Named subtests for multiple examples | `internal/provider/integration_resource_test.go` |
| State-check helpers | `internal/provider/acctest/xknownvalue/{not_empty_string,ref,map}.go` |

## Steps

1. **`internal/provider/<name>_resource.go`** — implement `resource.Resource` +
   `ResourceWithConfigure` (+ `ResourceWithImportState` for import). Standard schema shape:
   `metadata` (name `RequiresReplace`, computed `uuid` with `UseStateForUnknown`), `spec`,
   `status`. See `project_resource.go`.
2. **`client/`** — add the API client methods (typed via `MeshObjectClient[M]`).
3. **`provider.go`** — register the resource/data source in the provider's lists.
4. **`examples/resources/meshstack_<name>/resource.tf`** — only the single resource block; put
   any dependencies (data sources, providers) in `test-support_*.tf`. Never hardcode
   identifiers — reference data sources / resources (see `REFERENCE.md` → Dependency-first).
5. **`internal/provider/acctest/testconfig/build_<name>.go`** — a public builder (see below).
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

## The builder

A public function in `testconfig`, named without `Build`/`Config`, `t` first, named returns
(`config` first), with the resource-under-test as the `.Join` receiver and dependencies as
arguments. Modifier preference: `SetString`/`SetValue` → `SetAddr` → `SetRawExpr` (last resort).
`Descend` nests only when a parent has multiple children — flatten single-child chains. Provide a
`*AndWorkspace` wrapper when a single resource + its workspace is commonly needed. Full rules and
a worked `build_project.go` are in `REFERENCE.md`.

## The TestAcc test

A good test is multi-step (create → update → import), uses the builder, and asserts with
`plancheck` (the planned action) + `statecheck`/`xknownvalue` (resulting state). Prefer the
`xknownvalue` helpers (`NotEmptyString`, `Ref`, `MapExact`) over raw `knownvalue` where they fit.
See `REFERENCE.md` for the full `TestAccProject` example.

## Data source test

Reference a **resource attribute** (so Terraform infers the dependency — never `depends_on`) and
fluent-chain `.Config(t).WithFirstBlock(...).Join(...)` in one expression. Full example in
`REFERENCE.md`.

## Multiple example files → named subtests

When a resource has several suffixed example files (`resource_01_github.tf`,
`resource_02_azure_devops.tf`), give **each** its own named `t.Run()` — never a generic loop. The
top-level function calls `t.Parallel()`; each `ApplyAndTest` also parallelizes. See
`integration_resource_test.go`.

```go
func TestAccIntegrationResource(t *testing.T) {
    t.Parallel()
    t.Run("01_github", func(t *testing.T) {
        config, addr := testconfig.Integration(t, "_01_github")
        ApplyAndTest(t, resource.TestCase{...})
    })
    t.Run("02_azure_devops", func(t *testing.T) {
        config, addr := testconfig.Integration(t, "_02_azure_devops")
        ApplyAndTest(t, resource.TestCase{...})
    })
}
```

## Computed-only output fields

When a resource/data source needs a computed output **derived from API response fields** (not
stored on the client struct), use a local model struct holding the client's `tfsdk:`-tagged
fields plus the derived field — do **not** modify client types or call `SetAttribute` after
`generic.Set`. Full pattern and example in `REFERENCE.md`.

## Conventions worth flagging in review

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
