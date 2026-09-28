# The restricted key the consumer workspace lists landing zones with.

resource "meshstack_api_key" "example" {
  metadata = {
    owned_by_workspace = meshstack_workspace.other.metadata.name
  }

  spec = {
    display_name = "ci-key"
    permissions  = ["LANDINGZONE_LIST"]
  }
}
