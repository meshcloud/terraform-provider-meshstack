data "meshstack_building_blocks" "all" {
  # All filters are optional. Only active building blocks are returned.

  # workspace_identifier = "my-workspace"
  # project_identifier   = "my-project"
  # platform_identifier  = "my-platform.my-location"
  # name                 = "my-building-block"

  # Filter by the owning building block definition (the definition, not a version):
  # definition_uuid = "00000000-0000-0000-0000-000000000000"

  # Filter by a building block definition version, either by its UUID or by its number
  # (a plain "1" or a "v"-prefixed "v1" are both accepted):
  # version_uuid   = "00000000-0000-0000-0000-000000000000"
  # version_number = "v1"

  # tenant_uuid = "00000000-0000-0000-0000-000000000000"
  # target_kind = "meshTenant" # or "meshWorkspace"
  # status      = "SUCCEEDED"

  # Platform-operator scope (requires the MANAGED_BUILDINGBLOCK_LIST authority): list building
  # blocks created from definitions you own, even when they live in other workspaces.
  # managed_by_definition_uuid      = "00000000-0000-0000-0000-000000000000"
  # managed_by_workspace_identifier = "my-workspace"
}
