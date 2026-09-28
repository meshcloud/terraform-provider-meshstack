data "meshstack_building_block" "example" {
  metadata = {
    uuid = meshstack_building_block.child.metadata.uuid
  }
}
