---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "meshstack_workspace Data Source - terraform-provider-meshstack"
subcategory: ""
description: |-
  Read a single workspace by identifier.
---

# meshstack_workspace (Data Source)

Read a single workspace by identifier.

## Example Usage

```terraform
data "meshstack_workspace" "example" {
  metadata = {
    name = "my-workspace-identifier"
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `metadata` (Attributes) (see [below for nested schema](#nestedatt--metadata))

### Read-Only

- `api_version` (String) Workspace API version.
- `kind` (String) meshObject type, always `meshWorkspace`.
- `spec` (Attributes) (see [below for nested schema](#nestedatt--spec))

<a id="nestedatt--metadata"></a>
### Nested Schema for `metadata`

Required:

- `name` (String) Workspace identifier.

Read-Only:

- `created_on` (String) Creation date of the workspace.
- `deleted_on` (String) Deletion date of the workspace.
- `tags` (Map of List of String) Tags of the workspace.


<a id="nestedatt--spec"></a>
### Nested Schema for `spec`

Read-Only:

- `display_name` (String) Display name of the workspace.
- `platform_builder_access_enabled` (Boolean) Whether platform builder access is enabled for the workspace.
