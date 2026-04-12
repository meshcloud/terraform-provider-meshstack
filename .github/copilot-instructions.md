# Copilot Instructions for meshStack Terraform Provider

## Overview

Official Terraform Provider for managing meshStack resources via Infrastructure as Code using the meshObject API (`/api/meshobjects`).
- **API Docs**: https://docs.meshcloud.io/api/index.html#mesh_objects
- **Architecture**: Standard Terraform Provider Plugin Framework

## Key Directories

- **`internal/provider/`**: Provider implementation (`provider.go`, `*_resource.go`, `*_data_source.go`)
- **`internal/provider/acctest/testconfig/`**: HCL config fluent API, all `Build*Config` builder functions, and `Traversal` type — used only in tests
- **`internal/provider/acctest/xknownvalue/`**: State check helpers (`NotEmptyString`, `Ref`, `MapExact`)
- **`client/`**: meshStack API client (JWT auth, RESTful CRUD operations)
- **`docs/`**: Auto-generated Terraform registry documentation
- **`examples/`**: Embedded .tf example files only (`resources/`, `data-sources/`); `embed.go` exposes `ReadTfFile` and `ReadTestSupportTfFile`

## Conventions

- If a variable contains an acronym of 2 or more letters, only the first letter should be uppercase (e.g., Id instead of ID).

## Development Patterns

### meshObject Schema Structure
All resources follow this standard schema:
- `api_version` - API version
- `kind` - meshObject type (e.g., "meshProject", "meshWorkspace")
- `metadata` - Object metadata (name, uuid, timestamps)
- `spec` - User-defined configuration
- `status` - System-managed state

### meshEntity Reference Pattern
For references to other meshEntities (e.g., `project_role_ref`):
- User provides: `name` (required)
- System sets: `kind` (computed with default, e.g., `stringdefault.StaticString("meshProjectRole")`)
- Use validators: `stringvalidator.OneOf()` for kind validation
- Use plan modifiers: `stringplanmodifier.UseStateForUnknown()` for kind

## Testing

### Test Modes — ApplyAndTest

All tests use the `ApplyAndTest(t, testCase)` helper which auto-selects mode:

- **Unit mode** (default, `TF_ACC` not set): Uses an in-memory mock client. Run with `go test ./internal/provider/`.
- **Acceptance mode** (`TF_ACC=1`): Runs against a real local meshStack. Run with `TF_ACC=1 go test ./internal/provider/ -parallel 8`.

Use `IsMockClientTest()` to skip tests incompatible with a specific mode:

```go
func TestAccSomething(t *testing.T) {
    if IsMockClientTest() {
        t.Skip("requires real meshStack")
    }
    // or:
    if !IsMockClientTest() {
        t.Skip("mock-only test")
    }
    ApplyAndTest(t, resource.TestCase{...})
}
```

### Running Acceptance Tests Locally

**Prerequisites — start meshStack services in `meshfed-release/` worktree:**

Always start Gradle services in the background with logs redirected to `/tmp/` files.
The `./gradlew :*:start` tasks block forever (they run Spring Boot apps), so they must
not be awaited. Use `nohup ... &` and verify health endpoints afterwards.

```bash
# 1. Start infrastructure (MariaDB, RabbitMQ, Keycloak, RavenDB)
cd meshfed-release/
docker compose up -d

# 2. Start all services in background with log files
nohup ./gradlew :meshfed:meshfed-api:start --console=plain > /tmp/meshstack-api.log 2>&1 &
nohup ./gradlew :buildingblocks:block-coordinator-api:start --console=plain > /tmp/block-coordinator.log 2>&1 &
nohup ./gradlew :buildingblocks:manual-block-runner:start --console=plain > /tmp/manual-runner.log 2>&1 &
nohup ./gradlew :meshfed:replicator:replicator-api:start --console=plain > /tmp/replicator.log 2>&1 &

# 3. Wait for meshfed-api to be ready (takes ~60-120s for first start)
until curl -sf http://localhost:8080/mesh/info > /dev/null 2>&1; do sleep 5; done
echo "meshfed-api ready"
```

**Run tests:**

