# Copilot Instructions for meshStack Terraform Provider

## Repository Overview

This is the official **meshStack Terraform Provider** developed by meshcloud GmbH. It enables Infrastructure as Code (IaC) management of meshStack resources through Terraform by integrating with the meshStack meshObject API.

**Key Information:**
- **Purpose**: Manage meshStack cloud resources (projects, workspaces, tenants, building blocks, etc.) using Terraform
- **API Integration**: Uses the meshStack meshObject API (`/api/meshobjects` endpoints)
- **API Documentation**: https://docs.meshcloud.io/api/index.html#mesh_objects
- **Provider Registry**: https://registry.terraform.io/providers/meshcloud/meshstack/latest/docs
- **License**: MPL-2.0

## Architecture Overview

This provider follows the standard Terraform Provider Plugin Framework pattern:

```
├── main.go                 # Provider entry point
├── internal/provider/      # Provider implementation (resources, data sources)
├── client/                 # API client for meshStack meshObject API
├── docs/                   # Terraform registry documentation
├── examples/               # Example Terraform configurations
└── templates/              # Documentation templates
```

## Directory Structure

### `/internal/provider/`
Contains all Terraform provider implementation:
- **`provider.go`**: Main provider configuration and setup
- **`*_resource.go`**: Resource implementations (Create, Read, Update, Delete)
- **`*_data_source.go`**: Data source implementations (Read-only)
- **Pattern**: Each meshObject type has separate resource and data source files

### `/client/`
meshStack API client implementation:
- **`client.go`**: Core HTTP client with authentication
- **`*.go`**: Individual API client methods for each resource type
- **Authentication**: JWT token-based with automatic refresh
- **API Base Path**: `/api/meshobjects`

### `/docs/`
Terraform registry documentation (auto-generated):
- **`index.md`**: Provider documentation
- **`resources/`**: Resource documentation
- **`data-sources/`**: Data source documentation

### `/examples/`
Example Terraform configurations for testing and documentation

## Development Patterns

### Resource Implementation Pattern
Each resource follows this structure:
```go
type resourceName struct {
    client *client.MeshStackProviderClient
}

// Standard Terraform resource interface methods:
func (r *resourceName) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse)
func (r *resourceName) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse)
func (r *resourceName) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse)
func (r *resourceName) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse)
func (r *resourceName) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse)
```

### API Client Pattern
API clients follow RESTful patterns:
```go
func (c *MeshStackProviderClient) CreateResource(resource ResourceType) (*ResourceType, error)
func (c *MeshStackProviderClient) ReadResource(id string) (*ResourceType, error)
func (c *MeshStackProviderClient) UpdateResource(resource ResourceType) (*ResourceType, error)
func (c *MeshStackProviderClient) DeleteResource(id string) error
```

### Schema Patterns
Resources use consistent schema structures:
- `api_version` - meshObject API version
- `kind` - meshObject type (e.g., "meshProject", "meshWorkspace")
- `metadata` - Object metadata (name, uuid, timestamps, etc.)
- `spec` - Object specification (user-defined configuration)
- `status` - Object status (system-managed state)

### meshEntity Reference Pattern
When referring to other meshEntities (like `project_role_ref` in landingzone resources), implement them with this pattern:
- **User provides**: Only the `name` attribute (required)
- **System sets**: The `kind` attribute automatically (computed with default value)

Example implementation:
```go
schema.SingleNestedAttribute{
    MarkdownDescription: "the meshProject role",
    Required:            true,
    Attributes: map[string]schema.Attribute{
        "name": schema.StringAttribute{
            Required:            true,
            MarkdownDescription: "The identifier of the meshProjectRole",
        },
        "kind": schema.StringAttribute{
            MarkdownDescription: "meshObject type, always `meshProjectRole`.",
            Computed:            true,
            Default:             stringdefault.StaticString("meshProjectRole"),
            Validators: []validator.String{
                stringvalidator.OneOf([]string{"meshProjectRole"}...),
            },
            PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
        },
    },
}
```

This pattern ensures:
- Simplified user experience (only need to specify the name)
- Consistent reference structure across all meshEntity references

## Common Development Tasks

### Adding a New Resource
1. Create `*_resource.go` in `/internal/provider/`
2. Implement the resource interface (Create, Read, Update, Delete, Schema)
3. Add API client methods in `/client/`
4. Register resource in `provider.go`
5. Create example in `/examples/resources/*/`
6. Run `go generate` to update documentation

### meshObject API Integration
- All resources are meshObjects with standard structure
- Use consistent error handling and HTTP status checking
- Implement proper authentication token refresh
- Follow meshStack API conventions for CRUD operations

### Data Structure Guidelines
**Pointer and `omitempty` Usage:**
- Only use pointers (`*type`) and `omitempty` JSON tags for fields that are **actually nullable** in the backend API
- Non-nullable fields should use value types (e.g., `string`, `int64`, `bool`) without `omitempty`
- This ensures proper validation and prevents sending incorrect null values to the API
- Example:
  ```go
  type Resource struct {
      RequiredField string  `json:"requiredField" tfsdk:"required_field"`           // Non-nullable
      OptionalField *string `json:"optionalField,omitempty" tfsdk:"optional_field"` // Nullable in backend
  }
  ```

## Key Dependencies

- **Terraform Plugin Framework**: Latest stable version for provider development
- **Standard Library**: HTTP client, JSON marshaling, context handling
- **meshStack API**: RESTful API following meshObject patterns
