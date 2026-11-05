resource "meshstack_workspace" "test" {
  metadata = {
    name = "test-ws-[[.suffix]]"
  }
  spec = {
    display_name = "Test Workspace"
  }
}

data "meshstack_workspace" "example" {
  metadata = {
    name = meshstack_workspace.test.metadata.name
  }
}

resource "meshstack_platform_type" "test" {
  metadata = {
    name               = upper("test-pt-[[.suffix]]")
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name     = "Test Platform Type [[.suffix]]"
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
    display_name      = "Test Platform"
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
      }
    }
    contributing_workspaces = []
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

    platform_ref = { uuid = meshstack_platform.test.metadata.uuid }

    platform_properties = {
      custom = {}
    }
  }
}

resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = data.meshstack_workspace.example.metadata.name }
  spec = {
    display_name        = "Test"
    description         = "Test"
    target_type         = "TENANT_LEVEL"
    supported_platforms = [{ name = meshstack_platform_type.test.metadata.name }]
  }
  version_spec = {
    draft          = true
    implementation = { manual = {} }
  }
}
