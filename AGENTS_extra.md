# AGENTS_extra.md — review artifact

> **Temporary human-review aid.** This file holds the verbose examples and reference material
> trimmed out of `AGENTS.md` during condensation. Review it and re-promote anything you want
> back into `AGENTS.md`, then delete this file (or keep it as an appendix). Section headings
> mirror `AGENTS.md`.

## Config builder pattern — worked examples & full style rules

Full builder example:

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

Usage in tests:

```go
func TestAccProject(t *testing.T) {
    config, projectAddr, workspaceAddr := testconfig.ProjectAndWorkspace(t)
    ApplyAndTest(t, resource.TestCase{...})
}
```

Test step updates — `WithFirstBlock` auto-clones, so just capture the return:

```go
// Step 2: Update display name
Config: config.WithFirstBlock(t,
    testconfig.Descend(t, "spec", "display_name")(testconfig.SetString("Updated Name")),
).String(),
```

Data source test pattern:

```go
var resourceAddress testconfig.Traversal
projectConfig, projectAddr, _ := testconfig.ProjectAndWorkspace(t)
config := testconfig.DataSource{Name: "project"}.Config(t).
    WithFirstBlock(t,
        testconfig.Descend(t, "metadata", "name")(testconfig.SetAddr(projectAddr, "metadata", "name")),
    ).Join(projectConfig)
```

### Full `SetRawExpr` / `SetAddr` style rules

Prefer modifiers in this order:
1. `SetString("value")` / `SetCty(ctyVal)` — literal values.
2. `SetAddr(addr, "segment1", "segment2")` — resource attribute references
   (e.g. `meshstack_workspace.example.metadata.name`).
3. `SetRawExpr(format, args...)` — last resort for complex HCL (objects, lists, interpolation).

`SetRawExpr` rules:
- Format args go to `fmt.Sprintf` internally — do **not** wrap in `fmt.Sprintf` yourself.
- Use raw backtick strings when the expression contains HCL quotes: `` `{(%s) = ["value"]}` ``;
  only use double-quoted Go strings when there are no embedded HCL quotes.
- Compact inline objects: `{key = value, key2 = value2}` — commas, not newlines.
- Pass `Traversal` values directly as `%s` args: `SetRawExpr("{uuid = %s}", addr.Join("metadata", "uuid"))`
  — Go calls `.String()` on `Traversal` via `%s` automatically.
- Prefer `SetAddr(addr, "metadata", "name")` over `SetRawExpr(addr.Join("metadata", "name").String())`.

`Descend` nesting examples:

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

### Data source tests — fluent vs non-fluent

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

### Multiple-example subtest skeleton

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

## CI/CD — full action reference & gotestsum notes

Standard actions used:

| Action | Purpose |
|--------|---------|
| `actions/checkout` | Clone repository |
| `actions/setup-go` | Install Go from `go.mod` |
| `golangci/golangci-lint-action` | Lint and format check (inline annotations) |
| `hashicorp/setup-terraform` | Install Terraform CLI (for doc generation) |
| `goreleaser/goreleaser-action` | Build and release binaries |
| `crazy-max/ghaction-import-gpg` | Import GPG key for release signing |

Example SHA-pinned refs (versions drift — check before copying):

```yaml
- uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
- uses: actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v6.4.0
- uses: golangci/golangci-lint-action@1e7e51e771db61008b38414a730f564565cf7c20 # v9.2.0
- uses: hashicorp/setup-terraform@5e8dbf3c6d9deaf4193ca7a8fb23f2ac83bb6c85 # v4.0.0
```

To update action versions:
1. Check latest release: `gh api repos/actions/checkout/releases/latest --jq '.tag_name'`.
2. Get SHA for tag: `gh api repos/actions/checkout/git/refs/tags/v6.0.2 --jq '.object.sha'`.
3. Update workflow with new SHA + version comment.

Testing with gotestsum:
- Tests use [gotestsum](https://github.com/gotestyourself/gotestsum) for better output + JUnit XML.
- Installed as a Go tool dependency in `go.mod` (`tool gotest.tools/gotestsum`); run via
  `go tool gotestsum` (version managed by Dependabot, gomod ecosystem).
- Uses `-coverpkg=./...` for cross-package coverage. Coverage posted to PRs via `gh pr comment`
  and summarized in the job via `GITHUB_STEP_SUMMARY`.

## Computed-only output fields — full pattern detail

1. Define a local model struct with the same `tfsdk:`-tagged fields as the client struct, plus
   the extra computed field(s).
2. Add a helper to populate it from the API DTO (deriving the extra field).
3. Use the model struct for `generic.Set` (write state) and `generic.Get` (read plan/config);
   extract embedded client fields explicitly when calling the API.
4. The same struct can be shared between resource and data source if the TF schema shape matches.
5. Do **not** add `json:"-"` fields to client structs — keep them clean and API-aligned.

```go
func myResourceModelFromDto(p *client.MeshFoo) myResourceModel {
    return myResourceModel{
        Metadata: p.Metadata,
        Spec:     p.Spec,
        MyOutput: p.Metadata.Name + "." + p.Spec.SomeName, // derived
    }
}
```

## Go 1.26 `new(value)` — extended notes

- Chaining works: `new(new(new("value")))` creates `***string`.
- Works with expressions: `new(a + b)`, `new(fmt.Sprintf("sha256:%s", hash))`.
- Use for inline pointer creation in struct literals, function arguments, return values.
