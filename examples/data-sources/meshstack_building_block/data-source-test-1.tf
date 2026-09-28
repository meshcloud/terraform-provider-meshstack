data "meshstack_building_block" "example" {
  metadata = {
    uuid = meshstack_building_block.example_workspace.metadata.uuid
  }
}
