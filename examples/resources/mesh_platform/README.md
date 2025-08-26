# meshPlatform Resource

This directory contains examples for using the `meshstack_mesh_platform` resource.

## Overview

The `meshstack_mesh_platform` resource allows you to manage meshStack platforms through Terraform. A platform in meshStack represents a cloud platform (like AWS, Azure, GCP, OpenStack, etc.) that can host tenants and workloads.

## Basic Usage

```hcl
resource "meshstack_mesh_platform" "example" {
  metadata = {
    name = "my-platform"
    tags = {
      environment = ["production"]
      team        = ["platform-team"]
    }
  }

  spec = {
    display_name   = "My Platform"
    platform_type  = "OpenStack"
    description    = "Platform for production workloads"
  }
}
```

## Schema

### `metadata`
- `name` (Required) - Platform identifier. Must be unique and follow naming conventions.
- `tags` (Optional) - Key-value pairs for organizing platforms. Values must be lists of strings.
- `created_on` (Computed) - Timestamp when the platform was created
- `deleted_on` (Computed) - Timestamp when the platform was deleted (if applicable)

### `spec`  
- `display_name` (Required) - Human-readable name for the platform
- `platform_type` (Required) - Type of platform (e.g., "OpenStack", "Azure", "AWS", "GCP")
- `description` (Optional) - Description of the platform
- `tags` (Optional) - Additional platform-specific tags

## Import

Existing platforms can be imported using their identifier:

```bash
terraform import meshstack_mesh_platform.example my-platform-id
```

## Notes

- Managing platforms requires an API key with sufficient admin permissions
- The platform name cannot be changed after creation (requires replacement)
- This resource follows the meshStack API conventions for platform objects