# Step files / composition / state-check reference

Detailed reference for the acceptance-test config layer. Load this alongside `SKILL.md` when
writing or changing a resource's example `.tf` files, the per-step variants its test applies, or a
TestAcc test. `SKILL.md` is the end-to-end walkthrough; this file is the API surface plus full
worked examples.

## Where a step's HCL comes from

Nothing builds HCL in Go. Each step applies files checked in next to the documented example, and
the test only names which ones to concatenate:

- `examples/<kind>s/meshstack_<name>/<kind>-test-<index>.tf` — the example as step `<index>` applies
  it: the same block, with the data sources the documented example reads swapped for resources the
  test creates, and every name built from `var.suffix`. Numbered from 1; the index is what links
  the file to the step.
- `examples/<kind>s/meshstack_<name>/test-support_<name>.tf` — the prerequisites those step files
  reference (the owning workspace, tag definitions, a provider alias, …) plus the `variable` blocks
  the test fills through `ConfigVariables`.

`examples/README.md` holds the file conventions, including the `-test-` filter in
`templates/{resources,data-sources}.md.tmpl` that keeps the step files out of the generated docs.

## Composition API (`examples/embed.go`)

`import "github.com/meshcloud/terraform-provider-meshstack/examples"`

| Call | Returns |
|---|---|
| `examples.Resource.TestStepConfig(t, name, index, supports...)` | `resource-test-<index>.tf` followed by each named `test-support_<support>.tf` |
| `examples.DataSource.TestStepConfig(t, name, index, supports...)` | the same for `data-source-test-<index>.tf` |
| `examples.Resource.TestSupportConfigs(t, name, supports...)` | only the support files — for a step that composes several of an example's step files, or none of them |
| `examples.JoinTestStepConfigs(configs...)` | several examples' step configs as the one config a step applies |

Rules:

- **Exactly one file in a composed config declares a given `variable`.** Two examples that each pull
  in their own variables file collide, so a stack names it once.
- **A subject that depends on another example's resources composes that example's step** instead of
  redeclaring them. The `meshstack_project` data source reads back what the project resource example
  created, and both project bindings target it.
- **Resource addresses are plain string constants** (`meshstack_project.example`), declared once at
  the top of the test file — the labels in the step files are fixed, so nothing needs extracting.
- **A step file may reuse a label another support file declares.** Every building block definition
  support file declares `meshstack_building_block_definition.example`, so swapping which file a
  config composes swaps the definition the step files wire themselves to, without touching them.

## Variable or step file?

Both exist; what decides is what changes between the steps:

- **A scalar the flow dials** — a display name, an input value, a flag — is a `variable` whose
  default is the documented example's value. The steps then share one `Config` and differ only in
  `ConfigVariables`.
- **A structural difference** — an attribute appearing or disappearing, a different expression
  behind a ref, an extra block — is its own step file. A conditional would have to reconcile both
  branches' types, and the file stays readable as plain HCL.

`SuffixVariables(suffix)` is what every case starts from: it passes the run's random suffix, which
every test-created name is built from, so parallel runs and re-runs never collide. Steps that must
address the same resources share one value, so a case builds it once and reuses it — **including an
import step**, whose plan the framework builds from the preceding step's config, so a missing
variable there fails the whole case.

`building_block_resource_test.go` shows the pattern at its largest: a small `bbVariables` type whose
`with*` methods return a copy, so a value one step introduces never reaches the variables an earlier
step already ran with.

## State check helpers (`xknownvalue`)

Use these instead of raw `knownvalue` functions
(`import xknownvalue "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"`):

| Helper | Description |
|---|---|
| `xknownvalue.NotEmptyString(consumers...)` | Non-whitespace string; optional extra assertions |
| `xknownvalue.Ref(addr, kind, &uuidOut)` | `ref` attribute has expected kind + stable non-empty uuid |
| `xknownvalue.MapExact(map[string]knownvalue.Check{...})` | `MapExact` with descriptive diff output |

## Worked TestAcc test (create → update → import)

A good test is multi-step and asserts with `plancheck` (the planned action) + `statecheck` /
`xknownvalue` (the resulting state). Steps 1 and 2 are two step files of the same example; the
import step repeats the preceding step's `ConfigVariables`. Live example:
`project_resource_test.go`.

```go
const (
    projectResourceAddr          = "meshstack_project.example"
    projectWorkspaceResourceAddr = "meshstack_workspace.example"
)

func TestAccProject(t *testing.T) {
    vars := SuffixVariables(acctest.RandString(8))

    ApplyAndTest(t, resource.TestCase{
        Steps: []resource.TestStep{
            { // create
                Config:          examples.Resource.TestStepConfig(t, "project", 1, "prerequisites"),
                ConfigVariables: vars,
                ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
                    plancheck.ExpectResourceAction(projectResourceAddr, plancheck.ResourceActionCreate)}},
                ConfigStateChecks: []statecheck.StateCheck{
                    statecheck.ExpectKnownValue(projectResourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), xknownvalue.NotEmptyString()),
                    statecheck.ExpectKnownValue(projectResourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Project's Display Name")),
                },
            },
            { // update — step file 2 is step file 1 with a changed display name
                Config:          examples.Resource.TestStepConfig(t, "project", 2, "prerequisites"),
                ConfigVariables: vars,
                ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
                    plancheck.ExpectResourceAction(projectResourceAddr, plancheck.ResourceActionUpdate)}},
                ConfigStateChecks: []statecheck.StateCheck{
                    statecheck.ExpectKnownValue(projectResourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Updated Display Name"))},
            },
            { // import
                ResourceName:    projectResourceAddr,
                ImportState:     true,
                ImportStateKind: resource.ImportBlockWithID,
                ConfigVariables: vars,
                ImportStateIdFunc: func(s *terraform.State) (string, error) {
                    rs := s.RootModule().Resources[projectResourceAddr]
                    if rs == nil {
                        return "", fmt.Errorf("resource not found: %s", projectResourceAddr)
                    }
                    ws := s.RootModule().Resources[projectWorkspaceResourceAddr]
                    if ws == nil {
                        return "", fmt.Errorf("workspace resource not found: %s", projectWorkspaceResourceAddr)
                    }
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

The data source reads back what the resource example created, so the step composes both: the data
source's own step file and the resource example's step with its prerequisites (which is what carries
`variable "suffix"`). The data source step file references a **resource attribute**, so Terraform
infers the dependency — never `depends_on`.

```go
const projectDataSourceAddr = "data.meshstack_project.example"

func TestAccProjectDataSource(t *testing.T) {
    config := examples.JoinTestStepConfigs(
        examples.DataSource.TestStepConfig(t, "project", 1),
        examples.Resource.TestStepConfig(t, "project", 1, "prerequisites"),
    )

    ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{{
        Config:          config,
        ConfigVariables: SuffixVariables(acctest.RandString(8)),
        ConfigStateChecks: []statecheck.StateCheck{
            statecheck.ExpectKnownValue(projectDataSourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Project's Display Name"))},
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
