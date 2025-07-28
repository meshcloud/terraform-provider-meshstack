data "meshstack_project" "example" {
  metadata = {
    name               = "my-project-identifier"
    owned_by_workspace = "my-workspace-identifier"
  }
}

resource "meshstack_tenant_v4" "example" {
  metadata = {
    owned_by_workspace = data.meshstack_project.example.metadata.owned_by_workspace
    owned_by_project   = data.meshstack_project.example.metadata.name
  }

  spec = {
    platform_identifier     = "my-platform-identifier"
    landing_zone_identifier = "platform-landing-zone-identifier"
  }
}
