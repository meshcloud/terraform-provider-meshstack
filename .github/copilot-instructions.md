# Copilot Instructions for meshStack Terraform Provider

## Overview

Official Terraform Provider for managing meshStack resources via Infrastructure as Code using the meshObject API (`/api/meshobjects`).
- **API Docs**: https://docs.meshcloud.io/api/index.html#mesh_objects
- **Architecture**: Standard Terraform Provider Plugin Framework

## Key Directories

- **`internal/provider/`**: Provider implementation (`provider.go`, `*_resource.go`, `*_data_source.go`)
- **`internal/provider/acctest/testconfig/`**: HCL config fluent API on `Config.WithFirstBlock` with `ExpressionConsumer`, all builder functions (e.g. `testconfig.Workspace(t)`) - used only in tests
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
        config, addr := testconfig.Integration(t, "_01_github")
        ApplyAndTest(t, resource.TestCase{...})
    })

    t.Run("02_azure_devops", func(t *testing.T) {
        config, addr := testconfig.Integration(t, "_02_azure_devops")
        ApplyAndTest(t, resource.TestCase{...})
    })
}
```

### Config Builder Pattern

Each resource has a builder function in `internal/provider/acctest/testconfig/` that composes `Config` objects from embedded example HCL files. The package name `testconfig` already conveys the context, so builder names drop the `Build` prefix and `Config` suffix — e.g. `testconfig.Workspace(t)`, `testconfig.BBDTerraform(t)`.

`Config` wraps `*hclwrite.File` and is **immutable** — all modifications return a new `Config`. `WithFirstBlock` always clones internally; no need to call `Clone()` (which doesn't exist).

Import alias convention:
```go
import testconfig "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
```

Rules for builder functions:
- Live in `internal/provider/acctest/testconfig/build_<resource>.go` as **public** functions
- Named without `Build` prefix or `Config` suffix: `Workspace`, `Project`, `BBDTerraform`, etc.
- Take `t *testing.T` as **first** parameter; pass `t` to all testconfig calls
- Use **named return values** — first return is always `config testconfig.Config`
- Use **full variable names**, no abbreviations (`workspaceConfig` not `wsConfig`, `workspaceAddr` not `wsAddr`)
- The **resource under test** must be the **receiver** of `.Join()`, with dependency configs as arguments
- Call `config.Join(A, B)` not chained `config.Join(A).Join(B)`
- **Declare all Traversal variables upfront** with `var` before any `.WithFirstBlock` calls that populate them
- **Return inline**: use `return expr.Join(...), addr` — do not assign to a named `config` var and then return
- **Consolidate `WithFirstBlock` calls**: combine all modifiers (including `OwnedByWorkspace`) into a single `WithFirstBlock` call rather than calling it multiple times
- **Prefer `SetAddr` over `SetRawExpr`** for simple resource attribute references: `SetAddr(addr, "metadata", "name")` instead of `SetRawExpr(addr.Join("metadata", "name").String())`
- **`SetRawExpr` format args**: pass Traversal values directly as format args — `SetRawExpr("{uuid = %s}", addr.Join("metadata", "uuid"))`. Do **not** wrap in `fmt.Sprintf`; `SetRawExpr` calls `fmt.Sprintf` internally. Go calls `.String()` on Traversal via `%s` automatically
- **Use raw strings** (backticks) for `SetRawExpr` format strings containing HCL quotes: `` SetRawExpr(`{(%s) = ["12345"]}`, tagAddr.Join("spec", "key")) `` — avoids `\"` escaping. Only use double-quoted Go strings when the expression contains no embedded HCL quotes

```go
// internal/provider/acctest/testconfig/build_workspace.go
func Workspace(t *testing.T) (config testconfig.Config, workspaceAddr testconfig.Traversal) {
    t.Helper()
    name := "test-ws-" + acctest.RandString(8)
    tagConfig, tagAddr := TagDefinition(t, "meshWorkspace")
    return Resource{Name: "workspace"}.Config(t).WithFirstBlock(t,
        testconfig.ExtractIdentifier(&workspaceAddr),
        testconfig.Descend(t, "metadata")(
            testconfig.Descend(t, "name")(testconfig.SetString(name)),
            testconfig.Descend(t, "tags")(testconfig.SetRawExpr(`{(%s) = ["12345"]}`, tagAddr.Join("spec", "key"))),
        ),
    ).Join(tagConfig), workspaceAddr
}
```

