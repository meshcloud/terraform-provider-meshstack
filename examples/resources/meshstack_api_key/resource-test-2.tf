resource "meshstack_api_key" "example" {
  metadata = {
    owned_by_workspace = meshstack_workspace.example.metadata.name
  }

  spec = {
    display_name = "updated-key"
    permissions  = ["LANDINGZONE_LIST", "PROJECT_LIST"]
  }
}
