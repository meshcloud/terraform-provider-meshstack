---
name: new-resource-datasource
description: End-to-end walkthrough for adding a new meshStack resource or data source — the implementation, the example .tf files, the testconfig builder, and a good TestAcc test (create→update→import, plancheck/statecheck, xknownvalue). Use when adding or substantially reworking a resource/data source, or when writing its acceptance test. Cites the cleanest existing examples to copy from.
---

# Adding a resource / data source (with a good TestAcc test)

This is the procedure for adding a new meshStack resource or data source end-to-end. For the
concise testconfig API, builder rules, and builder-chain reference, see **`AGENTS.md`** (Testing
section) — this skill is the walkthrough plus worked examples and the code exemplars to copy.

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
4. **`examples/resources/meshstack_<name>/resource.tf`** — only the single resource block;
   put any dependencies (data sources, providers) in `test-support_*.tf`. Never hardcode
   identifiers — reference data sources / resources (see `AGENTS.md` → Dependency-first examples).
5. **`internal/provider/acctest/testconfig/build_<name>.go`** — a public builder (see below).
6. **`internal/provider/<name>_resource_test.go`** — a `TestAcc<Name>` test (see below).
7. `task generate` (docs) and update `CHANGELOG.md`.

## The builder

Public function in `testconfig`, named without `Build`/`Config`, `t` first, named returns
(`config` first), resource-under-test is the `.Join` receiver. Worked example:

```go
// internal/provider/acctest/testconfig/build_project.go
func Project(t *testing.T, workspaceAddr Traversal) (config Config, projectAddr Traversal) {
    t.Helper()
    projectName := "test-proj-" + acctest.RandString(8)
    tagConfig, tagDefinitionAddr, _ := TagDefinition(t, "meshProject")
    paymentMethodConfig, paymentMethodAddr := PaymentMethod(t, workspaceAddr)
    return Resource{Name: "project"}.Config(t).WithFirstBlock(
        ExtractAddress(&projectAddr),
        OwnedByWorkspace(workspaceAddr),
        Descend("metadata", "name")(SetString(projectName)),
        Descend("spec")(
            Descend("payment_method_identifier")(SetAddr(paymentMethodAddr, "metadata", "name")),
            Descend("tags")(SetRawExpr(`{(%s) = ["tag-value1", "tag-value2"]}`, tagDefinitionAddr.Join("spec", "key"))),
        ),
    ).Join(tagConfig, paymentMethodConfig), projectAddr
}
```

Provide a `*AndWorkspace` convenience wrapper when a single resource + its workspace is commonly
needed (e.g. `ProjectAndWorkspace`). Modifier preference: `SetString`/`SetValue` → `SetAddr` →
`SetRawExpr` (last resort). `SetRawExpr` calls `fmt.Sprintf` internally (don't wrap), takes
`Traversal` args via `%s`, and uses raw backtick strings when the HCL contains quotes. `Descend`
nests only when a parent has multiple children — flatten single-child chains.

## The TestAcc test

A good test is multi-step (create → update → import), uses the builder, and asserts with
`plancheck` (the planned action) + `statecheck`/`xknownvalue` (resulting state):

```go
func TestAccProject(t *testing.T) {
    config, resourceAddress, workspaceAddr := testconfig.ProjectAndWorkspace(t)
    updateConfig := config.WithFirstBlock(
        testconfig.Descend("spec", "display_name")(testconfig.SetString("Updated Display Name")),
    )
    ApplyAndTest(t, resource.TestCase{
        Steps: []resource.TestStep{
            { // create
                Config: config.String(),
                ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
                    plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionCreate)}},
                ConfigStateChecks: []statecheck.StateCheck{
                    statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata").AtMapKey("name"), xknownvalue.NotEmptyString()),
                    statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Project's Display Name")),
                },
            },
            { // update
                Config: updateConfig.String(),
                ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
                    plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionUpdate)}},
                ConfigStateChecks: []statecheck.StateCheck{
                    statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Updated Display Name"))},
            },
            { // import
                ResourceName: resourceAddress.String(), ImportState: true, ImportStateKind: resource.ImportBlockWithID,
                ImportStateIdFunc: func(s *terraform.State) (string, error) {
                    rs := s.RootModule().Resources[resourceAddress.String()]
                    ws := s.RootModule().Resources[workspaceAddr.String()]
                    return ws.Primary.Attributes["metadata.name"] + "." + rs.Primary.Attributes["metadata.name"], nil
                },
            },
        },
    })
}
```

Use `xknownvalue` helpers over raw `knownvalue` where they fit: `NotEmptyString()` (non-blank,
optional extra assertions), `Ref(addr, kind, &uuidOut)` (asserts a `ref` block's kind + captures
a stable uuid across steps), `MapExact{...}` (diff-friendly map assertion).

## Data source test

Reference a **resource attribute** (so Terraform infers the dependency — never `depends_on`) and
fluent-chain in one expression:

```go
func TestAccProjectDataSource(t *testing.T) {
    projectConfig, projectAddr, workspaceAddr := testconfig.ProjectAndWorkspace(t)
    dataSourceAddress := testconfig.Traversal{"data.meshstack_project", "example"}
    config := testconfig.DataSource{Name: "project"}.Config(t).WithFirstBlock(
        testconfig.Descend("metadata")(
            testconfig.Descend("name")(testconfig.SetAddr(projectAddr, "metadata", "name")),
            testconfig.Descend("owned_by_workspace")(testconfig.SetAddr(workspaceAddr, "metadata", "name")),
        )).Join(projectConfig)
    ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{{
        Config: config.String(),
        ConfigStateChecks: []statecheck.StateCheck{
            statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Project's Display Name"))},
    }}})
}
```

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

## Computed-only output fields (TF model struct embedding)

When a resource/data source needs a computed output **derived from API response fields** (not
stored on the client struct), use a local model struct — do **not** modify client types or call
`SetAttribute` after `generic.Set`:

1. Local model struct with the client's `tfsdk:`-tagged fields plus the extra computed field.
2. A `…FromDto` helper that populates and derives it.
3. Use the model struct for `generic.Set`/`generic.Get`; extract embedded client fields when
   calling the API (`client.MeshFoo{Metadata: model.Metadata, Spec: model.Spec}`).
4. Share the struct between resource and data source if the schema shape matches.
5. Do **not** add `json:"-"` fields to client structs.

```go
type myResourceModel struct {
    Metadata client.MeshFooMetadata `tfsdk:"metadata"`
    Spec     client.MeshFooSpec     `tfsdk:"spec"`
    MyOutput string                 `tfsdk:"my_output"` // derived
}
func myResourceModelFromDto(p *client.MeshFoo) myResourceModel {
    return myResourceModel{Metadata: p.Metadata, Spec: p.Spec, MyOutput: p.Metadata.Name + "." + p.Spec.SomeName}
}
```

## Running it

Iterate with the mock client first (`task test -- -run TestAcc<Name>`), then against a real
backend (`task testacc -- -run TestAcc<Name>`). For the local backend + suite runbooks see the
**`meshstack-services`** and **`acceptance-testing`** skills.