**Key style rules for `SetRawExpr` and `SetAddr`:**

Prefer modifiers in this order:
1. `SetString("value")` / `SetCty(ctyVal)` — for literal values
2. `SetAddr(addr, "segment1", "segment2")` — for resource attribute references (e.g. `meshstack_workspace.example.metadata.name`)
3. `SetRawExpr(format, args...)` — last resort for complex HCL expressions (objects, lists, interpolation templates)

`SetRawExpr` rules:
- Format args are passed to `fmt.Sprintf` internally — do **not** wrap in `fmt.Sprintf` yourself
- Use raw backtick strings when the expression contains HCL quotes: `` `{(%s) = ["value"]}` ``
- Compact inline: `{key = value, key2 = value2}` — commas not newlines
- Traversal values as format args: `SetRawExpr("{uuid = %s}", addr.Join("metadata", "uuid"))` — Go calls `.String()` via `%s`

**Usage in tests:**
```go
func TestAccProject(t *testing.T) {
    config, projectAddr, workspaceAddr := testconfig.ProjectAndWorkspace(t)
    ApplyAndTest(t, resource.TestCase{...})
}
```

**Data source test pattern:**
```go
var resourceAddress testconfig.Traversal
projectConfig, projectAddr, _ := testconfig.ProjectAndWorkspace(t)
config := testconfig.DataSource{Name: "project"}.Config(t).
    WithFirstBlock(t,
        testconfig.Descend(t, "metadata", "name")(testconfig.SetAddr(projectAddr, "metadata", "name")),
    ).Join(projectConfig)
```

**Test step updates** — `WithFirstBlock` auto-clones, so just capture the return:
```go
// Step 2: Update display name
Config: config.WithFirstBlock(t,
    testconfig.Descend(t, "spec", "display_name")(testconfig.SetString("Updated Name")),
).String(),
```

**Builder chain (bottom-up):**

Builders that take a `workspaceAddr` parameter require a workspace to already exist in the config. Builders suffixed `*AndWorkspace` are convenience wrappers that create a fresh workspace internally — use these when a test only needs **one** resource and its workspace.

**`*AndWorkspace` pattern:**
- `testconfig.ProjectAndWorkspace(t)` creates a workspace + project in one call — convenient for simple single-resource tests.
- For acceptance tests that test **multiple dependent resources**, build the workspace once and share it via `Config.Join`:

```go
// Acceptance test: shared workspace across multiple resources
workspaceConfig, workspaceAddr := testconfig.Workspace(t)
projectConfig, projectAddr := testconfig.Project(t, workspaceAddr)
platformConfig, platformAddr, platformTypeAddr := testconfig.CustomPlatform(t, workspaceAddr)
landingZoneConfig, landingZoneAddr := testconfig.LandingZone(t, workspaceAddr, platformAddr, platformTypeAddr)

config := landingZoneConfig.Join(platformConfig, projectConfig, workspaceConfig)
```

This avoids creating redundant workspace resources in acceptance tests and is the standard pattern for composing complex test configurations.

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

### Dependency-first Examples

- **Resource example files must contain only the single resource block.** Any additional blocks (data sources, providers, etc.) required for tests go into `test-support_*.tf` files. This keeps examples clean and user-facing.
- **Never use hardcoded identifiers** (like `"my-workspace"` or `"4af5864a-..."`) in example HCL resource files. Always use data source or resource references.
- Use `data.meshstack_workspace.example.metadata.name` for `owned_by_workspace` (not `"my-workspace"`).
- Use `data.meshstack_platform.example.ref` for `platform_ref` (not `{ uuid = "...", kind = "meshPlatform" }`).
- If a plural data source is available, prefer `one(data.meshstack_<plural>.<name>.<items>)` for wiring references.
- Prefer reusable computed outputs (`ref`, `identifier`, `version_latest`, `version_latest_release`) where available.
- When adding or changing a resource/data source, consider whether an additional computed read-only reference output would improve cross-resource wiring.
- **Example files intentionally omit data source declarations.** Resource example `.tf` files reference data sources (e.g. `data.meshstack_workspace.example`) without declaring them. This is by design — the data source blocks live in `test-support_*.tf` files that are loaded alongside the example during tests. Do **not** flag missing data source blocks in example files or generated docs as issues. The generated docs (`docs/`) inherit this pattern from the example files and are also correct as-is.

