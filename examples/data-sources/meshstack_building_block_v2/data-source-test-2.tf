data "meshstack_building_block_v2" "example" {
  metadata = {
    uuid = meshstack_building_block_v2.sensitive_user_input.metadata.uuid
  }
}
