# AGENTS.md — meshStack Terraform Provider

Conventions for working in this repo. This is the single source of truth for both AI agents
and humans. `AGENTS_extra.md` holds verbose examples and reference material trimmed from here.

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

### Multiple example files → explicit subtests

When a resource has multiple suffixed example files (`resource_01_github.tf`,
`resource_02_azure_devops.tf`), each gets its own **named** `t.Run()` subtest — never collapse
into a generic loop. The top-level function calls `t.Parallel()`; each `ApplyAndTest` also
calls `t.Parallel()` internally.

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
- Use explicit version suffixes in file names when multiple versions exist
  (`build_building_block_v1.go`, `build_tenant_v4.go`); omit when only one version exists.

Modifier preference order: `SetString`/`SetValue` (literals) → `SetAddr(addr, "metadata", "name")`
(resource references) → `SetRawExpr(format, args...)` (complex HCL, last resort). For
`SetRawExpr`, pass `Traversal` values directly as `%s` args (it calls `fmt.Sprintf` internally —
do not wrap), and use raw backtick strings when the expression contains HCL quotes. See
`AGENTS_extra.md` for worked examples and the full `SetRawExpr`/`Descend` style rules.

Data source tests must reference a **resource attribute** (so Terraform infers the dependency),
and should fluent-chain `.Config(t).WithFirstBlock(t, ...).Join(...)` in one expression:

```go
config := testconfig.DataSource{Name: "workspace"}.Config(t).WithFirstBlock(t,
    testconfig.Descend(t, "metadata", "name")(testconfig.SetAddr(workspaceAddr, "metadata", "name")),
).Join(workspaceConfig)
```

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

### State check helpers (`xknownvalue`)

Use these instead of raw `knownvalue` functions
(`import xknownvalue "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"`):

| Helper | Description |
|---|---|
| `xknownvalue.NotEmptyString(consumers...)` | Non-whitespace string; optional extra assertions |
| `xknownvalue.Ref(addr, kind, &uuidOut)` | `ref` attribute has expected kind + stable non-empty uuid |
| `xknownvalue.MapExact(map[string]knownvalue.Check{...})` | `MapExact` with descriptive diff output |

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
testconfig.TenantV4(t, projectAddr, platformAddr, landingZoneAddr)         → (config, tenantAddr)
testconfig.TenantV4AndWorkspace(t)                                         → (config, tenantAddr)
testconfig.TenantV3(t, projectAddr, platformAddr, landingZoneAddr)         → (config, tenantAddr)
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

### Dependency-first examples

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

## Lint policy

Lint runs **only** via `task lint` → `golangci-lint` (config in `.golangci.yml`, golangci-lint
v2). `.golangci.yml` already enables `govet` as a linter, so **do not run `go vet` separately**.
Formatting (gci import ordering: stdlib → third-party → local, blank-line separated; plus gofmt)
is enforced by the same tool — fix with `task lint -- --fix`. Depguard rules isolate concerns by
directory (e.g. `clientmock` is test-only; use `hclog`, never the `log` package).

## Code review

Verify `CHANGELOG.md` has entries for all changes (features, fixes, breaking changes).

## CI/CD & action pinning

Workflows follow the HashiCorp terraform-provider-scaffolding-framework template (no Terraform
version matrix; `golangci-lint` runs in its own `golangci` job).

- **Pin every action to a full 40-char SHA**, with a version comment: `@<sha> # v1.2.3`.
  Never use mutable version tags.
- To update: `gh api repos/<owner>/<repo>/releases/latest --jq '.tag_name'`, then
  `gh api repos/<owner>/<repo>/git/refs/tags/<tag> --jq '.object.sha'`.

Full action table and gotestsum/coverage notes are in `AGENTS_extra.md`.

## Adding a new resource

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

## Computed-only output fields (TF model struct embedding)

When a resource/data source needs a computed output **derived from API response fields** (not
stored in the client struct), use a local model struct that embeds/holds the client fields plus
the extra computed field — do **not** modify client types or call `SetAttribute` after
`generic.Set`.

```go
type myResourceModel struct {
    Metadata client.MeshFooMetadata `tfsdk:"metadata"`
    Spec     client.MeshFooSpec     `tfsdk:"spec"`
    MyOutput string                 `tfsdk:"my_output"` // derived
}
func myResourceModelFromDto(p *client.MeshFoo) myResourceModel { /* populate, derive MyOutput */ }
```

Use the model struct for `generic.Set`/`generic.Get`; extract embedded client fields explicitly
when calling the API (`client.MeshFoo{Metadata: model.Metadata, Spec: model.Spec}`). The same
struct can be shared between resource and data source if the schema shape matches. Do **not** add
`json:"-"` fields to client structs.

## Client receiver & data structure rules

- **Value receivers** (not pointer) for all client implementation structs and mock clients; do
  **not** return pointers from `new*Client` functions (interface is satisfied by value).
- **Pointers + `omitempty`** only for fields actually nullable in the backend API; non-nullable
  fields use value types without `omitempty`.

## Go 1.26 `new(value)`

`new` accepts an expression, not just a type. Prefer `new(value)` over helper functions like
`ptr.To` (removed):

```go
s := new("hello")              // *string
n := new(int64(1))             // *int64
p := new(fmt.Sprintf("x:%s", h))
```
