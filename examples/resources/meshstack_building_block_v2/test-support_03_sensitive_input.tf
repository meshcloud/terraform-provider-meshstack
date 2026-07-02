resource "meshstack_building_block_v2" "sensitive" {
  spec = {
    building_block_definition_version_ref = { uuid = "placeholder" }
    display_name                          = "my-sensitive-building-block"
    target_ref                            = { kind = "meshWorkspace", name = "placeholder" }
    inputs                                = {}
  }
}
