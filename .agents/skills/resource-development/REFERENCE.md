# testconfig / builder / state-check reference

Detailed reference for the acceptance-test config layer. Load this alongside `SKILL.md` when
writing or changing a `testconfig` builder, a TestAcc test, or a resource's example `.tf` files.
`SKILL.md` is the end-to-end walkthrough; this file is the API surface plus full worked examples.

## Config builder pattern (`internal/provider/acctest/testconfig`)

Each resource has a public builder in `internal/provider/acctest/testconfig/build_<resource>.go`
that composes `Config` objects from embedded example HCL. `Config` wraps `*hclwrite.File` and is
**immutable** — every method returns a new `Config`; `WithFirstBlock` clones internally (there
is no `Clone()`).

Import alias: `import testconfig "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"`.

Builder rules:
- Named without `Build` prefix / `Config` suffix: `Workspace`, `Project`, `BBDTerraform`.
- Take `t *testing.T` as the **first** param; pass `t` to all testconfig calls.
- Use **named return values**; the first return is always `config testconfig.Config`.
- Full variable names, no abbreviations (`workspaceConfig`, not `wsConfig`).
- The **resource under test** is the **receiver** of `.Join()`; dependencies are arguments.
  Call `config.Join(A, B)`, not chained `.Join(A).Join(B)`.
- Declare all `Traversal` vars upfront with `var` before `WithFirstBlock` calls that populate them.
- Return inline (`return expr.Join(...), addr`); consolidate all modifiers into a single `WithFirstBlock`.
- Use explicit version suffixes in file names when multiple versions exist
  (`build_building_block_v1.go`, `build_building_block_v2.go`); omit when only one version exists.

Modifier preference order: `SetString`/`SetValue` (literals) → `SetAddr(addr, "metadata", "name")`
(resource references) → `SetRawExpr(format, args...)` (complex HCL, last resort). For
`SetRawExpr`, pass `Traversal` values directly as `%s` args (it calls `fmt.Sprintf` internally —
do not wrap), and use raw backtick strings when the expression contains HCL quotes.

Worked builder example:

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
needed (e.g. `ProjectAndWorkspace`).

## Config API reference

`Config` wraps `*hclwrite.File`; all modifications return a new `Config`. File layout:
`config.go` (`Config`, `Block`, `Expression`, `Descend`, `WalkAttributes`, `Resource`/`DataSource`
loaders), `config_expr.go` (`ExpressionConsumer` + constructors), `config_fake_block.go`,
`traversal.go`, `build_*.go`.

```go
type Config struct { ... }                              // immutable; stores t from Config(t)
type Traversal []string                                 // e.g. ["meshstack_workspace", "my_ws"]
type Expression interface { Get(); Set(); RenameKey() }
type ExpressionConsumer func(t *testing.T, e Expression)

func NewConfig(t *testing.T, src []byte) Config
func (c Config) WithFirstBlock(mods ...ExpressionConsumer) Config  // clones, returns new
func (c Config) Join(others ...Config) Config
func (c Config) String() string                                   // for TestStep.Config

// Modifier constructors (no t — received at invocation)
testconfig.SetString("value")
testconfig.SetValue(cty.NumberIntVal(3))
testconfig.SetAddr(addr, "metadata", "name")            // preferred for resource attributes
testconfig.SetRawExpr(`{uuid = %s}`, addr)              // fmt.Sprintf format args
testconfig.RenameKey("new_name")
testconfig.ExtractAddress(&addr)

// Higher-order (no t — received from ExpressionConsumer)
testconfig.Descend("spec", "name")(modifier)            // navigate nested attribute
testconfig.WalkAttributes()(modifier)
testconfig.OwnedByWorkspace(workspaceAddr)              // sets metadata.owned_by_workspace

// Traversal helpers
addr.String()                              // "meshstack_workspace.my_ws"
addr.Join("metadata", "name")              // appends segments

// Loading .tf files
testconfig.Resource{Name: "workspace"}.Config(t)                       // examples/resources/meshstack_workspace/resource.tf
testconfig.Resource{Name: "platform", Suffix: "_01_azure"}.Config(t)
testconfig.Resource{Name: "landingzone"}.TestSupportConfig(t, "_bbd")  // test-support_bbd.tf
testconfig.DataSource{Name: "project"}.Config(t)
```

`Descend` nesting: nest only when a parent has **multiple** children; flatten single-child chains
(`Descend("spec", "display_name")(...)`, not `Descend("spec")(Descend("display_name")(...))`).

## Builder chain reference (bottom-up)

