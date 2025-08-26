# meshPlatform Resource

This directory contains examples for using the `meshstack_mesh_platform` resource.

## Overview

The `meshstack_mesh_platform` resource allows you to manage meshStack platforms through Terraform. A platform in meshStack represents a cloud platform (like AWS, Azure, GCP, OpenStack, etc.) that can host tenants and workloads.

## AWS Platform Configuration (Strongly Typed)

```hcl
resource "meshstack_mesh_platform" "aws_example" {
  metadata = {
    name = "my-aws-platform"
  }

  spec = {
    display_name   = "Production AWS Platform"
    platform_type  = "AWS"
    description    = "Production AWS platform for compute workloads"
    
    tags = {
      environment = ["production"]
      team        = ["platform-team"]
    }
    
    config = {
      aws = {
        account_id  = "123456789012"
        region      = "us-west-2"
        role_arn    = "arn:aws:iam::123456789012:role/meshPlatformRole"
        external_id = "unique-external-id"
      }
    }
  }
}
```

## Legacy Platform Configuration (Transitioning to Strongly Typed)

Other platform types are currently supported with generic configuration but will be migrated to strongly typed configs:

```hcl
resource "meshstack_mesh_platform" "openstack_example" {
  metadata = {
    name = "my-openstack-platform"
  }

  spec = {
    display_name   = "Production OpenStack Platform" 
    platform_type  = "OpenStack"
    description    = "Platform for production workloads"
    
    tags = {
      environment = ["production"]
      team        = ["platform-team"]
    }
    
    # OpenStack config will be strongly typed in future updates
  }
}
```

## Schema

### `metadata`
- `name` (Required) - Platform identifier. Must be unique and follow naming conventions.
- `created_on` (Computed) - Timestamp when the platform was created
- `deleted_on` (Computed) - Timestamp when the platform was deleted (if applicable)

### `spec`  
- `display_name` (Required) - Human-readable name for the platform
- `platform_type` (Required) - Type of platform (e.g., "OpenStack", "Azure", "AWS", "GCP")
- `description` (Optional) - Description of the platform
- `tags` (Optional) - Key-value pairs for organizing platforms. Values must be lists of strings.
- `config` (Optional) - Platform-specific configuration options

### `config` - Strongly Typed Platform Configurations

#### AWS Configuration (`config.aws`)
- `account_id` (Required) - AWS Account ID
- `region` (Required) - AWS Region
- `endpoint_url` (Optional) - AWS API endpoint URL (defaults to standard AWS endpoints)
- `role_arn` (Optional) - IAM Role ARN for cross-account access
- `external_id` (Optional) - External ID for role assumption (used with role_arn)
- `assume_role_session_name` (Optional) - Session name for role assumption

**Note**: Other platform types (Azure, OpenStack, GCP, etc.) will be migrated to strongly typed configurations in future updates.

## Import

Existing platforms can be imported using their identifier:

```bash
terraform import meshstack_mesh_platform.example my-platform-id
```

## Notes

- Managing platforms requires an API key with sufficient admin permissions
- The platform name cannot be changed after creation (requires replacement)
- This resource follows the meshStack API conventions for platform objects
- AWS platform configuration is now strongly typed - other platform types will follow