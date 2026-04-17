resource "meshstack_platform" "example_custom" {
  metadata = {
    name               = "my-platform"
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
