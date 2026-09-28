resource "meshstack_building_block_v2" "example_workspace" {
  spec = {
    building_block_definition_version_ref = meshstack_building_block_definition.example.version_latest

    display_name = "my-workspace-building-block"
    target_ref   = meshstack_workspace.example.ref

    inputs = {
      name        = { value_string = "my-name" }
      size        = { value_int = 16 }
      environment = { value_single_select = "dev" }
    }
  }
}
