resource "meshstack_tenant_v4" "example" {
  metadata = {
    owned_by_workspace = one(data.meshstack_projects.example_all_projects_in_workspace.projects).metadata.owned_by_workspace
    owned_by_project   = one(data.meshstack_projects.example_all_projects_in_workspace.projects).metadata.name
  }

  spec = {
    platform_identifier     = data.meshstack_platform.example.identifier
    landing_zone_identifier = data.meshstack_landingzone.example.metadata.name
  }

  # wait_for_completion = true
}
