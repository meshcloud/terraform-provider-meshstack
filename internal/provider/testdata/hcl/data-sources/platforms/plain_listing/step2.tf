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

data "meshstack_platforms" "published" {
  # All filters are optional and map to the platform list endpoint.
  publication_state  = "PUBLISHED"
  owned_by_workspace = meshstack_workspace.test.metadata.name
  # restriction        = "RESTRICTED"
  # owned_by_workspace       = "my-operator-workspace"
  # platform_type_identifier = "my-custom-platform-type"
}

# Select exactly one platform by its identifier (no hardcoded uuid needed) and reuse its
# computed `ref` as `platform_ref` in landing zone / tenant resources.
locals {
  platform = one([
    for platform in data.meshstack_platforms.published.platforms : platform
    if platform.identifier == "my-platform.global"
  ])
}
