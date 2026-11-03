---
name: resource-development
description: How to develop meshStack Terraform resources and data sources — the implementation, example .tf files, the golden HCL step files a TestAcc test applies, a good create→update→import TestAcc test (plancheck/statecheck, xknownvalue), and the cross-cutting schema/client design conventions (meshObject refs, DTOs, Id/Uuid naming, value receivers, list-query structs, preview API, computed-only outputs). Use when adding or reworking a resource/data source, writing its acceptance test, or applying the provider's schema/client conventions. Cites the cleanest existing examples to copy from.
---

# Developing resources & data sources

The end-to-end procedure for adding or reworking a meshStack resource or data source, plus the
schema/client design conventions that apply to all of them. This file is the walkthrough; load the
companion **`REFERENCE.md`** for the step-file rules, the `xknownvalue` state-check helpers,
complete worked code examples (TestAcc test, data source test, computed-only field), and the legacy
`testconfig` builder that tests not yet migrated still use.

## Golden-path exemplars (copy these)

Mid-complexity, clean, and complete — prefer these over the large `building_block_*` files:

| Piece | File |
|---|---|
| Resource (CRUD + Schema + ImportState) | `internal/provider/project_resource.go` |
| Resource with validators/defaults/refs | `internal/provider/workspace_resource.go` |
| Data source | `internal/provider/project_data_source.go` |
| Data source that looks up by filters | `internal/provider/building_block_definition_data_source.go` |
| Example `.tf` (simple) | `examples/resources/meshstack_project/` (`resource.tf`, `import-by-string-id.tf`) |
| Example `.tf` (complex, multi-file) | `examples/resources/meshstack_building_block_definition/` |
| Step files | `internal/provider/testdata/hcl/resource_project_step{1,2,3}.tf` |
| Resource test (create→update→import) | `internal/provider/project_resource_test.go` |
| Data source test | `internal/provider/project_data_source_test.go` |
| One step file for several variants | `internal/provider/project_binding_resource_test.go` + `testdata/hcl/resource_project_binding_step1.tf` |
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
4. **`examples/resources/meshstack_<name>/resource.tf`** — only the single resource block. It
   reaches its workspace by reference, such as `data.meshstack_workspace.example`, and declares
   none of its prerequisites (see `REFERENCE.md` → Example conventions).
5. **`internal/provider/testdata/hcl/resource_<name>_step<N>.tf`** — one HCL file per test step
   (see below).
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
- **A singular data source for a meshObject with a UUID looks it up by filters.** Users know such
  an object by its display name and workspace, not by its UUID. Wrap `metadata` and `spec` with
  `asFilter` (`data_source_filter.go`), so the identifying attributes become optional filters. Read
  by UUID when it is set, otherwise list, and fail unless exactly one object matches. When several
  match, the error names each of them and asks for `metadata.uuid`, because a lookup that picks one
  silently reads the wrong object. Copy `building_block_definition_data_source.go`.
- **Preview-API resources carry the shared disclaimer.** When a resource/data source's HTTP client
  uses an `apiVersion` ending in `-preview`, append `previewDisclaimer()` (`schema_utils.go`) to its
  `MarkdownDescription` — never inline a custom string. A **breaking** change to a `-preview`
  meshObject API needs a cross-repo handshake (meshcloud-internal — see meshfed-release's
  `terraform-provider-compat` skill): a matching provider PR landing alongside the API change plus a
  minimum-provider-version entry, so meshStack can surface "needs provider ≥ vX.Y.Z" instead of a
  cryptic failure.

## Step files

Each config step of a TestAcc test is one file in `internal/provider/testdata/hcl/`, named
`<resource|datasource>_<name>[_<case>]_step<N>.tf`. `<name>` is the Terraform type without
`meshstack_`, `<case>` names a second test case of the same resource, and `<N>` is the step's
position in `Steps`, from 1. An import step has no file. `hclStep(t, name, N, vars)` renders the file
with `text/template`, so a reviewer reads the HCL that runs instead of Go that rewrites it.

- **Placeholders** are `{{.key}}`, filled from the `hclStepVars` the test passes to every step. The
  type exists only to feed step files. A key missing from it fails the test. `vars["suffix"]` holds `acctest.RandString(8)`, and every
  name a step file picks carries it (`"test-ws-{{.suffix}}"`), so tests never collide.
