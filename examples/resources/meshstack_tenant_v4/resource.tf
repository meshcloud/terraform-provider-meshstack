resource "meshstack_tenant_v4" "example" {
  metadata = {
    owned_by_workspace = "my-workspace"
    owned_by_project   = "my-project"
  }

  spec = {
    platform_identifier     = "my-location.my-platform"
    landing_zone_identifier = "my-landing-zone"
  }

  # wait_for_completion = true
}
