resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

resource "meshstack_platform_type" "test" {
  metadata = {
    name               = "TEST-PT-[[.upper_suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name     = "Test Platform Type [[.upper_suffix]]"
    default_endpoint = "https://platform.example.com"
    icon             = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciLz4="
  }
}

resource "meshstack_platform" "test" {
  metadata = {
    name               = "test-platform-[[.suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }

  spec = {
    display_name      = "Example Platform"
    description       = "Custom platform using a meshPlatformType"
    endpoint          = "https://custom-platform.example.com"
    documentation_url = "https://docs.example.com"
    location_ref      = { name = "global" }

    availability = {
      restriction              = "PUBLIC"
      publication_state        = "PUBLISHED"
      restricted_to_workspaces = []
    }

    quota_definitions = []

    config = {
      custom = {
        platform_type_ref = meshstack_platform_type.test.ref

        metering = {
          processing = {
            enabled = true
          }
        }
      }
    }

    contributing_workspaces = []
  }
}

resource "meshstack_building_block_definition" "test" {
  metadata = {
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }

  spec = {
    display_name = "Platform Tenant ID Provider"
    description  = "Provides a platform tenant ID for custom platform tenants"
    target_type  = "TENANT_LEVEL"

    supported_platforms = [{ name = meshstack_platform_type.test.metadata.name }]
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
      tenant_id = {
        display_name    = "Tenant ID"
        assignment_type = "PLATFORM_TENANT_ID"
      }
    }
  }
}

resource "meshstack_landingzone" "test" {
  metadata = {
    name               = "test-lz-[[.suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
    tags               = {}
  }

  spec = {
    display_name                  = "My Custom Landing Zone"
    description                   = "A custom landing zone"
    automate_deletion_approval    = false
    automate_deletion_replication = false
    info_link                     = "https://example.com"

    platform_ref = meshstack_platform.test.ref

    platform_properties = {
      custom = {}
    }
    mandatory_building_block_refs = [meshstack_building_block_definition.test.ref]
  }
}

data "meshstack_platforms" "published" {
  publication_state  = "PUBLISHED"
  owned_by_workspace = meshstack_workspace.test.metadata.name
}

[[ example "data-sources/meshstack_landingzones/data-source.tf" ]]
