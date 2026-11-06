# Step file / state-check / legacy builder reference

Detailed reference for the acceptance-test config layer. Load this alongside `SKILL.md` when
writing or changing a TestAcc test, its step files, or a resource's example `.tf` files. `SKILL.md`
is the end-to-end walkthrough; this file is the API surface plus full worked examples.

## Step files (`internal/provider/testdata/hcl`)

```go
type hclStepVars map[string]string

func hclStep(t *testing.T, dir string, step int, vars hclStepVars) string
```

`hclStep(t, "resources/project", 2, vars)` reads `testdata/hcl/resources/project/step2.tf`, renders
it with `text/template` (`missingkey=error`), and returns the HCL for `TestStep.Config`. It lives in
`internal/provider/hcl_step_test.go`. The template sees `vars` as `.` and one function:

| In the step file | Renders as |
|---|---|
| `[[.suffix]]` | `vars["suffix"]`; any key of `vars` works the same way |
| `[[ example "resources/meshstack_project/resource.tf" ]]` | that file under `examples/`, verbatim |
| `[[ example "resources/meshstack_project_" .subject_kind "_binding/resource.tf" ]]` | the arguments joined into one path |

A step file of the project test: the prerequisites, then the example. The data sources are what the
example references; they point at objects with random names.

```hcl
# testdata/hcl/resources/project/step2.tf
resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

data "meshstack_workspace" "example" {
  metadata = {
    name = meshstack_workspace.test.metadata.name
  }
}

# ... payment method, its data source, the tag definition "tag-key" ...

[[ example "resources/meshstack_project/resource.tf" ]]
```

`resources/project/step1.tf` is the same without the example line, because the example's literal tag
key `tag-key` has no reference that would order the project after its tag definition.
`resources/project/step3.tf` replaces the example line with a copy of the project block whose
`display_name` changed.

## State check helpers (`xknownvalue`)

Use these instead of raw `knownvalue` functions
(`import xknownvalue "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"`):

| Helper | Description |
|---|---|
| `xknownvalue.NotEmptyString(consumers...)` | Non-whitespace string; optional extra assertions |
| `xknownvalue.Ref(addr, kind, &uuidOut)` | `ref` attribute has expected kind + stable non-empty uuid |
| `xknownvalue.MapExact(map[string]knownvalue.Check{...})` | `MapExact` with descriptive diff output |

## Worked TestAcc test (create → update → import)

A good test is multi-step, applies one step file per step, and asserts with `plancheck` (the
planned action) + `statecheck`/`xknownvalue` (resulting state). Addresses are plain strings, since
the step files name them:

```go
func TestAccProject(t *testing.T) {
    suffix := acctest.RandString(8)
    vars := hclStepVars{"suffix": suffix}
    resourceAddress := "meshstack_project.example"

    ApplyAndTest(t, resource.TestCase{
        Steps: []resource.TestStep{
            { // prerequisites
                Config: hclStep(t, "resources/project", 1, vars),
            },
            { // create, from the example
                Config: hclStep(t, "resources/project", 2, vars),
                ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
                    plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionCreate)}},
                ConfigStateChecks: []statecheck.StateCheck{
                    statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("my-project")),
                    statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), knownvalue.StringExact("test-ws-"+suffix)),
                    statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Project's Display Name")),
                },
            },
            { // update
                Config: hclStep(t, "resources/project", 3, vars),
                ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
                    plancheck.ExpectResourceAction(resourceAddress, plancheck.ResourceActionUpdate)}},
                ConfigStateChecks: []statecheck.StateCheck{
                    statecheck.ExpectKnownValue(resourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Updated Display Name"))},
            },
            { // import
                ResourceName:    resourceAddress,
                ImportState:     true,
                ImportStateKind: resource.ImportBlockWithID,
                ImportStateId:   "test-ws-" + suffix + ".my-project",
            },
        },
    })
}
```

Use `xknownvalue` helpers over raw `knownvalue` where they fit: `NotEmptyString()` for a value
meshStack generates (non-blank, optional extra assertions), `Ref(addr, kind, &uuidOut)` (asserts a
`ref` block's kind + captures a stable uuid across steps), `MapExact{...}` (diff-friendly map
assertion).

## Worked data source test

The example looks the project up by its literal name, so step 1 creates it and step 2 adds the
example. Both files point `data.meshstack_workspace.example` at the test's workspace:

```go
func TestAccProjectDataSource(t *testing.T) {
    suffix := acctest.RandString(8)
    vars := hclStepVars{"suffix": suffix}
    dataSourceAddress := "data.meshstack_project.example"
    ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
        {Config: hclStep(t, "data-sources/project", 1, vars)},
        {
            Config: hclStep(t, "data-sources/project", 2, vars),
            ConfigStateChecks: []statecheck.StateCheck{
                statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), knownvalue.StringExact("test-ws-"+suffix)),
                statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Project's Display Name"))},
        },
    }})
}
```

A lookup the example does not show goes into a later step that references the resource's
attributes, so Terraform infers the dependency — never `depends_on`
(`data-sources/project_binding/step3.tf`).

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

## Example conventions

- Resource example `.tf` files contain **only the single resource block**. They reference their
  prerequisites (`data.meshstack_workspace.example`, `meshstack_project.example`) **without
  declaring them** — by design; the step file that includes the example declares them. Do **not**
  flag missing blocks in example files or generated docs as issues. Older examples still keep
  such declarations in `test-support_*.tf` files for the legacy builder.
- **The HCL a step file applies reaches the workspace by reference**
  (`data.meshstack_workspace.example.metadata.name`), never by a literal such as `"my-workspace"`.
  The step file points the reference at a random workspace, which isolates every literal name below
  it. A commented-out alternative, such as a lookup by literal names in a data source example, is
  never applied and may keep the literals a user would write.
- **A globally unique literal stays unique across all examples** (tag keys, binding names), and only
  one test applies each example. Some kinds share one namespace: meshStack rejects a project user
  binding whose name a project group binding already has, and the reverse. A test that needs the
  same kind of object creates its own with `[[.suffix]]` in the name.
- An example that must wait for another object references it, so Terraform orders the two: the
  project binding examples take their subject from `meshstack_workspace_user_binding.example`,
  because meshStack binds only workspace members to a project.
- Prefer `one(data.meshstack_<plural>.<name>.<items>)` and reusable computed outputs (`ref`,
  `identifier`, `version_latest`). When adding/changing a resource, consider whether a new
  computed read-only reference output would improve cross-resource wiring.

## Legacy: testconfig builder

Tests not yet migrated to step files build their configs with the builders below. Do not add new
builders; migrate a resource's tests to step files when you rework it, and delete a builder once no
test calls it.

### Config builder pattern (`internal/provider/acctest/testconfig`)

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
- Use explicit version suffixes in file names when a resource has builders for two API versions at
  the same time (`build_thing_v1.go`, `build_thing_v2.go`); omit the suffix otherwise.

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
needed (e.g. `PaymentMethodAndWorkspace`).

### Config API reference

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

### Builder chain reference (bottom-up)

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
```
