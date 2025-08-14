# Workspace Building Block
resource "meshstack_building_block_v2" "example_workspace" {
  spec = {
    building_block_definition_version_ref = {
      uuid = "00000000-0000-0000-0000-000000000000" # Replace with actual definition version UUID
    }

    display_name = "my-building-block"
    target_ref = {
      kind       = "meshWorkspace"
      identifier = "my-workspace-identifier" # Replace with actual workspace identifier
    }

    inputs = {
      name = { value_string = "my-name" }
      size = { value_int = 16 }
    }
  }
}

# Tenant Building Block
data "meshstack_project" "example" {
  metadata = {
    name               = "my-project-identifier"
    owned_by_workspace = "my-workspace-identifier"
  }
}

resource "meshstack_tenant_v4" "example" {
  metadata = {
    owned_by_workspace = data.meshstack_project.example.metadata.owned_by_workspace
    owned_by_project   = data.meshstack_project.example.metadata.name
  }

  spec = {
    platform_identifier     = "my-platform-identifier"
    landing_zone_identifier = "platform-landing-zone-identifier"
  }
}

resource "meshstack_building_block_v2" "example_tenant" {
  spec = {
    building_block_definition_version_ref = {
      uuid = "00000000-0000-0000-0000-000000000001" # Replace with actual definition version UUID
    }

    display_name = "my-tenant-building-block"
    target_ref = {
      kind = "meshTenant"
      uuid = meshstack_tenant_v4.example.metadata.uuid
    }

    inputs = {
      name = { value_string = "my-name" }
      size = { value_int = 16 }
    }
  }
}
