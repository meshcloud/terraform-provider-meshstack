resource "meshstack_tenant" "example" {
  metadata = {
    owned_by_workspace  = "my-workspace"
    owned_by_project    = "my-project"
    platform_identifier = "my-platform.my-location"
  }

  spec = {
    landing_zone_identifier = "my-landing-zone"
  }
}
