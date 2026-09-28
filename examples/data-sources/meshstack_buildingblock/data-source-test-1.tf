data "meshstack_buildingblock" "example" {
  metadata = {
    uuid = meshstack_buildingblock.my_buildingblock.metadata.uuid
  }
}