### Config API (`internal/provider/acctest/testconfig`)

`Config` wraps `*hclwrite.File`. All modifications return a new `Config` — no mutation.

**File layout:**
- `config.go` — `Config`, `Block`, `Expression`, `Descend`, `WalkAttributes`, `Resource`/`DataSource` loaders, internals (`attributeExpression`, `stringTraversable`, `parent`)
- `config_expr.go` — `ExpressionConsumer` type and all public constructors (`SetValue`, `SetString`, `SetAddr`, `SetRawExpr`, `RenameKey`, `ExtractIdentifier`, `OwnedByWorkspace`)
- `config_fake_block.go` — `fakeBlock` (wraps `attributeExpression` to implement `parent` for nested object traversal), `quotedNameMap`, `unquoteAttributeNames`/`requoteAttributeNames`
- `traversal.go` — `Traversal` type and helpers
- `build_*.go` — builder functions, named after the resource they build

**Builder file naming:**
- Use explicit version suffixes when multiple versions of a resource exist (e.g. `build_building_block_v1.go`, `build_building_block_v2.go`, `build_tenant_v3.go`, `build_tenant_v4.go`), even if the Terraform resource name itself omits the version (e.g. `meshstack_buildingblock` → `build_building_block_v1.go`).
- Do not add version suffixes when only one version exists (e.g. `build_workspace.go`, `build_project.go`).

**Types:**
```go
type Config struct { ... }                              // immutable; stores t from Config(t), use WithFirstBlock/Join
type Traversal []string                                 // resource address e.g. ["meshstack_workspace", "my_ws"]
type Expression interface { Get(); Set(); RenameKey() }
type ExpressionConsumer func(t *testing.T, e Expression)
```

**Config methods:**
```go
func NewConfig(t *testing.T, src []byte) Config
func (c Config) WithFirstBlock(mods ...ExpressionConsumer) Config  // clones, returns new; t from Config
func (c Config) Join(others ...Config) Config                       // returns new
func (c Config) String() string                                     // for TestStep.Config
```

**Modifier constructors (no `t` in constructor — receive from invocation):**
```go
testconfig.SetString("value")                           // set string literal
testconfig.SetValue(cty.NumberIntVal(3))                // set any cty.Value
testconfig.SetAddr(addr, "metadata", "name")            // set traversal reference (preferred for resource attributes)
testconfig.SetRawExpr(`{uuid = %s}`, addr)              // set raw HCL expression with fmt.Sprintf format args
testconfig.RenameKey("new_name")                        // rename last block label
testconfig.ExtractAddress(&addr)                        // capture resource Traversal into &addr
```

**Higher-order (no `t` — receive it from ExpressionConsumer):**
```go
testconfig.Descend("spec", "name")(modifier)            // navigate into nested attribute
testconfig.WalkAttributes()(modifier)                    // visit every attribute recursively
testconfig.OwnedByWorkspace(workspaceAddr)               // convenience: sets metadata.owned_by_workspace
```

**`Descend` nesting** — nest only when a parent has **multiple** children. Flatten single-child chains:
```go
// Good: flat — single child
testconfig.Descend("spec", "display_name")(testconfig.SetString("value"))

// Good: nested — multiple children under "spec"
testconfig.Descend("spec")(
    testconfig.Descend("platform_ref")(testconfig.SetAddr(addr, "ref")),
    testconfig.Descend("tags")(testconfig.SetRawExpr(`{(%s) = ["v"]}`, tagAddr)),
)

// Bad: unnecessary nesting with single child
testconfig.Descend("spec")(testconfig.Descend("display_name")(testconfig.SetString("value")))
```

**`Traversal` helpers:**
```go
addr.String()                              // "meshstack_workspace.my_ws"
addr.Join("metadata", "name")              // Traversal{"meshstack_workspace", "my_ws", "metadata", "name"}
addr.Join("metadata", "name").String()     // "meshstack_workspace.my_ws.metadata.name"
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

Data source tests must reference a **resource attribute** (not `depends_on`) so Terraform infers the dependency automatically.

**Always use fluent chaining** — chain `.Config(t).WithFirstBlock(t, ...).Join(...)` in a single expression instead of separate reassignments:

```go
// Good: fluent chaining
config := testconfig.DataSource{Name: "workspace"}.Config(t).WithFirstBlock(t,
    testconfig.Descend(t, "metadata", "name")(testconfig.SetAddr(workspaceAddr, "metadata", "name")),
).Join(workspaceConfig)

