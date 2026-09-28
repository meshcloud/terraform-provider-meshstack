# The same listing, narrowed by the server-side version_number filter. The test supplies the value
# so one step can assert "v1" matches the block's definition version and another that "v2" — which
# does not exist for it — comes back empty, which is what proves the backend applies the parameter.

variable "version_number" {
  type = string
}

data "meshstack_building_blocks" "all" {
  workspace_identifier = meshstack_building_block.example_workspace.metadata.owned_by_workspace
  version_number       = var.version_number
}
