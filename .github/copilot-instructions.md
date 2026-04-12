# Copilot Instructions for meshStack Terraform Provider

## Overview

Official Terraform Provider for managing meshStack resources via Infrastructure as Code using the meshObject API (`/api/meshobjects`).
- **API Docs**: https://docs.meshcloud.io/api/index.html#mesh_objects
- **Architecture**: Standard Terraform Provider Plugin Framework

## Key Directories

- **`internal/provider/`**: Provider implementation (`provider.go`, `*_resource.go`, `*_data_source.go`)
- **`client/`**: meshStack API client (JWT auth, RESTful CRUD operations)
- **`docs/`**: Auto-generated Terraform registry documentation
- **`examples/`**: Example Terraform configurations

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

## Development Guidelines

### Code Review Requirements
- Verify that `CHANGELOG.md` includes entries for all changes (features, fixes, breaking changes)

### Adding New Resources
1. Create `*_resource.go` in `/internal/provider/` with CRUD + Schema methods
2. Add API client methods in `/client/`
3. Register in `provider.go`
4. Add example in `/examples/resources/*/`
5. Run `go generate` for docs
6. Update `CHANGELOG.md` with appropriate entry

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
