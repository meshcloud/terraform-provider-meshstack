data "meshstack_building_block_definitions" "example" {
  # Optional server-side filter: limit to one workspace.
  # Additional filtering (e.g. by display name) is done in Terraform expressions.
  workspace_identifier = "my-workspace"
}

# Example: select exactly one BBD by display name pattern.
# (Use can(regex(...)) so non-matching entries are filtered without evaluation errors.)
locals {
  selected_bbd = one([
    for bbd in data.meshstack_building_block_definitions.example.building_block_definitions : bbd
    if can(regex("^Example Building Block.*$", bbd.spec.display_name))
  ])
}
