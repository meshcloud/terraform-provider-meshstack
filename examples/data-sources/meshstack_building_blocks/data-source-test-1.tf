# Filtering by the freshly created workspace yields exactly the one block, so indexing
# building_blocks.0 is deterministic; referencing the block's own attribute makes Terraform read the
# data source only after it exists.

data "meshstack_building_blocks" "all" {
  workspace_identifier = meshstack_building_block.example_workspace.metadata.owned_by_workspace
}
