data "meshstack_platform" "example" {
  metadata = {
    name = "my-platform-identifier"
  }
}

# Use the platform data in another resource
resource "meshstack_tenant" "example" {
  metadata = {
    owned_by_workspace     = "my-workspace"
    owned_by_project       = "my-project"
    platform_identifier   = data.meshstack_platform.example.metadata.name
  }

  spec = {
    landing_zone_identifier = "default-landing-zone"
  }
}