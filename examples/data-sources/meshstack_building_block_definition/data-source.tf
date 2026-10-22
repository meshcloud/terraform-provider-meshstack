data "meshstack_building_block_definition" "example" {
  spec = {
    display_name = "Example Building Block"
  }
  # Narrow the lookup when several workspaces publish a definition of that name, or look it up by UUID:
  # metadata = {
  #   owned_by_workspace = "my-workspace"
  #   uuid               = "6a1f4e9c-2b7d-4c3a-9e8f-1d5b0c7a3e42"
  # }
}

# The computed version refs drop straight into a building block, and the ref into a definition's dependencies:
#
#   resource "meshstack_building_block" "example" {
#     spec = {
#       building_block_definition_version_ref = data.meshstack_building_block_definition.example.version_latest_release
#       # ...
#     }
#   }
#
#   dependency_refs = [data.meshstack_building_block_definition.example.ref]
