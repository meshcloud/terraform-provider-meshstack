# A mandatory building block definition that provides a platform tenant ID.
# Used in landing zone tests to enable tenant replication on custom platforms.
resource "meshstack_building_block_definition" "mandatory_bbd" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }

  spec = {
    display_name = "Platform Tenant ID Provider"
    description  = "Provides a platform tenant ID for custom platform tenants"
    target_type  = "TENANT_LEVEL"

    supported_platforms = [{ name = "MY-PLATFORM-TYPE" }]
  }

  version_spec = {
    draft = false

    inputs = {
      tenant_id = {
        display_name    = "Tenant ID"
        type            = "STRING"
        assignment_type = "STATIC"
        argument        = jsonencode("test-tenant-id")
      }
    }

    implementation = {
      manual = {}
    }

    outputs = {
      # Manual building block output: the type is always derived from the matching input and must not be set.
      # assignment_type PLATFORM_TENANT_ID makes this a tracked override.
      tenant_id = {
        display_name    = "Tenant ID"
        assignment_type = "PLATFORM_TENANT_ID"
      }
    }
  }
}