- **`{{ example "resources/meshstack_project/resource.tf" }}`** inserts an example verbatim. It
  joins its arguments, so a variant can pick its example:
  `{{ example "resources/meshstack_project_" .subject_kind "_binding/resource.tf" }}`.
- **Nothing else.** No conditionals, no loops. A variant that would need one gets its own file.
- **Each file is complete.** It declares every prerequisite inline, and only those its test needs.
  An update step is a copy of the step before it with the change, so a `diff` of the two files is
  the change.

One step applies the example, so the examples stay tested against a real meshStack. An example
carries fixed names, which a test must not collide on:

- An example reaches its workspace only by reference, and the step file points that reference at a
  workspace with a random name. Names below the workspace (a project, a project's binding target)
  stay literal.
- A globally unique literal (a tag key, a binding name) is unique across all examples, and exactly
  one test applies each example.
- When the example depends on something it cannot reference, such as a literal tag key or a data
  source looking up a literal name, an earlier step creates it. `resource_project_step1.tf` creates
  the tag definition `tag-key`, and step 2 adds the project example.

## The TestAcc test

A good test is multi-step (create → update → import), applies one step file per step, and asserts with
`plancheck` (the planned action) + `statecheck`/`xknownvalue` (resulting state). Prefer the
`xknownvalue` helpers (`Ref`, `MapExact`) over raw `knownvalue` where they fit. Assert the exact
value whenever the test knows it (a literal from the step file, `"test-ws-" + suffix`); keep
`xknownvalue.NotEmptyString()` for values meshStack generates.
See `REFERENCE.md` for the full `TestAccProject` example.

One `TestAcc*` test per resource or data source covers it, since it runs the real Terraform flow in
both modes. Add a unit test only for provider logic an acceptance test cannot reach. Leave
meshStack's own validation errors to its API tests (*meshcloud-internal*: meshfed-release); assert an
error only when provider code produces or handles it, and match it with `expectErrorWords` and a few
key words.

## Data source test

A data source example looks its object up by a literal name, so step 1 creates the object and step 2
adds the example (`datasource_project_step{1,2}.tf`). A lookup the example does not show goes into a
later step that references the resource's attributes, so Terraform infers the dependency — never
`depends_on`. Full example in `REFERENCE.md`.

## Multiple example files → named subtests

When a resource has several suffixed example files (`resource_01_github.tf`,
`resource_02_azure_devops.tf`), give **each** its own named `t.Run()` and its own step files
(`resource_integration_01_github_step1.tf`) — never a generic loop. The top-level function calls
`t.Parallel()`; each `ApplyAndTest` also parallelizes.

```go
func TestAccIntegrationResource(t *testing.T) {
    t.Parallel()
    t.Run("01_github", func(t *testing.T) {
        vars := hclStepVars{"suffix": acctest.RandString(8)}
        ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
            {Config: hclStep(t, "resource_integration_01_github", 1, vars), ...},
        }})
    })
    t.Run("02_azure_devops", func(t *testing.T) {
        ...
    })
}
```

Variants that differ only in a few values share one step file instead: the binding tests pass
`subject_kind` and `subject_name` in `vars` (`project_binding_resource_test.go`).

## Legacy: the testconfig builder

Tests not yet migrated build their configs with `internal/provider/acctest/testconfig`, which loads
an example and rewrites it. Write step files for any new test, and migrate a resource's tests when
you rework it. Delete a builder once no test calls it. `REFERENCE.md` → Legacy keeps its API for the
tests that still use it.

## Computed-only output fields

When a resource/data source needs a computed output **derived from API response fields** (not
stored on the client struct), use a local model struct holding the client's `tfsdk:`-tagged
fields plus the derived field — do **not** modify client types or call `SetAttribute` after
`generic.Set`. Full pattern and example in `REFERENCE.md`.

## State upgrade tests

Test a resource's state upgraders in one top-level `Test<Resource>UpgradeState` with one subtest per
upgrade path, named after it (`v0_to_v2`, `v1_to_v2`). Keep each prior state as a JSON file under
`internal/provider/testdata/<resource>/state-v<N>.json` (`state-v<N>-<case>.json` for several) and
load it with `//go:embed`, noting the provider version that wrote it. A new schema version then adds
a file and a subtest. Short inline JSON is fine for table cases that each vary one field.
Exemplars: `building_block_state_upgrade_test.go`, `tenant_state_upgrade_test.go`.

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
