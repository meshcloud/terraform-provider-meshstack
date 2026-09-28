data "meshstack_building_block_v2" "example" {
  metadata = {
    uuid = meshstack_building_block_v2.example_workspace.metadata.uuid
  }
}