```bash
cd terraform-provider-meshstack/
set -a && source .env && set +a   # exports MESHSTACK_ENDPOINT, MESHSTACK_API_KEY, MESHSTACK_API_SECRET
TF_ACC=1 go test ./internal/provider/ -parallel 8 -timeout 600s -v 2>&1 | tee /tmp/acc-tests.log
```

**Investigating failures while tests run:**

```bash
# Check which tests passed/failed so far
grep -E -- '--- (PASS|FAIL|SKIP)' /tmp/acc-tests.log

# Check meshfed-api logs for errors
tail -f /tmp/meshstack-api.log | grep -i 'error\|exception\|warn'

# Check block coordinator logs
tail -f /tmp/block-coordinator.log

# Check manual runner logs (BB runs stuck at PENDING?)
tail -f /tmp/manual-runner.log

# Check replicator logs (tenant deletion stuck?)
tail -f /tmp/replicator.log
```

**Common failure causes:**
- **BB stuck at PENDING** → block-coordinator or manual-block-runner not running
- **Tenant delete 400** → replicator not running, or mandatory BBs still pending
- **409 Conflict** → stale data from previous test run (tag definitions, etc.)
- **422 on bindings** → groups/users referenced in examples don't exist locally

### Multiple Example Files and Subtests

When a resource has multiple example files with suffixes (e.g., `resource_01_github.tf`, `resource_02_azure_devops.tf`), each example **must** have its own explicit `t.Run()` subtest. Do not collapse these into a generic loop — keep subtests named and explicit so each scenario is easy to find, read, and extend independently.

The top-level test function calls `t.Parallel()` so it doesn't block other tests. Each subtest calls `ApplyAndTest` which also calls `t.Parallel()` internally.

```go
func TestAccIntegrationResource(t *testing.T) {
    t.Parallel()

    t.Run("01_github", func(t *testing.T) {
        config, addr := testconfig.BuildIntegrationConfig(t, "_01_github")
        ApplyAndTest(t, resource.TestCase{...})
    })

    t.Run("02_azure_devops", func(t *testing.T) {
        config, addr := testconfig.BuildIntegrationConfig(t, "_02_azure_devops")
        ApplyAndTest(t, resource.TestCase{...})
    })
}
```

### Config Builder Pattern

Each resource has a `Build*Config(t *testing.T, ...)` function in `internal/provider/acctest/testconfig/` that composes `Config` objects from embedded example HCL files. Configs are randomized so tests run against an empty meshStack without naming conflicts.