// Bad: separate reassignment
config := testconfig.DataSource{Name: "workspace"}.Config(t)
config = config.WithFirstBlock(t, ...)
config = config.Join(workspaceConfig)
```

### Code Review Requirements
- Verify that `CHANGELOG.md` includes entries for all changes (features, fixes, breaking changes)

### CI/CD Best Practices

The GitHub Actions workflows follow the [HashiCorp terraform-provider-scaffolding-framework](https://github.com/hashicorp/terraform-provider-scaffolding-framework) template with minor adjustments.

**Key differences from HashiCorp template:**
- **No Terraform version matrix** — tests run against the single Terraform version installed by `hashicorp/setup-terraform`
- **Separate linting job** — `golangci-lint` runs in its own job (`golangci`) rather than being integrated into the build job

**Action pinning rules:**
- **Always pin actions to full SHA** (40 characters), not version tags
- **Add version comment** after the SHA for readability: `@<sha> # v1.2.3`
- **Use latest stable versions** — check for updates periodically

```yaml
# Good: SHA-pinned with version comment
- uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
- uses: actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v6.4.0
- uses: golangci/golangci-lint-action@1e7e51e771db61008b38414a730f564565cf7c20 # v9.2.0
- uses: hashicorp/setup-terraform@5e8dbf3c6d9deaf4193ca7a8fb23f2ac83bb6c85 # v4.0.0

# Bad: version tag only (mutable, security risk)
- uses: actions/checkout@v4
```

**Standard actions used:**
| Action | Purpose |
|--------|---------|
| `actions/checkout` | Clone repository |
| `actions/setup-go` | Install Go from `go.mod` |
| `golangci/golangci-lint-action` | Lint and format check (provides inline annotations) |
| `hashicorp/setup-terraform` | Install Terraform CLI (for doc generation) |
| `goreleaser/goreleaser-action` | Build and release binaries |
| `crazy-max/ghaction-import-gpg` | Import GPG key for release signing |

**Testing with gotestsum:**
- Tests use [gotestsum](https://github.com/gotestyourself/gotestsum) for better output and JUnit XML generation
- Installed as Go tool dependency in `go.mod` (`tool gotest.tools/gotestsum`)
- Run via `go tool gotestsum` (version managed by Dependabot via gomod ecosystem)
- Uses `-coverpkg=./...` for accurate cross-package coverage measurement
- Coverage posted to PRs via `gh pr comment` (official GitHub CLI, no third-party actions)
- Coverage summary displayed in GitHub job summary via `GITHUB_STEP_SUMMARY`

**To update action versions:**
1. Check latest release on GitHub (e.g., `gh api repos/actions/checkout/releases/latest --jq '.tag_name'`)
2. Get SHA for tag: `gh api repos/actions/checkout/git/refs/tags/v6.0.2 --jq '.object.sha'`
3. Update workflow with new SHA and version comment

### Adding New Resources
1. Create `*_resource.go` in `/internal/provider/` with CRUD + Schema methods
2. Add API client methods in `/client/`
3. Register in `provider.go`
4. Add example in `/examples/resources/*/`
5. Add builder function in `internal/provider/acctest/testconfig/build_<resource>.go` (e.g. `func MyResource(t *testing.T, ...) (config Config, addr Traversal)`)
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

### Go 1.26 `new(value)` for Pointers
Go 1.26 extended the `new` builtin to accept an expression, not just a type. Use `new(value)` to create a pointer to a value in a single expression:

```go
// Before Go 1.26: required a helper function or intermediate variable
func ptrTo[T any](v T) *T { return &v }
s := ptrTo("hello")

// Go 1.26+: use new(value) directly
s := new("hello")           // *string pointing to "hello"
n := new(42)                // *int pointing to 42
n := new(int64(1))          // *int64 pointing to 1
p := new(myStruct{A: "x"})  // *myStruct
```

**Guidelines:**
- **Prefer `new(value)`** over helper functions like `ptr.To(value)` — the helper package was removed
- **Use for inline pointer creation**: struct literals, function arguments, return values
- **Chaining works**: `new(new(new("value")))` creates `***string`
- **Works with expressions**: `new(a + b)`, `new(fmt.Sprintf("sha256:%s", hash))`
