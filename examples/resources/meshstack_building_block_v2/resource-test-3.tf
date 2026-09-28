resource "meshstack_building_block_v2" "sensitive" {
  spec = {
    building_block_definition_version_ref = meshstack_building_block_definition.sensitive.version_latest
    display_name                          = "my-sensitive-building-block"
    target_ref                            = meshstack_workspace.example.ref
    inputs                                = {}
  }
}
