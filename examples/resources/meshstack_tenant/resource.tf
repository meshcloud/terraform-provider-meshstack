data "meshstack_project" "example" {
  metadata = {
    name               = "my-project-identifier"
    owned_by_workspace = "my-workspace-identifier"
  }
}

resource "meshstack_tenant" "example" {
  metadata = {
    owned_by_workspace  = data.meshstack_project.example.metadata.owned_by_workspace
    owned_by_project    = data.meshstack_project.example.metadata.name
    platform_identifier = "my-platform-identifier"
  }

  spec = {
    landing_zone_identifier = "platform-landing-zone-identifier"
  }
}
