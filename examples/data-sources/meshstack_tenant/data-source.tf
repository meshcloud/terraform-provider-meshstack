data "meshstack_tenant" "name" {
  metadata = {
    owned_by_project    = "my-project"
    owned_by_workspace  = "my-workspace"
    platform_identifier = "my-platform.my-location"
  }
}
