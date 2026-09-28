data "meshstack_tenant" "name" {
  metadata = {
    owned_by_workspace  = meshstack_tenant.example.metadata.owned_by_workspace
    owned_by_project    = meshstack_tenant.example.metadata.owned_by_project
    platform_identifier = meshstack_platform.example_custom.identifier
  }
}