`*AndWorkspace` builders create a fresh workspace internally — use for single-resource tests.
For tests with multiple dependent resources, build the workspace once and share via `Config.Join`:

```go
workspaceConfig, workspaceAddr := testconfig.Workspace(t)
projectConfig, projectAddr := testconfig.Project(t, workspaceAddr)
platformConfig, platformAddr, platformTypeAddr := testconfig.CustomPlatform(t, workspaceAddr)
landingZoneConfig, landingZoneAddr := testconfig.LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)
config := landingZoneConfig.Join(platformConfig, projectConfig, workspaceConfig)
```

```
testconfig.Workspace(t)                                                    → (config, workspaceAddr)
testconfig.Project(t, workspaceAddr)                                       → (config, projectAddr)
testconfig.ProjectAndWorkspace(t)                                          → (config, projectAddr, workspaceAddr)
testconfig.PlatformType(t, workspaceAddr)                                  → (config, platformTypeAddr)
testconfig.PlatformTypeAndWorkspace(t)                                     → (config, platformTypeAddr)
testconfig.CustomPlatform(t, workspaceAddr)                                → (config, platformAddr, platformTypeAddr)
testconfig.CustomPlatformAndWorkspace(t)                                   → (config, platformAddr, workspaceAddr)
testconfig.PlatformAndWorkspace(t, suffix)                                 → (config, platformAddr)
testconfig.LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)   → (config, landingZoneAddr)
testconfig.LandingZoneAndWorkspace(t)                                      → (config, landingZoneAddr)
testconfig.SimpleLandingZone(t, workspaceAddr, platformAddr)               → (config, landingZoneAddr)
testconfig.PaymentMethod(t, workspaceAddr)                                 → (config, paymentMethodAddr)
testconfig.PaymentMethodAndWorkspace(t)                                    → (config, paymentMethodAddr, workspaceAddr)
testconfig.Integration(t, suffix)                                          → (config, integrationAddr)
testconfig.Tenant(t, projectAddr, platformAddr, landingZoneAddr)           → (config, tenantAddr)
testconfig.TenantAndWorkspace(t)                                           → (config, tenantAddr)
testconfig.TagDefinition(t, targetKind)                                    → (config, tagDefinitionAddr, tagKey)
testconfig.Location(t, workspaceAddr)                                      → (config, locationAddr, locationName)
testconfig.BBDTerraform(t)                                                 → (config, buildingBlockDefinitionAddr)
testconfig.BBDWithIntegration(t, suffix)                                   → (config, buildingBlockDefinitionAddr)
testconfig.BBDManual(t)                                                    → (config, buildingBlockDefinitionAddr)
testconfig.BBDGitlabPipeline(t)                                            → (config, buildingBlockDefinitionAddr)
testconfig.BBv1Tenant(t)                                                   → (config, buildingBlockAddr)
testconfig.BBv2Workspace(t)                                                → (config, buildingBlockAddr)
testconfig.BBv2Tenant(t)                                                   → (config, buildingBlockAddr)
```

## State check helpers (`xknownvalue`)

Use these instead of raw `knownvalue` functions
(`import xknownvalue "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"`):

| Helper | Description |
|---|---|
| `xknownvalue.NotEmptyString(consumers...)` | Non-whitespace string; optional extra assertions |
| `xknownvalue.Ref(addr, kind, &uuidOut)` | `ref` attribute has expected kind + stable non-empty uuid |
| `xknownvalue.MapExact(map[string]knownvalue.Check{...})` | `MapExact` with descriptive diff output |

## Worked TestAcc test (create → update → import)

A good test is multi-step, uses the builder, and asserts with `plancheck` (the planned action) +
`statecheck`/`xknownvalue` (resulting state):

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

## Worked data source test

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

## Worked computed-only output field (TF model struct embedding)

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

## Dependency-first example conventions

- Resource example `.tf` files contain **only the single resource block**. Supporting blocks
  (data sources, providers) go in `test-support_*.tf` files. Example files reference data
  sources (e.g. `data.meshstack_workspace.example`) **without declaring them** — by design; the
  declarations live in `test-support_*.tf` and are loaded alongside during tests. Do **not**
  flag missing data source blocks in example files or generated docs as issues.
- **Never hardcode identifiers** (`"my-workspace"`, UUIDs) in example HCL — always use data
  source / resource references (`data.meshstack_workspace.example.metadata.name`,
  `data.meshstack_platform.example.ref`).
- Prefer `one(data.meshstack_<plural>.<name>.<items>)` and reusable computed outputs (`ref`,
  `identifier`, `version_latest`). When adding/changing a resource, consider whether a new
  computed read-only reference output would improve cross-resource wiring.
