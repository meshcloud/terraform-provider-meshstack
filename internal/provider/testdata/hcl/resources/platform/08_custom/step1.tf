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
    name               = "TEST-PT-[[.upper_suffix]]"
    owned_by_workspace = meshstack_workspace.test.metadata.name
  }
  spec = {
    display_name     = "Test Platform Type [[.upper_suffix]]"
    default_endpoint = "https://platform.example.com"
    icon             = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciLz4="
  }
}

data "meshstack_platform_type" "example" {
  metadata = {
    name               = meshstack_platform_type.test.metadata.name
    owned_by_workspace = meshstack_platform_type.test.metadata.owned_by_workspace
  }
}

resource "meshstack_platform" "example_custom" {
  metadata = {
    name               = "test-platform-[[.suffix]]"
    owned_by_workspace = data.meshstack_workspace.example.metadata.name
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
        platform_type_ref = data.meshstack_platform_type.example.ref

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
