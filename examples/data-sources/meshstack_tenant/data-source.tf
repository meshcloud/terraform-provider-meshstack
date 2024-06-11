data "meshstack_tenant" "name" {
  metadata = {
    owned_by_project    = "my-project-identifier"
    owned_by_workspace  = "my-workspace-identifier"
    platform_identifier = "my-platform-identifier"
  }
}
