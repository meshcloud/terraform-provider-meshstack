# The restricted key the consumer workspace lists building block definitions with.

resource "meshstack_api_key" "example" {
  metadata = {
    owned_by_workspace = meshstack_workspace.other.metadata.name
  }

  spec = {
    display_name = "ci-key"
    permissions  = ["BUILDINGBLOCKDEFINITION_LIST"]
  }
}
