resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "test-ws-[[.suffix]]" }
  spec     = { display_name = "Test", description = "Test" }
  version_spec = {
    draft = true
    inputs = {
      tenant = { display_name = "Tenant", type = "STRING", assignment_type = "STATIC", argument = jsonencode("t") }
      region = { display_name = "Region", type = "STRING", assignment_type = "STATIC", argument = jsonencode("r") }
    }
    implementation = { manual = {} }
    outputs        = { tenant = { assignment_type = "[[.assignment_type]]" } }
  }
}
