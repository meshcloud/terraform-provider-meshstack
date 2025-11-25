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
