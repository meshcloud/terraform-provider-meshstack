data "meshstack_building_block_definitions" "example" {
  workspace_identifier = meshstack_building_block_definition.example_03_manual.metadata.owned_by_workspace
}
