resource "meshstack_tenant" "example" {
  metadata = {
    owned_by_workspace  = data.meshstack_workspace.example.metadata.name
    owned_by_project    = data.meshstack_project.example.metadata.name
    platform_identifier = data.meshstack_platform.example.identifier
  }

  spec = {
    landing_zone_identifier = data.meshstack_landingzone.example.metadata.name
  }
}