`Config` wraps `*hclwrite.File` and is **immutable** — all modifications return a new `Config`. `WithFirstBlock` always clones internally; no need to call `Clone()` (which doesn't exist).

Import alias convention:
```go
import testconfig "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
```

Rules for builder functions:
- Live in `internal/provider/acctest/testconfig/build_<resource>.go` as **public** `Build*Config` functions
- Take `t *testing.T` as **first** parameter; pass `t` to all testconfig calls
- Use **named return values** — first return is always `config testconfig.Config`
- Use **full variable names**, no abbreviations (`workspaceConfig` not `wsConfig`, `workspaceAddr` not `wsAddr`)
- The **resource under test** must be the **receiver** of `.Join()`, with dependency configs as arguments
- Call `config.Join(A, B)` not chained `config.Join(A).Join(B)`
- **Declare all Traversal variables upfront** with `var` before any `.WithFirstBlock` calls that populate them
- **Return inline**: use `return expr.Join(...), addr` — do not assign to a named `config` var and then return
- **Consolidate `WithFirstBlock` calls**: combine all modifiers (including `OwnedByWorkspace`) into a single `WithFirstBlock` call rather than calling it multiple times
- **Prefer `Traversal.Join` over `Traversal.Format`** for simple path segment appending; keep `Format` only for complex wrapping like `"[%s.ref]"` or template strings with `%s` already embedded
- **Compact `SetRawExpr`**: use comma-separated attributes in objects/lists — no `\n` or extra indentation. Pass Traversal values via `fmt.Sprintf` (uses `.String()` automatically) or call `.String()` explicitly

```go
// internal/provider/acctest/testconfig/build_workspace.go
func BuildWorkspaceConfig(t *testing.T) (config testconfig.Config, workspaceAddr testconfig.Traversal) {
    t.Helper()
    name := "test-ws-" + acctest.RandString(8)
    tagConfig, tagAddr := BuildTagDefinitionConfig(t, "meshWorkspace")
    return Resource{Name: "workspace"}.Config(t).WithFirstBlock(t,
        testconfig.ExtractIdentifier(&workspaceAddr),
        testconfig.Traverse(t, "metadata")(
            testconfig.Traverse(t, "name")(testconfig.SetString(name)),
            testconfig.Traverse(t, "tags")(testconfig.SetRawExpr(
                fmt.Sprintf("{(%s) = [\"12345\"]}", tagAddr.Join("spec", "key")),
            )),
        ),
    ).Join(tagConfig), workspaceAddr
}
```

**Key style rules for `SetRawExpr`:**
- Compact inline: `{key = value, key2 = value2}` — commas not newlines
- Traversal in format string: `fmt.Sprintf("{uuid = %s}", addr.Join("metadata", "uuid"))` — Go calls `.String()` on the Traversal via `%s`
- Traversal as standalone: `addr.Join("metadata", "name").String()` — explicit `.String()` call

**Usage in tests:**
```go
func TestAccProject(t *testing.T) {
    config, projectAddr, workspaceAddr := testconfig.BuildProjectAndWorkspaceConfig(t)
    ApplyAndTest(t, resource.TestCase{...})
}
```

**Data source test pattern:**
```go
var resourceAddress testconfig.Traversal
projectConfig, projectAddr, _ := testconfig.BuildProjectAndWorkspaceConfig(t)
config := testconfig.DataSource{Name: "project"}.Config(t).
    WithFirstBlock(t,
        testconfig.Traverse(t, "metadata", "name")(testconfig.SetRawExpr(projectAddr.Join("metadata", "name").String())),
    ).Join(projectConfig)
```

**Test step updates** — `WithFirstBlock` auto-clones, so just capture the return:
```go
// Step 2: Update display name
Config: config.WithFirstBlock(t,
    testconfig.Traverse(t, "spec", "display_name")(testconfig.SetString("Updated Name")),
).String(),
```

**Builder chain (bottom-up):**
```
testconfig.BuildWorkspaceConfig(t)                                  → (config, workspaceAddr)
testconfig.BuildProjectConfig(t, workspaceAddr)                     → (config, projectAddr)
testconfig.BuildProjectAndWorkspaceConfig(t)                        → (config, projectAddr, workspaceAddr)
testconfig.BuildPlatformTypeConfig(t, workspaceAddr)                → (config, platformTypeAddr)
testconfig.BuildPlatformConfig(t, suffix)                           → (config, platformAddr)
testconfig.BuildLandingZoneConfig(t, wsAddr, platformAddr, ptAddr)  → (config, landingZoneAddr)
testconfig.BuildPaymentMethodConfig(t)                              → (config, paymentMethodAddr, workspaceAddr)
testconfig.BuildIntegrationConfig(t, suffix)                        → (config, integrationAddr)
testconfig.BuildTenantConfig(t)                                     → (config, tenantAddr)
testconfig.BuildTagDefinitionConfig(t, targetKind)                  → (config, tagAddr)
testconfig.BuildLocationConfig(t, workspaceAddr)                    → (config, locationAddr)
testconfig.BuildBBDTerraformConfig(t)                               → (config, bbdAddr)
testconfig.BuildBBDWithIntegrationConfig(t, suffix)                 → (config, bbdAddr)
testconfig.BuildBBDManualConfig(t)                                  → (config, bbdAddr)
testconfig.BuildBBDGitlabPipelineConfig(t)                          → (config, bbdAddr)
testconfig.BuildBBv2WorkspaceConfig(t)                              → (config, bbAddr)
testconfig.BuildBBv2TenantConfig(t)                                 → (config, bbAddr)
```

### Dependency-first Examples

- In example HCL, prefer data source references for dependencies over hardcoded identifiers.
- If a plural data source is available, prefer `one(data.meshstack_<plural>.<name>.<items>)` for wiring references.
- Prefer reusable computed outputs (`ref`, `identifier`, `version_latest`, `version_latest_release`) where available.
- When adding or changing a resource/data source, consider whether an additional computed read-only reference output would improve cross-resource wiring.

### Config API (`internal/provider/acctest/testconfig`)

`Config` wraps `*hclwrite.File`. All modifications return a new `Config` — no mutation.

**Types:**
```go
type Config struct { ... }                           // immutable; use WithFirstBlock/Join/FlipBooleans
type Traversal []string                              // resource address e.g. ["meshstack_workspace", "my_ws"]
type Expression interface { Get(); Set(); RenameKey() }
type ExpressionModifier func(t *testing.T, e Expression)
```

**Config methods:**
```go
func NewConfig(t *testing.T, src []byte) Config
func (c Config) WithFirstBlock(t *testing.T, mods ...ExpressionModifier) Config  // clones, returns new
func (c Config) Join(others ...Config) Config                                     // returns new
func (c Config) FlipBooleans() Config                                             // returns new
func (c Config) String() string                                                   // for TestStep.Config
```

**Modifier constructors (no `t` in constructor — receive from invocation):**
```go
testconfig.SetString("value")                           // set string literal
testconfig.SetCty(cty.NumberIntVal(3))                  // set any cty.Value
testconfig.SetRawExpr("other_resource.example.id")      // set raw HCL expression string
testconfig.RemoveKey()                                  // remove the attribute
testconfig.RenameKey("new_name")                        // rename last block label
testconfig.ExtractIdentifier(&addr)                     // capture resource Traversal into &addr
```

**Higher-order (take `t` for tree traversal):**
```go
testconfig.Traverse(t, "spec", "name")(modifier)        // navigate into nested attribute
testconfig.OwnedByWorkspace(t, workspaceAddr)           // convenience: sets metadata.owned_by_workspace
```

**`Traverse` nesting** — modifiers returned by `Traverse` are themselves `ExpressionModifier`, so they can be nested:
```go
config.WithFirstBlock(t,
    testconfig.Traverse(t, "spec")(
        testconfig.Traverse(t, "nested_block")(testconfig.SetString("value")),
    ),
)
```

**`Traversal` helpers:**
```go
addr.String()                // "meshstack_workspace.my_ws"
addr.Join("metadata", "name").String()  // "meshstack_workspace.my_ws.metadata.name"
addr.Format("[%s.ref]")     // keep for complex HCL expressions only
```

**Loading .tf files:**
```go
testconfig.Resource{Name: "workspace"}.Config(t)                // examples/resources/meshstack_workspace/resource.tf
testconfig.Resource{Name: "platform", Suffix: "_01_azure"}.Config(t)  // resource_01_azure.tf
testconfig.Resource{Name: "landingzone"}.TestSupportConfig(t, "_bbd")  // test-support_bbd.tf
testconfig.DataSource{Name: "project"}.Config(t)                // examples/data-sources/meshstack_project/data-source.tf
```

### State Check Helpers (`internal/provider/acctest/xknownvalue`)

Use `xknownvalue` helpers instead of raw `knownvalue` functions:

```go
import xknownvalue "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
```

| Helper | Description |
|---|---|
| `xknownvalue.NotEmptyString(consumers...)` | Asserts value is a non-whitespace string; optional consumer funcs for extra assertions |
| `xknownvalue.Ref(addr, kind, &uuidOut)` | Asserts `ref` attribute has expected kind and stable non-empty uuid |
| `xknownvalue.MapExact(map[string]knownvalue.Check{...})` | AI-friendly `MapExact` with descriptive diff output |

### Data Source Tests

Data source tests must reference a **resource attribute** (not `depends_on`) so Terraform infers the dependency automatically:

```go
// Good: attribute reference creates implicit dependency
var resourceAddress testconfig.Traversal
workspaceConfig, workspaceAddr := testconfig.BuildWorkspaceConfig(t)
config := testconfig.DataSource{Name: "workspace"}.Config(t).
    Join(workspaceConfig)
config = config.WithFirstBlock(t,
    testconfig.Traverse(t, "metadata", "name")(testconfig.SetRawExpr(workspaceAddr.Join("metadata", "name").String())),
)
```

### Code Review Requirements
- Verify that `CHANGELOG.md` includes entries for all changes (features, fixes, breaking changes)

### Adding New Resources
1. Create `*_resource.go` in `/internal/provider/` with CRUD + Schema methods
2. Add API client methods in `/client/`
3. Register in `provider.go`
4. Add example in `/examples/resources/*/`
5. Add `Build*Config(t *testing.T, ...)` builder in `internal/provider/acctest/testconfig/build_<resource>.go`
6. Run `go generate` for docs
7. Update `CHANGELOG.md` with appropriate entry

### Preview API Resources
Resources and data sources that use a preview API must include a standardized disclaimer in their `MarkdownDescription`. Use the `previewDisclaimer()` helper from `internal/provider/schema_utils.go`:

```go
resp.Schema = schema.Schema{
    MarkdownDescription: "Describe the resource here." + previewDisclaimer(),
    // ...
}
```

Do **not** inline a custom disclaimer string.

Identify if a resource or data source uses a preview API by checking if the HTTP client is constructed with an `apiVersion` that has a `-preview` suffix.

### Running Tests

**Unit tests** (mock client, no real meshStack needed):
```bash
go test -count=1 -parallel 4 -timeout 300s ./internal/provider/ 2>&1 | tee /tmp/test-unit.log
```
- `TF_ACC` must be **unset** — the Terraform Plugin SDK then skips real provider startup and the mock client injected via `SetupMockClient()` is used instead.
- Always use `-parallel 4` since the mock-backed tests run the full acceptance test harness and are slow without it.
- Always tee to a file so errors can be inspected without re-running.

**Acceptance tests** (real local meshStack):
```bash
set -a && source .env && set +a
TF_ACC=1 go test -count=1 -parallel 4 -timeout 600s ./internal/provider/ 2>&1 | tee /tmp/test-acc.log
```
- `set -a && source .env && set +a` exports all variables from `.env` to child processes (`source` alone does not export them to `go test`).
- Requires a running local meshStack at `http://localhost:...` with env vars `MESHSTACK_ENDPOINT`, `MESHSTACK_API_KEY`, `MESHSTACK_API_SECRET`.

### Adding Computed-Only Output Fields to Resources/Data Sources

When a resource or data source needs a computed output field that is **derived from API response fields** (not stored in the client struct), use the **TF model struct embedding pattern** instead of modifying client types or calling `SetAttribute` after `generic.Set`.

**Pattern:**
1. Define a local model struct with the same `tfsdk:`-tagged fields as the client struct, plus the extra computed field(s):
   ```go
   type myResourceModel struct {
       Metadata client.MeshFooMetadata `tfsdk:"metadata"`
       Spec     client.MeshFooSpec     `tfsdk:"spec"`
       MyOutput string                 `tfsdk:"my_output"` // extra computed field
   }
   ```
2. Add a helper to populate it from the API DTO:
   ```go
   func myResourceModelFromDto(p *client.MeshFoo) myResourceModel {
       return myResourceModel{
           Metadata: p.Metadata,
           Spec:     p.Spec,
           MyOutput: p.Metadata.Name + "." + p.Spec.SomeName, // derived
       }
   }
   ```
3. Use the model struct for `generic.Set` (writing state) and `generic.Get` (reading plan/config). When passing to API calls, extract the embedded client fields explicitly: `client.MeshFoo{Metadata: model.Metadata, Spec: model.Spec}`.
4. The same model struct can be shared between resource and data source if the TF schema shape is identical.
5. **Do not** add `json:"-"` fields to client structs — keep client structs clean and API-aligned.

### Data Structure Rules
- **Use pointers & `omitempty`** only for fields that are **actually nullable** in the backend API
- **Non-nullable fields**: Use value types (`string`, `int64`, `bool`) without `omitempty`
- Example:
  ```go
  type Resource struct {
      RequiredField string  `json:"requiredField" tfsdk:"required_field"`           // Non-nullable
      OptionalField *string `json:"optionalField,omitempty" tfsdk:"optional_field"` // Nullable
  }
  ```
