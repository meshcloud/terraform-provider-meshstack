resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "test-ws-[[.suffix]]" }
  spec = {
    display_name        = "Test"
    description         = "Test"
    target_type         = "TENANT_LEVEL"
    supported_platforms = [{ kind = "meshPlatform", name = "my-azure.eu-de" }]
  }
  version_spec = {
    draft          = true
    implementation = { manual = {} }
  }
}
