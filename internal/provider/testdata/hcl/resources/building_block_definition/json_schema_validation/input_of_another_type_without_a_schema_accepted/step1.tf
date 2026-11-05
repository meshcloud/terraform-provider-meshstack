resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "test-ws-[[.suffix]]" }
  spec     = { display_name = "Test", description = "Test" }
  version_spec = {
    draft = true
    inputs = {
      settings = { display_name = "Settings", type = "STRING", assignment_type = "USER_INPUT" }
    }
    implementation = { manual = {} }
  }
}
