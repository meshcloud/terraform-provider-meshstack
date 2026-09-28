# P_priv: PRIVATE + UNPUBLISHED and not shared with the consumer. It is owned by the same operator
# (so an owner-scoped filter would match it) and reuses P_pub's platform type, which is what makes
# the consumer's restricted key failing to list it a statement about entitlement and nothing else.

resource "meshstack_platform" "priv_custom" {
  metadata = {
    name               = "priv-${var.suffix}"
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name      = "Example Platform"
    description       = "Custom platform using a meshPlatformType"
    endpoint          = "https://custom-platform.example.com"
    documentation_url = "https://docs.example.com"
    location_ref      = { name = "global" }

    availability = {
      restriction       = "PRIVATE"
      publication_state = "UNPUBLISHED"
      # A PRIVATE platform must list exactly its owner (backend validation); it is still not shared
      # with the consumer, so the consumer's restricted key must not see it.
      restricted_to_workspaces = [meshstack_workspace.example.metadata.name]
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
