# P_pub: the custom platform, but RESTRICTED and published to the consumer workspace as well as its
# operator — which is what makes it consumer-specific entitlement rather than a public listing.

resource "meshstack_platform" "example_custom" {
  metadata = {
    name               = "my-platform-${var.suffix}"
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name      = "Example Platform"
    description       = "Custom platform using a meshPlatformType"
    endpoint          = "https://custom-platform.example.com"
    documentation_url = "https://docs.example.com"
    location_ref      = { name = "global" }

    availability = {
      restriction       = "RESTRICTED"
      publication_state = "PUBLISHED"
      restricted_to_workspaces = [
        meshstack_workspace.example.metadata.name,
        meshstack_workspace.other.metadata.name,
      ]
    }

    quota_definitions = []

    config = {
      custom = {
        platform_type_ref = meshstack_platform_type.example.ref

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
