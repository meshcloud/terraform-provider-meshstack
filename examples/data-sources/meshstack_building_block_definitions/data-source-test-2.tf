# The same listing, run as the consumer workspace through the meshstack-other provider alias, to
# show a released definition is visible across workspace boundaries.

data "meshstack_building_block_definitions" "example" {
  provider = meshstack-other

  workspace_identifier = meshstack_building_block_definition.example_03_manual.metadata.owned_by_workspace
}
